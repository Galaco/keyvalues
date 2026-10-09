package keyvalues

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"io"
	"strings"
	"unicode/utf8"
)

type tokenKind int

const (
	tokenEOF         tokenKind = iota
	tokenString                // a quoted or bare string
	tokenOpen                  // {
	tokenClose                 // }
	tokenConditional           // [$WIN32], with text holding the part inside the brackets
)

type token struct {
	kind   tokenKind
	text   string
	line   int
	column int
}

// scanner splits KeyValues text into tokens, skipping whitespace and
// comments. It mirrors the engine's tokenizer: a bare string runs until
// whitespace, a quote or a brace, and comments are only recognised where a
// token could start.
type scanner struct {
	r       *bufio.Reader
	escapes bool
	err     error // from looking for a byte order mark, returned by the first scan

	// Position of the next unread byte, both 1-based. Columns count
	// characters rather than bytes.
	line   int
	column int

	peeked    token
	hasPeeked bool
}

var (
	utf8BOM    = []byte{0xEF, 0xBB, 0xBF}
	utf16LEBOM = []byte{0xFF, 0xFE}
	utf16BEBOM = []byte{0xFE, 0xFF}
)

// newScanner returns a scanner reading r. Input starting with a UTF-16 byte
// order mark is converted to UTF-8; anything else is read as UTF-8.
func newScanner(r io.Reader, escapes bool) *scanner {
	s := &scanner{escapes: escapes, line: 1, column: 1}
	br := bufio.NewReader(r)
	head, err := br.Peek(len(utf8BOM))
	if err != nil && err != io.EOF {
		s.err = err
	}
	switch {
	case bytes.HasPrefix(head, utf8BOM):
		_, _ = br.Discard(len(utf8BOM))
	case bytes.HasPrefix(head, utf16LEBOM):
		_, _ = br.Discard(len(utf16LEBOM))
		br = bufio.NewReader(newUTF16Reader(br, binary.LittleEndian))
	case bytes.HasPrefix(head, utf16BEBOM):
		_, _ = br.Discard(len(utf16BEBOM))
		br = bufio.NewReader(newUTF16Reader(br, binary.BigEndian))
	}
	s.r = br
	return s
}

// next returns the next token, which is tokenEOF once input is exhausted.
func (s *scanner) next() (token, error) {
	if s.hasPeeked {
		s.hasPeeked = false
		return s.peeked, nil
	}
	return s.scan()
}

// peek returns the next token without consuming it.
func (s *scanner) peek() (token, error) {
	if !s.hasPeeked {
		t, err := s.scan()
		if err != nil {
			return token{}, err
		}
		s.peeked, s.hasPeeked = t, true
	}
	return s.peeked, nil
}

func (s *scanner) scan() (token, error) {
	if s.err != nil {
		return token{}, s.err
	}
	if err := s.skipSpaceAndComments(); err != nil {
		if err == io.EOF {
			return token{kind: tokenEOF, line: s.line, column: s.column}, nil
		}
		return token{}, err
	}
	t := token{line: s.line, column: s.column}
	b, err := s.readByte()
	if err != nil {
		return token{}, err
	}
	switch b {
	case '{':
		t.kind = tokenOpen
	case '}':
		t.kind = tokenClose
	case '"':
		t.kind = tokenString
		t.text, err = s.scanQuoted(t)
	default:
		t.kind = tokenString
		t.text, err = s.scanBare(b)
		if len(t.text) >= 2 && t.text[0] == '[' && t.text[len(t.text)-1] == ']' {
			t.kind = tokenConditional
			t.text = t.text[1 : len(t.text)-1]
		}
	}
	return t, err
}

func (s *scanner) skipSpaceAndComments() error {
	for {
		b, err := s.peekByte()
		if err != nil {
			return err
		}
		switch {
		case isSpace(b):
			_, _ = s.readByte()
		case b == '/':
			p, err := s.r.Peek(2)
			if err != nil && err != io.EOF {
				return err
			}
			if len(p) < 2 || p[1] != '/' {
				return nil
			}
			if err := s.skipLine(); err != nil {
				return err
			}
		default:
			return nil
		}
	}
}

// skipLine consumes up to and including the next newline.
func (s *scanner) skipLine() error {
	for {
		b, err := s.readByte()
		if err == io.EOF || b == '\n' {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

// scanQuoted reads the rest of a quoted string whose opening quote, at start,
// has been consumed. Quoted strings may span lines.
func (s *scanner) scanQuoted(start token) (string, error) {
	var sb strings.Builder
	for {
		b, err := s.readByte()
		if err == io.EOF {
			return "", syntaxError(start, "unterminated quoted string")
		}
		if err != nil {
			return "", err
		}
		switch {
		case b == '"':
			return sb.String(), nil
		case b == '\\' && s.escapes:
			c, err := s.readByte()
			if err == io.EOF {
				return "", syntaxError(start, "unterminated quoted string")
			}
			if err != nil {
				return "", err
			}
			if r, ok := unescape(c); ok {
				sb.WriteByte(r)
			} else {
				// Keep unknown escapes as written.
				sb.WriteByte('\\')
				sb.WriteByte(c)
			}
		default:
			sb.WriteByte(b)
		}
	}
}

// scanBare reads the rest of an unquoted string starting with first.
func (s *scanner) scanBare(first byte) (string, error) {
	var sb strings.Builder
	sb.WriteByte(first)
	for {
		b, err := s.peekByte()
		if err == io.EOF {
			return sb.String(), nil
		}
		if err != nil {
			return "", err
		}
		if isSpace(b) || b == '"' || b == '{' || b == '}' {
			return sb.String(), nil
		}
		_, _ = s.readByte()
		sb.WriteByte(b)
	}
}

func (s *scanner) readByte() (byte, error) {
	b, err := s.r.ReadByte()
	if err != nil {
		return 0, err
	}
	if b == '\n' {
		s.line++
		s.column = 1
	} else if utf8.RuneStart(b) {
		s.column++
	}
	return b, nil
}

func (s *scanner) peekByte() (byte, error) {
	p, err := s.r.Peek(1)
	if err != nil {
		return 0, err
	}
	return p[0], nil
}

func isSpace(b byte) bool {
	switch b {
	case ' ', '\t', '\n', '\r', '\v', '\f':
		return true
	}
	return false
}

// unescape returns the byte an escape sequence ending in c stands for. These
// are the sequences the engine recognises.
func unescape(c byte) (byte, bool) {
	switch c {
	case 'n':
		return '\n', true
	case 't':
		return '\t', true
	case 'v':
		return '\v', true
	case 'b':
		return '\b', true
	case 'r':
		return '\r', true
	case 'f':
		return '\f', true
	case 'a':
		return '\a', true
	case '\\', '?', '\'', '"':
		return c, true
	}
	return 0, false
}
