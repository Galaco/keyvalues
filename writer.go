package keyvalues

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// Writer writes KeyValues text, as UTF-8 by default.
type Writer struct {
	w        io.Writer
	escapes  bool
	utf16    bool
	wroteBOM bool
}

// NewWriter returns a Writer that writes to w.
func NewWriter(w io.Writer) *Writer {
	return &Writer{w: w}
}

// UseEscapeSequences makes the Writer escape backslashes, double quotes,
// newlines, tabs and carriage returns in keys and values. Text written this
// way must be read with Reader.UseEscapeSequences.
func (w *Writer) UseEscapeSequences() {
	w.escapes = true
}

// UseUTF16 makes the Writer write little-endian UTF-16 starting with a byte
// order mark, as localization files use. The byte order mark is written once,
// before the first node. Invalid UTF-8 in keys and values is written as
// U+FFFD.
func (w *Writer) UseUTF16() {
	w.utf16 = true
}

var escaper = strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`, "\t", `\t`, "\r", `\r`)

// Write writes kv and everything below it. A block with an empty key and no
// parent, such as the document returned by Reader.Read, is written as a
// document: its children are written at the top level, without braces.
//
// Without escape sequences, a key or value containing a double quote cannot
// be written, and Write returns an error.
func (w *Writer) Write(kv *KeyValue) error {
	out := w.w
	if w.utf16 {
		// Output always ends with a complete rune (a newline, or the byte
		// order mark of an empty document), so once everything is
		// flushed no partial rune is left in the utf16Writer.
		out = &utf16Writer{w: w.w}
	}
	bw := bufio.NewWriter(out)
	writeBOM := w.utf16 && !w.wroteBOM
	if writeBOM {
		bw.WriteRune('\uFEFF')
	}

	nodes := []*KeyValue{kv}
	if kv.block && kv.key == "" && kv.parent == nil {
		nodes = kv.children
	}
	for _, n := range nodes {
		if err := w.writeNode(bw, n, 0); err != nil {
			return err
		}
	}
	if err := bw.Flush(); err != nil {
		return err
	}
	if writeBOM {
		w.wroteBOM = true
	}
	return nil
}

// writeNode writes kv at the given depth. bufio.Writer errors are sticky, so
// they are left for the final Flush to report.
func (w *Writer) writeNode(bw *bufio.Writer, kv *KeyValue, depth int) error {
	indent := strings.Repeat("\t", depth)
	key, err := w.quote(kv.key)
	if err != nil {
		return err
	}
	bw.WriteString(indent)
	bw.WriteString(key)

	if !kv.block {
		value, err := w.quote(kv.value)
		if err != nil {
			return err
		}
		bw.WriteByte('\t')
		bw.WriteString(value)
		writeCondition(bw, kv)
		bw.WriteByte('\n')
		return nil
	}

	writeCondition(bw, kv)
	bw.WriteString("\n" + indent + "{\n")
	for _, child := range kv.children {
		if err := w.writeNode(bw, child, depth+1); err != nil {
			return err
		}
	}
	bw.WriteString(indent + "}\n")
	return nil
}

func (w *Writer) quote(s string) (string, error) {
	if w.escapes {
		return `"` + escaper.Replace(s) + `"`, nil
	}
	if strings.Contains(s, `"`) {
		return "", fmt.Errorf("keyvalues: cannot write %q without escape sequences, as it contains a double quote", s)
	}
	return `"` + s + `"`, nil
}

func writeCondition(bw *bufio.Writer, kv *KeyValue) {
	if kv.condition != "" {
		bw.WriteString(" [" + kv.condition + "]")
	}
}
