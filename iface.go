package serror

// ErrorOrigin is the interface that provides the Origin() method,
// which returns information about the error origin or where
// the error first occurred.
type ErrorOrigin interface {
	Origin() Origin
}

// StructuredError is the interface that provides the StructuredError() method,
// which returns an error string with attached log attributes if any are present.
type StructuredError interface {
	StructuredError() string
}

// StackTracer is the interface that provides the StackTrace() method,
// which returns StackTrace object.
type StackTracer interface {
	StackTrace() StackTrace
}
