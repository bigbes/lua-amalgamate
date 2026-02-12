package transform

import (
	"bytes"
)

type shebangTransformer struct{}

func (t *shebangTransformer) Transform(source []byte) ([]byte, error) {
	return StripShebang(source), nil
}

func StripShebang(source []byte) []byte {
	// Simple version used by minify transformer
	var out bytes.Buffer
	i := 0
	n := len(source)
	// Skip leading whitespace
	for i < n && (source[i] == ' ' || source[i] == '\t' || source[i] == '\r' || source[i] == '\n') {
		i++
	}
	if i+1 < n && source[i] == '#' && source[i+1] == '!' {
		// Skip shebang line
		for i < n && source[i] != '\n' {
			i++
		}
		if i < n && source[i] == '\n' {
			i++
		}
	}
	out.Write(source[i:])
	return out.Bytes()
}
