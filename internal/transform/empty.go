package transform

import (
	"bytes"
	"strings"
)

type emptyLineTransformer struct{}

func (t *emptyLineTransformer) Transform(source []byte) ([]byte, error) {
	lines := strings.Split(string(source), "\n")
	var out bytes.Buffer
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			out.WriteString(line)
			out.WriteByte('\n')
		}
	}
	// Remove trailing newline if original didn't have it
	result := out.Bytes()
	if len(source) > 0 && source[len(source)-1] != '\n' && len(result) > 0 && result[len(result)-1] == '\n' {
		result = result[:len(result)-1]
	}
	return result, nil
}
