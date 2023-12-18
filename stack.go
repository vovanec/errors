package serror

import (
	"fmt"
	"runtime"
	"strings"
)

type StackTrace []Origin

func (st StackTrace) String() string {
	var ret []string
	for _, o := range st {
		ret = append(ret, o.String())
	}
	return strings.Join(ret, " ")
}

func getOrigin(n int) Origin {
	if _, file, line, ok := runtime.Caller(n); ok {
		return Origin{
			Line: line,
			File: file,
		}
	}
	return Origin{}
}

type Origin struct {
	Line int
	File string
}

func (o Origin) String() string {
	return fmt.Sprintf("%s:%d", o.File, o.Line)
}

func (o Origin) Empty() bool {
	return o.File == ""
}
