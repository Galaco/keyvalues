package keyvalues

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"strings"
	"testing"
	"testing/iotest"
	"unicode/utf16"
	"unicode/utf8"
)

var byteOrders = map[string]binary.ByteOrder{
	"little-endian": binary.LittleEndian,
	"big-endian":    binary.BigEndian,
}

func readBytes(t *testing.T, b []byte) (*KeyValue, error) {
	t.Helper()
	return NewReader(bytes.NewReader(b)).Read()
}

// TestRead_UTF16 checks UTF-16 input parses exactly as the same text in UTF-8.
func TestRead_UTF16(t *testing.T) {
	for _, tc := range readCases {
		if !utf8.ValidString(tc.input) || strings.HasPrefix(tc.input, string(utf8BOM)) {
			continue
		}
		for name, order := range byteOrders {
			t.Run(tc.name+"/"+name, func(t *testing.T) {
				doc, err := readBytes(t, encodeUTF16(tc.input, order))
				if err != nil {
					t.Fatal(err)
				}
				if got := dump(doc); got != tc.want {
					t.Errorf("got  %s\nwant %s", got, tc.want)
				}
				checkParents(t, doc)
			})
		}
	}
}

func TestRead_UTF16NonASCII(t *testing.T) {
	// In UTF-16 "Ż" contains the byte for '{' and "Ģ" the byte for '"', so
	// they would break a parser reading the raw bytes.
	const input = `"Ż" { "Ģ" "日本語 😀" }`
	const want = `"Ż"{"Ģ"="日本語 😀"}`
	for name, order := range byteOrders {
		t.Run(name, func(t *testing.T) {
			doc, err := readBytes(t, encodeUTF16(input, order))
			if err != nil {
				t.Fatal(err)
			}
			if got := dump(doc); got != want {
				t.Errorf("got  %s\nwant %s", got, want)
			}
		})
	}
}

func TestRead_Localization(t *testing.T) {
	doc := readFile(t, "lang_english.txt", true)
	lang := find(t, doc, "lang")
	if got := asString(t, find(t, lang, "Language")); got != "English" {
		t.Errorf("Language = %q", got)
	}
	tokens := find(t, lang, "Tokens")
	want := map[string]string{
		"Menu_Title":          "Café ŻÓŁĆ Ģirts",
		"[english]Menu_Title": "Café",
		"Menu_Quote":          "Say \"hi\"\nNew line",
		"Menu_Emoji":          "Game over 😀",
		"Menu_Japanese":       "ゲーム",
		"Menu_Console":        "Console",
	}
	for key, value := range want {
		if got := asString(t, find(t, tokens, key)); got != value {
			t.Errorf("%s = %q, want %q", key, got, value)
		}
	}
	if got := find(t, tokens, "Menu_Console").Condition(); got != "$WIN32" {
		t.Errorf("Menu_Console condition = %q", got)
	}
	checkParents(t, doc)
}

func TestRead_UTF16ByteOrderMarkOnly(t *testing.T) {
	for _, input := range []string{"\xff\xfe", "\xfe\xff"} {
		doc, err := readBytes(t, []byte(input))
		if err != nil || dump(doc) != "" {
			t.Errorf("Read(%q) = %v, %v; want an empty document", input, doc, err)
		}
	}
}

func TestRead_UTF16UnpairedSurrogates(t *testing.T) {
	units := func(s string) []uint16 { return utf16.Encode([]rune(s)) }
	join := func(parts ...[]uint16) []uint16 {
		var all []uint16
		for _, p := range parts {
			all = append(all, p...)
		}
		return all
	}
	tests := []struct {
		name  string
		units []uint16
		want  string
	}{
		{"high surrogate before a character", join(units(`"a" "`), []uint16{0xD800}, units(`x"`)), `"a"="�x"`},
		{"high surrogate before a quote", join(units(`"a" "`), []uint16{0xD83D}, units(`"`)), `"a"="�"`},
		{"low surrogate alone", join(units(`"a" "`), []uint16{0xDC00}, units(`"`)), `"a"="�"`},
		{"pair in the wrong order", join(units(`"a" "`), []uint16{0xDE00, 0xD83D}, units(`"`)), `"a"="��"`},
		{"two high surrogates then a low", join(units(`"a" "`), []uint16{0xD83D, 0xD83D, 0xDE00}, units(`"`)), `"a"="�😀"`},
		{"high surrogate at EOF", join(units(`"a" `), []uint16{0xD800}), `"a"="�"`},
	}
	for _, tc := range tests {
		for name, order := range byteOrders {
			t.Run(tc.name+"/"+name, func(t *testing.T) {
				doc, err := readBytes(t, unitsToBytes(order, tc.units))
				if err != nil {
					t.Fatal(err)
				}
				if got := dump(doc); got != tc.want {
					t.Errorf("got  %s\nwant %s", got, tc.want)
				}
			})
		}
	}
}

func TestRead_UTF16Truncated(t *testing.T) {
	tests := []struct {
		name   string
		input  []byte
		line   int
		column int
	}{
		{"after a value", append(encodeUTF16(`"a" "b"`, binary.LittleEndian), 'x'), 1, 8},
		{"in a quoted string", append(encodeUTF16(`"a" "b`, binary.LittleEndian), 'x'), 1, 7},
		{"on a later line", append(encodeUTF16("\"a\"\n\"b\"", binary.BigEndian), 'x'), 2, 4},
		{"after a high surrogate", append(unitsToBytes(binary.LittleEndian, append(utf16.Encode([]rune(`"a" `)), 0xD800)), 'x'), 1, 6},
		{"only a byte order mark and a byte", []byte{0xFF, 0xFE, 'x'}, 1, 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := readBytes(t, tc.input)
			var syntaxErr *SyntaxError
			if !errors.As(err, &syntaxErr) {
				t.Fatalf("got %v, %v; want a *SyntaxError", doc, err)
			}
			want := SyntaxError{Line: tc.line, Column: tc.column, Msg: "input ends partway through a UTF-16 character"}
			if *syntaxErr != want {
				t.Errorf("got  %+v\nwant %+v", *syntaxErr, want)
			}
		})
	}
}

func TestRead_UTF16ReaderErrors(t *testing.T) {
	errBoom := errors.New("boom")
	tests := []struct {
		name  string
		input []byte
	}{
		{"after byte order mark", []byte{0xFF, 0xFE}},
		{"in a block", encodeUTF16(`"a" { "b" "c`, binary.LittleEndian)},
		{"in a code unit", append(encodeUTF16(`"a" "b`, binary.LittleEndian), 'x')},
		{"after a high surrogate", unitsToBytes(binary.LittleEndian, append(utf16.Encode([]rune(`"a" "`)), 0xD83D))},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := io.MultiReader(bytes.NewReader(tc.input), iotest.ErrReader(errBoom))
			if _, err := NewReader(r).Read(); !errors.Is(err, errBoom) {
				t.Errorf("got %v, want %v", err, errBoom)
			}
		})
	}
}

// failOnceReader returns err from its first Read, then reads from r.
type failOnceReader struct {
	r   io.Reader
	err error
}

func (f *failOnceReader) Read(p []byte) (int, error) {
	if err := f.err; err != nil {
		f.err = nil
		return 0, err
	}
	return f.r.Read(p)
}

func TestRead_ErrorWhileCheckingByteOrderMark(t *testing.T) {
	errBoom := errors.New("boom")
	r := &failOnceReader{r: strings.NewReader(`"a" "b"`), err: errBoom}
	if _, err := NewReader(r).Read(); !errors.Is(err, errBoom) {
		t.Errorf("got %v, want %v", err, errBoom)
	}
}

// TestRead_UTF16ReaderBehaviour checks parsing doesn't depend on how the
// underlying reader splits its input, including partway through a code unit.
func TestRead_UTF16ReaderBehaviour(t *testing.T) {
	input := []byte(fixtures(t)["testdata/lang_english.txt"])
	want := dump(readFile(t, "lang_english.txt", false))
	readers := map[string]func(io.Reader) io.Reader{
		"one byte":   iotest.OneByteReader,
		"half":       iotest.HalfReader,
		"data + EOF": iotest.DataErrReader,
	}
	for name, wrap := range readers {
		t.Run(name, func(t *testing.T) {
			doc, err := NewReader(wrap(bytes.NewReader(input))).Read()
			if err != nil {
				t.Fatal(err)
			}
			if got := dump(doc); got != want {
				t.Errorf("got  %s\nwant %s", got, want)
			}
		})
	}
}

func TestRead_UTF16LongValue(t *testing.T) {
	// Long enough that converted characters are split across reads.
	value := strings.Repeat("aゲーム😀", 3000)
	doc, err := readBytes(t, encodeUTF16(`"key" "`+value+`"`, binary.LittleEndian))
	if err != nil {
		t.Fatal(err)
	}
	if got := asString(t, find(t, doc, "key")); got != value {
		t.Errorf("value changed: got %d bytes, want %d", len(got), len(value))
	}
}

func TestUTF16Reader(t *testing.T) {
	const text = "a ゲーム 😀 Ż\x00 \ufffd"
	for name, order := range byteOrders {
		t.Run(name, func(t *testing.T) {
			input := encodeUTF16(text, order)[2:] // the reader starts after the byte order mark
			r := newUTF16Reader(bufio.NewReader(bytes.NewReader(input)), order)
			// TestReader reads with buffers of several sizes, including
			// ones smaller than a single character.
			if err := iotest.TestReader(r, []byte(text)); err != nil {
				t.Error(err)
			}
		})
	}
}

func TestRead_ColumnsCountCharacters(t *testing.T) {
	const input = `"é" "日本" }`
	for name, b := range map[string][]byte{
		"UTF-8":  []byte(input),
		"UTF-16": encodeUTF16(input, binary.LittleEndian),
	} {
		_, err := readBytes(t, b)
		var syntaxErr *SyntaxError
		if !errors.As(err, &syntaxErr) || syntaxErr.Line != 1 || syntaxErr.Column != 10 {
			t.Errorf("%s: got %v, want an error at line 1, column 10", name, err)
		}
	}
}

func TestWriter_UTF16(t *testing.T) {
	kv := NewBlock("Ż", NewKeyValue("emoji", "😀"), NewKeyValue("Ģ", "ゲーム"))
	var buf bytes.Buffer
	w := NewWriter(&buf)
	w.UseUTF16()
	if err := w.Write(kv); err != nil {
		t.Fatal(err)
	}
	want := encodeUTF16("\"Ż\"\n{\n\t\"emoji\"\t\"😀\"\n\t\"Ģ\"\t\"ゲーム\"\n}\n", binary.LittleEndian)
	if !bytes.Equal(buf.Bytes(), want) {
		t.Errorf("got  % x\nwant % x", buf.Bytes(), want)
	}

	doc, err := readBytes(t, buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if dump(doc) != dump(kv) {
		t.Errorf("read back %s, want %s", dump(doc), dump(kv))
	}
}

func TestWriter_UTF16ByteOrderMarkOnce(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)
	w.UseUTF16()

	// A failed write writes nothing, so the byte order mark is still to come.
	if err := w.Write(NewKeyValue("bad", `"`)); err == nil {
		t.Fatal("got no error writing a quote without escape sequences")
	}
	for _, kv := range []*KeyValue{NewKeyValue("a", "1"), NewKeyValue("b", "2")} {
		if err := w.Write(kv); err != nil {
			t.Fatal(err)
		}
	}
	want := encodeUTF16("\"a\"\t\"1\"\n\"b\"\t\"2\"\n", binary.LittleEndian)
	if !bytes.Equal(buf.Bytes(), want) {
		t.Errorf("got  % x\nwant % x", buf.Bytes(), want)
	}
}

func TestWriter_UTF16EmptyDocument(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)
	w.UseUTF16()
	if err := w.Write(read(t, "")); err != nil {
		t.Fatal(err)
	}
	if got := buf.Bytes(); !bytes.Equal(got, []byte{0xFF, 0xFE}) {
		t.Errorf("got % x, want just a byte order mark", got)
	}
}

func TestWriter_UTF16InvalidUTF8(t *testing.T) {
	tests := []struct {
		value string
		want  string
	}{
		{"x\xffy", "x�y"},
		{"\xe3\x82", "��"}, // the first two bytes of "ゲ"
	}
	for _, tc := range tests {
		var buf bytes.Buffer
		w := NewWriter(&buf)
		w.UseUTF16()
		if err := w.Write(NewKeyValue("k", tc.value)); err != nil {
			t.Fatal(err)
		}
		doc, err := readBytes(t, buf.Bytes())
		if err != nil {
			t.Fatal(err)
		}
		if got := asString(t, find(t, doc, "k")); got != tc.want {
			t.Errorf("wrote %q, read back %q, want %q", tc.value, got, tc.want)
		}
	}
}

func TestWriter_UTF16LongValue(t *testing.T) {
	// Long enough that the Writer's buffer is flushed partway through a
	// character.
	kv := NewKeyValue("key", strings.Repeat("aŻ😀", 3000))
	var buf bytes.Buffer
	w := NewWriter(&buf)
	w.UseUTF16()
	if err := w.Write(kv); err != nil {
		t.Fatal(err)
	}
	doc, err := readBytes(t, buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if got := asString(t, find(t, doc, "key")); got != kv.value {
		t.Errorf("value changed: got %d bytes, want %d", len(got), len(kv.value))
	}
}

func TestWriter_UTF16WriteError(t *testing.T) {
	errBoom := errors.New("boom")
	w := NewWriter(failingWriter{errBoom})
	w.UseUTF16()
	if err := w.Write(readFile(t, "map.vmf", false)); !errors.Is(err, errBoom) {
		t.Errorf("got %v, want %v", err, errBoom)
	}
}
