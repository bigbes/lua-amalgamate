package transform

type minifyTransformer struct{}

func (t *minifyTransformer) Transform(source []byte) ([]byte, error) {
	// Remove shebang line if present
	source = StripShebang(source)

	// First remove comments
	ct := &commentTransformer{}
	withoutComments, err := ct.Transform(source)
	if err != nil {
		return nil, err
	}

	// Then remove empty lines
	elt := &emptyLineTransformer{}
	withoutEmptyLines, err := elt.Transform(withoutComments)
	if err != nil {
		return nil, err
	}

	// Simple whitespace reduction: collapse multiple spaces to single space
	// This is a basic implementation; a proper minifier would do more
	result := reduceWhitespace(withoutEmptyLines)
	return result, nil
}

func reduceWhitespace(source []byte) []byte {
	var out []byte
	i := 0
	n := len(source)
	inString := false
	stringChar := byte(0)

	for i < n {
		ch := source[i]

		// Track string state
		if !inString && (ch == '"' || ch == '\'') {
			inString = true
			stringChar = ch
			out = append(out, ch)
			i++
			continue
		}

		if inString && ch == stringChar {
			// Check for escape
			if i > 0 && source[i-1] == '\\' {
				// escaped quote, still in string
				out = append(out, ch)
				i++
				continue
			}
			inString = false
			out = append(out, ch)
			i++
			continue
		}

		if !inString && ch == '[' && i+1 < n && source[i+1] == '[' {
			// Long string start
			out = append(out, ch)
			out = append(out, source[i+1])
			i += 2
			depth := 0
			for i < n {
				if i+1 < n && source[i] == ']' && source[i+1] == ']' && depth == 0 {
					out = append(out, ']')
					out = append(out, ']')
					i += 2
					break
				}
				if i+1 < n && source[i] == '[' && source[i+1] == '[' {
					depth++
				} else if i+1 < n && source[i] == ']' && source[i+1] == ']' && depth > 0 {
					depth--
				}
				out = append(out, source[i])
				i++
			}
			continue
		}

		if !inString && (ch == ' ' || ch == '\t') {
			// Collapse multiple whitespace to single space
			out = append(out, ' ')
			// Skip all consecutive whitespace
			for i < n && (source[i] == ' ' || source[i] == '\t') {
				i++
			}
			continue
		}

		// Remove trailing whitespace at end of line
		if !inString && ch == '\n' && len(out) > 0 && (out[len(out)-1] == ' ' || out[len(out)-1] == '\t') {
			// Remove trailing whitespace before newline
			for len(out) > 0 && (out[len(out)-1] == ' ' || out[len(out)-1] == '\t') {
				out = out[:len(out)-1]
			}
		}

		out = append(out, ch)
		i++
	}

	return out
}
