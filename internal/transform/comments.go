package transform

import (
	"bytes"
)

type commentTransformer struct{}

func (t *commentTransformer) Transform(source []byte) ([]byte, error) {
	var out bytes.Buffer
	i := 0
	n := len(source)

	for i < n {
		// Check for start of comment
		if i+1 < n && source[i] == '-' && source[i+1] == '-' {
			i += 2
			// Determine if it's a block comment
			if i < n && source[i] == '[' {
				// Count equals signs
				equals := 0
				j := i + 1
				for j < n && source[j] == '=' {
					equals++
					j++
				}
				if j < n && source[j] == '[' {
					// Block comment start: --[=...[
					// Skip to matching closing bracket
					closePattern := make([]byte, equals+2)
					closePattern[0] = ']'
					for k := 0; k < equals; k++ {
						closePattern[1+k] = '='
					}
					closePattern[equals+1] = ']'

					// Search for close pattern
					pos := bytes.Index(source[j+1:], closePattern)
					if pos == -1 {
						// Unclosed block comment, skip to end
						i = n
						break
					}
					i = j + 1 + pos + len(closePattern)
					continue
				}
			}
			// Line comment, skip to end of line
			for i < n && source[i] != '\n' {
				i++
			}
			// Keep the newline
			if i < n && source[i] == '\n' {
				out.WriteByte('\n')
				i++
			}
			continue
		}

		// Handle strings to avoid interpreting -- inside strings
		if source[i] == '"' || source[i] == '\'' {
			quote := source[i]
			out.WriteByte(quote)
			i++
			for i < n {
				if source[i] == '\\' && i+1 < n {
					// Escape sequence, copy both characters
					out.WriteByte(source[i])
					i++
					out.WriteByte(source[i])
					i++
					continue
				}
				if source[i] == quote {
					out.WriteByte(quote)
					i++
					break
				}
				out.WriteByte(source[i])
				i++
			}
			continue
		}

		// Handle long string [[ ... ]]
		if i+1 < n && source[i] == '[' && source[i+1] == '[' {
			out.WriteByte('[')
			out.WriteByte('[')
			i += 2
			depth := 0
			for i < n {
				if i+1 < n && source[i] == ']' && source[i+1] == ']' && depth == 0 {
					out.WriteByte(']')
					out.WriteByte(']')
					i += 2
					break
				}
				if i+1 < n && source[i] == '[' && source[i+1] == '[' {
					depth++
				} else if i+1 < n && source[i] == ']' && source[i+1] == ']' && depth > 0 {
					depth--
				}
				out.WriteByte(source[i])
				i++
			}
			continue
		}

		// Normal character
		out.WriteByte(source[i])
		i++
	}

	return out.Bytes(), nil
}
