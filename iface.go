package errors

type ErrorOriginer interface {
	Origin() ErrorOrigin
}

type StructuredError interface {
	StructuredError() string
}
