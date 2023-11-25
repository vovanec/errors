# Structured Errors

### Problem Statement:

- Locating and categorizing related logs is often challenging. One solution involves incorporating a trace or request 
id log attribute into a `slog.Logger` instance and transmitting it to the caller. However, the preference is to 
circumvent the necessity of ubiquitously passing a logger instance as a function parameter.
- Logging spam is a common anti-pattern observed in Go programs. When an error is caught, it is logged with additional 
data and then passed up the call stack. Each successive caller logs the error again, leading to redundant logging. 
To address this, errors should be wrapped at the point of origin, with the option to include useful data as log
attributes. The top-level caller can then log the error, revealing all the log attributes collected during the error's
propagation through the call stack.
- The origin of an error (where it was first detected) is often unknown.

### Requirements:

- Compatibility with the native Golang structured logging library slog.
- Traceable logs: Information such as request or trace id should be attachable to a context and passed down to the
callee for logging. Utilizing the native Go `context.Context` for passing logging arguments is ideal.
- Wrappable errors: Errors should be capable of being wrapped, with the ability to accumulate log attributes 
for comprehensive logging at the top level.
- Tracking the error origin: The origin of an error, at a minimum indicating the file and line, should be 
maintained and attached to the error as a log attribute.

### Solution

The `github.com/vovanec/errors` library provides (yet another) errors library with the following features:

- Compatibility with Go stdlib errors, including errors wrapping, unwrapping, and assertion through `As()` and `Is()`.
- Ability to capture and preserve `slog` log attributes, enabling the top-level caller to log them later.
- Capability to capture and preserve the error origin (file and line) as log attributes.
- The `github.com/vovanec/errors/loghelper` helper package offers the following convenience functions:
    - `loghelper.Context`: Adds log attributes as a value to the context.
    - `loghelper.Attr`: Similar to `slog.Any`, but allows extracting log attributes from the context and errors.
    - `logghelper.InitLogging`: Convenience function to initialize default `slog` logger.


### Example code

```golang
package main

import (
  "context"
  "database/sql"
  "fmt"
  "io"
  "log/slog"
  "net/http"
  "net/http/httptest"
  "os"
  "time"

  "github.com/vovanec/serror"
  "github.com/vovanec/serror/loghelper"
)

type AppVersion struct {
  Major int
  Minor int
  Patch int
}

func (v AppVersion) LogValue() slog.Value {
  return slog.GroupValue(
    slog.Int("major", v.Major),
    slog.Int("minor", v.Minor),
    slog.Int("patch", v.Patch),
  )
}

type Application struct {
  Name    string
  Version AppVersion
  Build   string
}

func (a Application) LogValue() slog.Value {
  return slog.GroupValue(
    slog.String("name", a.Name),
    slog.Any("version", a.Version),
    slog.Group("build",
      slog.String("hash", a.Build),
    ),
  )
}

func dbGetUser(ctx context.Context, _ string) error {

  /* This will dump the JSON log similar to below object, with log attributes provided by the caller in a context:
  {
    "time": "2023-11-24T20:39:57.203458-06:00",
    "level": "INFO",
    "msg": "getting user from the database",
    "application": {
      "name": "vovan",
      "version": {
        "major": 1,
        "minor": 7,
        "patch": 2
      },
      "build": {
        "hash": "20b8c3f"
      }
    },
    "request": {
      "id": "b4133182-89a6-11ee-b9d1-0242ac120002"
    },
    "user": {
      "id": "8b50d0c8-015a-497c-b98a-cc69fec2f9ed"
    }
  }
  */
  slog.Info("getting user from the database",
    loghelper.Attr(ctx))

  // code to get user data from the database

  return serror.Wrap(sql.ErrNoRows, "error getting user from database",
    // Log attributes can be attached to the error, they will be logged by the caller.
    loghelper.Attr(
      slog.Group("db",
        slog.String("query", "SELECT first_name, last_name FROM users WHERE id=$1"),
      ),
    ),
  )
}

func handleGetUser(ctx context.Context, userId string) error {
  if err := dbGetUser(ctx, userId); err != nil {
    // Error can be wrapped multiple times and additional log attributes can be attached.
    return serror.Wrap(err, "error in handleGetUser",
      // Log attributes can be attached to the error, they will be logged by the caller.
      loghelper.Attr(
        slog.Any("execution_time", time.Now()),
      ),
    )
  }
  return nil
}

func (a Application) HandleRequest(w http.ResponseWriter, r *http.Request) {

  // Add application information to the context. Application instance can be directly passed to
  // slog.Any since it implements slog.LogValuer interface.
  ctx := loghelper.Context(r.Context(),
    slog.Any("application", a),
  )

  var (
    requestId = r.Header.Get("x-request-id")
    userId    = r.URL.Query().Get("id")
  )

  // Some request-scope data can be extracted from the request and added
  // to the context, so it can be passed to the callee and logged.
  ctx = loghelper.Context(ctx,
    slog.Group("request",
      slog.String("id", requestId),
    ),
    slog.Group("user",
      slog.String("id", userId),
    ),
  )

  // context.Context contains application info, user id and request id they can be logged by the callee with minimal effort.
  if err := handleGetUser(ctx, userId); err != nil {

    /* This will dump the JSON log similar to below object:
    {
      "time": "2023-11-24T20:31:58.408805-06:00",
      "level": "ERROR",
      "msg": "error occurred",
      "application": {             <<- from the context
        "name": "vovan",
        "version": {
          "major": 1,
          "minor": 7,
          "patch": 2
        },
        "build": {
          "hash": "20b8c3f"
        }
      },
      "request": {                <<- from the context
        "id": "b4133182-89a6-11ee-b9d1-0242ac120002"
      },
      "user": {                   <<- from the context
        "id": "8b50d0c8-015a-497c-b98a-cc69fec2f9ed"
      }
      "db": {                     <<- from dbGetUser
        "query": "SELECT first_name, last_name FROM users WHERE id=$1"
      },
      "error": {
        "msg": "error in handleGetUser: error getting user from database: sql: no rows in result set",
        "origin": "/Users/vovan/work/errors/example/main.go:55"
      },
      "execution_time": "2023-11-24T20:31:58.408777-06:00",  <<- handleGetUser
    }
    */
    slog.Error("error occurred",
      loghelper.Attr(ctx, err),
    )

    w.WriteHeader(http.StatusInternalServerError)
    w.Write([]byte(http.StatusText(http.StatusInternalServerError)))
    return
  }

  w.Write(nil)
}

func main() {

  loghelper.InitLogging(
    loghelper.WithLevel(slog.LevelDebug),
    loghelper.WithOutput(os.Stderr),
  )

  var (
    app = Application{
      Name:  "vovan",
      Build: "20b8c3f",
      Version: AppVersion{
        Major: 1,
        Minor: 7,
        Patch: 2,
      },
    }
    req = httptest.NewRequest(
      http.MethodGet,
      "/user?id=8b50d0c8-015a-497c-b98a-cc69fec2f9ed",
      nil)
    w = httptest.NewRecorder()
  )

  req.Header.Set("x-request-id", "b4133182-89a6-11ee-b9d1-0242ac120002")
  app.HandleRequest(w, req)

  res := w.Result()
  defer res.Body.Close()

  if data, err := io.ReadAll(res.Body); err != nil {
    panic(fmt.Sprintf("expected error to be nil got %v", err))
  } else {
    fmt.Println(string(data))
  }
}
```

```json
{
  "time": "2023-11-24T20:42:49.572426-06:00",
  "level": "INFO",
  "msg": "getting user from the database",
  "application": {
    "name": "vovan",
    "version": {
      "major": 1,
      "minor": 7,
      "patch": 2
    },
    "build": {
      "hash": "20b8c3f"
    }
  },
  "request": {
    "id": "b4133182-89a6-11ee-b9d1-0242ac120002"
  },
  "user": {
    "id": "8b50d0c8-015a-497c-b98a-cc69fec2f9ed"
  }
}
```
```json
{
  "time": "2023-11-24T20:42:49.572713-06:00",
  "level": "ERROR",
  "msg": "error occurred",
  "application": {
    "name": "vovan",
    "version": {
      "major": 1,
      "minor": 7,
      "patch": 2
    },
    "build": {
      "hash": "20b8c3f"
    }
  },
  "db": {
    "query": "SELECT first_name, last_name FROM users WHERE id=$1"
  },
  "error": {
    "msg": "error in handleGetUser: error getting user from database: sql: no rows in result set",
    "origin": "/Users/vovan/work/errors/example/main.go:79"
  },
  "execution_time": "2023-11-24T20:42:49.572683-06:00",
  "request": {
    "id": "b4133182-89a6-11ee-b9d1-0242ac120002"
  },
  "user": {
    "id": "8b50d0c8-015a-497c-b98a-cc69fec2f9ed"
  }
}
```
