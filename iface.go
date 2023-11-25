package serror

type ErrorOriginer interface {
	Origin() ErrorOrigin
}

type StructuredError interface {
	StructuredError() string
}
