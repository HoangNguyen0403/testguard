package parser

func isTSIdentStart(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_' || c == '$'
}

func isTSIdentPart(c byte) bool {
	return isTSIdentStart(c) || (c >= '0' && c <= '9')
}

func isRegexStarter(prevToken string) bool {
	switch prevToken {
	case "", "(", ",", "=", ":", "[", "!", "&", "|", "?", "{", "}", ";", "return", "typeof":
		return true
	default:
		return false
	}
}

func findMatchingTSDelimiter(src string, openIndex int, openChar, closeChar byte) int {
	if openIndex < 0 || openIndex >= len(src) || src[openIndex] != openChar {
		return -1
	}

	count := 0
	n := len(src)
	prevToken := ""

	for i := openIndex; i < n; {
		c := src[i]

		// Whitespace
		if c == ' ' || c == '\t' || c == '\r' || c == '\n' {
			i++
			continue
		}

		// Single-line comment
		if i+1 < n && c == '/' && src[i+1] == '/' {
			i += 2
			for i < n && src[i] != '\n' {
				i++
			}
			continue
		}

		// Block comment
		if i+1 < n && c == '/' && src[i+1] == '*' {
			i += 2
			for i+1 < n && !(src[i] == '*' && src[i+1] == '/') {
				i++
			}
			i += 2
			continue
		}

		// Strings: ', ", `
		if c == '\'' || c == '"' || c == '`' {
			quote := c
			i++
			for i < n && src[i] != quote {
				if src[i] == '\\' {
					i += 2
					continue
				}
				if (quote == '\'' || quote == '"') && src[i] == '\n' {
					break
				}
				i++
			}
			if i < n && src[i] == quote {
				i++
			}
			prevToken = "string"
			continue
		}

		// Regex literal: /.../
		if c == '/' && isRegexStarter(prevToken) {
			i++ // skip opening /
			inClass := false
			closed := false
			for i < n {
				if src[i] == '\\' {
					i += 2
					continue
				}
				if src[i] == '\n' {
					break
				}
				if src[i] == '[' {
					inClass = true
					i++
					continue
				}
				if src[i] == ']' && inClass {
					inClass = false
					i++
					continue
				}
				if src[i] == '/' && !inClass {
					i++ // skip closing /
					closed = true
					break
				}
				i++
			}
			if closed {
				// skip regex flags
				for i < n && isTSIdentPart(src[i]) {
					i++
				}
				prevToken = "regex"
				continue
			}
		}

		// Delimiters
		if c == openChar {
			count++
			prevToken = string(c)
			i++
			continue
		}
		if c == closeChar {
			count--
			prevToken = string(c)
			if count == 0 {
				return i
			}
			i++
			continue
		}

		// Identifiers / keywords
		if isTSIdentStart(c) {
			start := i
			for i < n && isTSIdentPart(src[i]) {
				i++
			}
			prevToken = src[start:i]
			continue
		}

		// Other punctuation / operators
		prevToken = string(c)
		i++
	}

	return -1
}

// FindMatchingTSParen returns the closing paren index in TS source, correctly skipping JS regex literals, comments, and strings.
func FindMatchingTSParen(src string, openIndex int) int {
	return findMatchingTSDelimiter(src, openIndex, '(', ')')
}

// FindMatchingTSBrace returns the closing brace index in TS source, correctly skipping JS regex literals, comments, and strings.
func FindMatchingTSBrace(src string, openIndex int) int {
	return findMatchingTSDelimiter(src, openIndex, '{', '}')
}
