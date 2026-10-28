package parser

// FindMatchingParen returns the index of the matching closing parenthesis for the
// opening parenthesis at openIndex, correctly skipping string literals and comments.
func FindMatchingParen(src string, openIndex int) int {
	if openIndex < 0 || openIndex >= len(src) || src[openIndex] != '(' {
		return -1
	}

	parenCount := 0
	n := len(src)

	for i := openIndex; i < n; i++ {
		// Single-line comment
		if i+1 < n && src[i] == '/' && src[i+1] == '/' {
			i += 2
			for i < n && src[i] != '\n' {
				i++
			}
			continue
		}

		// Multi-line block comment
		if i+1 < n && src[i] == '/' && src[i+1] == '*' {
			i += 2
			for i+1 < n && !(src[i] == '*' && src[i+1] == '/') {
				i++
			}
			i++
			continue
		}

		// Raw string literal: r'...' or r"..."
		if (src[i] == 'r' || src[i] == 'R') && i+1 < n && (src[i+1] == '\'' || src[i+1] == '"') {
			quote := src[i+1]
			if i+3 < n && src[i+2] == quote && src[i+3] == quote {
				i += 4
				for i+2 < n && !(src[i] == quote && src[i+1] == quote && src[i+2] == quote) {
					i++
				}
				i += 2
				continue
			}
			i += 2
			for i < n && src[i] != quote && src[i] != '\n' {
				i++
			}
			continue
		}

		// Triple-quoted multi-line string literal: """...""" or '''...'''
		if (src[i] == '\'' || src[i] == '"') && i+2 < n && src[i+1] == src[i] && src[i+2] == src[i] {
			quote := src[i]
			i += 3
			for i+2 < n {
				if src[i] == '\\' {
					i += 2
					continue
				}
				if src[i] == quote && src[i+1] == quote && src[i+2] == quote {
					i += 2
					break
				}
				i++
			}
			continue
		}

		// Standard quoted string literal: "..." or '...' or `...`
		if src[i] == '\'' || src[i] == '"' || src[i] == '`' {
			quote := src[i]
			i++
			for i < n && src[i] != quote {
				if src[i] == '\\' {
					i++
				} else if src[i] == '\n' && quote != '`' {
					break
				}
				i++
			}
			continue
		}

		if src[i] == '(' {
			parenCount++
		} else if src[i] == ')' {
			parenCount--
			if parenCount == 0 {
				return i
			}
		}
	}

	return -1
}
