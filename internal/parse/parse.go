package parse

type RequireInfo struct {
	Name   string // the literal string argument, e.g. "foo.bar"
	Line   int    // source line number
	Static bool   // true if argument is a string literal
}

type Parser interface {
	Parse(source []byte, filename string) ([]RequireInfo, error)
}

func New() Parser {
	return &gopherParser{}
}
