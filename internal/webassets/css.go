package webassets

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/tdewolff/parse/v2"
	"github.com/tdewolff/parse/v2/css"
)

// rewriteCSS uses CSS tokens so comments, strings and data URLs are not mistaken for dependencies.
func (s *Snapshot) rewriteCSS(name string, data []byte) ([]byte, error) {
	input := parse.NewInputBytes(data)
	defer input.Restore()
	lexer := css.NewLexer(input)
	var out bytes.Buffer
	for {
		kind, token := lexer.Next()
		if kind == css.ErrorToken {
			if err := lexer.Err(); !errors.Is(err, io.EOF) {
				return nil, fmt.Errorf("lex CSS: %w", err)
			}
			return out.Bytes(), nil
		}
		if kind == css.FunctionToken {
			function := strings.ToLower(unescapeCSS(strings.TrimSuffix(string(token), "(")))
			switch function {
			case "url":
				return nil, fmt.Errorf("escaped CSS url function is not supported; use literal url")
			case "image-set", "-webkit-image-set", "src":
				return nil, fmt.Errorf("CSS image dependencies must use literal url")
			}
		}
		if kind == css.BadURLToken || kind == css.BadStringToken {
			return nil, fmt.Errorf("malformed CSS URL or string")
		}
		if kind == css.AtKeywordToken && strings.EqualFold(unescapeCSS(string(token[1:])), "import") {
			return nil, fmt.Errorf("CSS @import is not supported; concatenate first")
		}
		if kind == css.URLToken {
			open := bytes.IndexByte(token, '(')
			if open < 0 || token[len(token)-1] != ')' {
				return nil, fmt.Errorf("incomplete CSS URL")
			}
			raw := strings.TrimSpace(string(token[open+1 : len(token)-1]))
			if len(raw) >= 2 && (raw[0] == '\'' || raw[0] == '"') {
				raw = raw[1 : len(raw)-1]
			}
			decoded := unescapeCSS(raw)
			parsed, err := url.Parse(decoded)
			if err != nil {
				return nil, fmt.Errorf("invalid CSS URL: %w", err)
			}
			if strings.HasPrefix(decoded, "#") || parsed.IsAbs() || parsed.Host != "" {
				out.Write(token)
				continue
			}
			rewritten, err := s.cssURL(name, decoded)
			if err != nil {
				return nil, err
			}
			// Generated names are URL-escaped; quoting cannot inject CSS syntax.
			out.WriteString("url(\"")
			out.WriteString(rewritten)
			out.WriteString("\")")
		} else {
			out.Write(token)
		}
		if out.Len() > maxFileBytes {
			return nil, fmt.Errorf("expanded CSS exceeds size limit")
		}
	}
}

// unescapeCSS decodes CSS escapes, including hex and escaped line breaks, for dependency lookup.
func unescapeCSS(s string) string {
	var out strings.Builder
	for i := 0; i < len(s); {
		if s[i] != '\\' {
			out.WriteByte(s[i])
			i++
			continue
		}
		i++
		if i == len(s) {
			break
		}
		start := i
		for i < len(s) && i-start < 6 && strings.ContainsRune("0123456789abcdefABCDEF", rune(s[i])) {
			i++
		}
		if i > start {
			code, err := strconv.ParseInt(s[start:i], 16, 32)
			if err != nil || code == 0 || code > utf8.MaxRune || code >= 0xD800 && code <= 0xDFFF {
				code = utf8.RuneError
			}
			out.WriteRune(rune(code))
			if i < len(s) && strings.ContainsRune(" \t\r\n\f", rune(s[i])) {
				if s[i] == '\r' && i+1 < len(s) && s[i+1] == '\n' {
					i++
				}
				i++
			}
		} else if strings.ContainsRune("\r\n\f", rune(s[i])) {
			if s[i] == '\r' && i+1 < len(s) && s[i+1] == '\n' {
				i++
			}
			i++
		} else {
			out.WriteByte(s[i])
			i++
		}
	}
	return out.String()
}
