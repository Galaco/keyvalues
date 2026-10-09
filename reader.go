package keyvalues

import (
	"errors"
	"fmt"
	"io"
)

// SyntaxError describes malformed KeyValues text.
type SyntaxError struct {
	Line   int // 1-based
	Column int // 1-based, in characters
	Msg    string
}

func (e *SyntaxError) Error() string {
	return fmt.Sprintf("keyvalues: syntax error at line %d, column %d: %s", e.Line, e.Column, e.Msg)
}

func syntaxError(at token, format string, args ...interface{}) *SyntaxError {
	return &SyntaxError{Line: at.line, Column: at.column, Msg: fmt.Sprintf(format, args...)}
}

// Reader parses KeyValues text. It handles any KeyValues based format, such as
// gameinfo.txt, .vmt and .vmf files.
type Reader struct {
	r       io.Reader
	escapes bool
}

// NewReader returns a Reader that parses r.
func NewReader(r io.Reader) *Reader {
	return &Reader{r: r}
}

// UseEscapeSequences makes the Reader decode escape sequences such as \n and
// \" in quoted strings. By default backslashes are read literally, as the
// engine does for most files.
func (r *Reader) UseEscapeSequences() {
	r.escapes = true
}

// Read parses all of the input. It returns a document root: a block with an
// empty key whose children are the top-level nodes of the input.
//
// Input starting with a UTF-16 byte order mark, as localization files do, is
// read as UTF-16. Anything else is read as UTF-8, and a UTF-8 byte order mark
// is skipped. Unpaired UTF-16 surrogates are read as U+FFFD.
//
// Malformed input returns a *SyntaxError. Errors from the underlying reader
// are returned as is.
func (r *Reader) Read() (*KeyValue, error) {
	s := newScanner(r.r, r.escapes)
	doc, err := parse(s)
	if errors.Is(err, errTruncatedUTF16) {
		return nil, &SyntaxError{Line: s.line, Column: s.column, Msg: errTruncatedUTF16.Error()}
	}
	return doc, err
}

func parse(s *scanner) (*KeyValue, error) {
	root := NewBlock("")

	type openBlock struct {
		node  *KeyValue
		start token
	}
	var open []openBlock
	current := root

	for {
		t, err := s.next()
		if err != nil {
			return nil, err
		}
		switch t.kind {
		case tokenEOF:
			if len(open) > 0 {
				b := open[len(open)-1]
				return nil, syntaxError(b.start, "block %q is never closed", b.node.key)
			}
			return root, nil
		case tokenClose:
			if len(open) == 0 {
				return nil, syntaxError(t, "unexpected '}' with no open block")
			}
			closed := current
			open = open[:len(open)-1]
			current = closed.parent
			if err := readTrailingCondition(s, closed); err != nil {
				return nil, err
			}
		case tokenOpen:
			return nil, syntaxError(t, "unexpected '{', expected a key")
		case tokenConditional:
			return nil, syntaxError(t, "unexpected conditional [%s], expected a key", t.text)
		case tokenString:
			node, err := readPair(s, t)
			if err != nil {
				return nil, err
			}
			current.appendChild(node)
			if node.block {
				open = append(open, openBlock{node: node, start: t})
				current = node
			}
		}
	}
}

// readPair reads what follows key: a value or the '{' opening a block,
// optionally preceded by a conditional. For a leaf it also reads any
// conditional after the value.
func readPair(s *scanner, key token) (*KeyValue, error) {
	node := &KeyValue{key: key.text}
	t, err := s.next()
	if err != nil {
		return nil, err
	}
	if t.kind == tokenConditional {
		node.condition = t.text
		if t, err = s.next(); err != nil {
			return nil, err
		}
	}
	switch t.kind {
	case tokenEOF:
		return nil, syntaxError(key, "key %q has no value", key.text)
	case tokenClose:
		return nil, syntaxError(t, "unexpected '}', expected a value for key %q", key.text)
	case tokenConditional:
		return nil, syntaxError(t, "unexpected conditional [%s], expected a value for key %q", t.text, key.text)
	case tokenOpen:
		node.block = true
		return node, nil
	}
	node.value = t.text
	if err := readTrailingCondition(s, node); err != nil {
		return nil, err
	}
	return node, nil
}

// readTrailingCondition attaches a conditional following node's value or
// closing brace, if there is one.
func readTrailingCondition(s *scanner, node *KeyValue) error {
	t, err := s.peek()
	if err != nil || t.kind != tokenConditional {
		return err
	}
	if node.condition != "" {
		return syntaxError(t, "key %q has more than one conditional", node.key)
	}
	node.condition = t.text
	_, err = s.next()
	return err
}
