package keyvalues

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"
)

func write(t testing.TB, kv *KeyValue, escapes bool) string {
	t.Helper()
	var sb strings.Builder
	w := NewWriter(&sb)
	if escapes {
		w.UseEscapeSequences()
	}
	if err := w.Write(kv); err != nil {
		t.Fatalf("Write(%s): %v", dump(kv), err)
	}
	return sb.String()
}

func TestWriter_Write(t *testing.T) {
	tests := []struct {
		name string
		kv   *KeyValue
		want string
	}{
		{
			name: "nested blocks",
			kv: NewBlock("GameInfo",
				NewKeyValue("game", "Counter-Strike Source"),
				NewBlock("FileSystem",
					NewKeyValue("SteamAppId", "240"),
					NewBlock("SearchPaths"),
				),
			),
			want: "\"GameInfo\"\n{\n" +
				"\t\"game\"\t\"Counter-Strike Source\"\n" +
				"\t\"FileSystem\"\n\t{\n" +
				"\t\t\"SteamAppId\"\t\"240\"\n" +
				"\t\t\"SearchPaths\"\n\t\t{\n\t\t}\n" +
				"\t}\n" +
				"}\n",
		},
		{
			name: "leaf",
			kv:   NewKeyValue("a", "b"),
			want: "\"a\"\t\"b\"\n",
		},
		{
			name: "document",
			kv:   read(t, `"a" "b" "c" { }`),
			want: "\"a\"\t\"b\"\n\"c\"\n{\n}\n",
		},
		{
			name: "empty document",
			kv:   read(t, ``),
			want: "",
		},
		{
			name: "subtree",
			kv:   find(t, read(t, `"root" { "child" { "a" "b" } }`), "root", "child"),
			want: "\"child\"\n{\n\t\"a\"\t\"b\"\n}\n",
		},
		{
			name: "empty key with a parent is not a document",
			kv:   find(t, read(t, `"root" { "" { "a" "b" } }`), "root", ""),
			want: "\"\"\n{\n\t\"a\"\t\"b\"\n}\n",
		},
		{
			name: "conditionals",
			kv:   read(t, `"a" "b" [$WIN32] "c" { "d" "e" } [!$OSX]`),
			want: "\"a\"\t\"b\" [$WIN32]\n\"c\" [!$OSX]\n{\n\t\"d\"\t\"e\"\n}\n",
		},
		{
			name: "special characters without escapes",
			kv:   NewKeyValue(`path`, "materials\\nature\nline {two} // not a comment"),
			want: "\"path\"\t\"materials\\nature\nline {two} // not a comment\"\n",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := write(t, tc.kv, false); got != tc.want {
				t.Errorf("got\n%s\nwant\n%s", got, tc.want)
			}
		})
	}
}

func TestWriter_EscapeSequences(t *testing.T) {
	kv := NewKeyValue(`say "key"`, "a\\b\"c\nd\te\rf")
	got := write(t, kv, true)
	want := `"say \"key\""` + "\t" + `"a\\b\"c\nd\te\rf"` + "\n"
	if got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}

	r := NewReader(strings.NewReader(got))
	r.UseEscapeSequences()
	doc, err := r.Read()
	if err != nil {
		t.Fatal(err)
	}
	if dump(doc) != dump(kv) {
		t.Errorf("read back %s, want %s", dump(doc), dump(kv))
	}
}

func TestWriter_QuoteWithoutEscapes(t *testing.T) {
	for _, kv := range []*KeyValue{
		NewKeyValue(`a"b`, "c"),
		NewKeyValue("a", `b"c`),
		NewBlock("root", NewBlock("nested", NewKeyValue("a", `"`))),
	} {
		var sb strings.Builder
		if err := NewWriter(&sb).Write(kv); err == nil {
			t.Errorf("Write(%s) = %q, want an error", dump(kv), sb.String())
		}
	}
}

type failingWriter struct{ err error }

func (w failingWriter) Write([]byte) (int, error) { return 0, w.err }

func TestWriter_WriteError(t *testing.T) {
	errBoom := errors.New("boom")
	doc := readFile(t, "map.vmf", false)
	if err := NewWriter(failingWriter{errBoom}).Write(doc); !errors.Is(err, errBoom) {
		t.Errorf("got %v, want %v", err, errBoom)
	}
}

// TestRoundTrip checks writing a tree and reading it back gives the same tree.
func TestRoundTrip(t *testing.T) {
	inputs := fixtures(t)
	for _, tc := range readCases {
		inputs[tc.name] = tc.input
	}
	for name, input := range inputs {
		for _, escapes := range []bool{false, true} {
			if err := roundTrip(input, escapes); err != nil {
				t.Errorf("%s (escapes %v): %v", name, escapes, err)
			}
		}
	}
}

// roundTrip reads input, then writes it as UTF-8 and as UTF-16 and reads each
// output back, returning an error if any tree differs. Input that fails to
// parse is not an error.
func roundTrip(input string, escapes bool) error {
	doc, err := readString(input, escapes)
	if err != nil {
		return nil
	}
	for _, utf16 := range []bool{false, true} {
		var buf bytes.Buffer
		w := NewWriter(&buf)
		if escapes {
			w.UseEscapeSequences()
		}
		if utf16 {
			if !validUTF8(doc) {
				continue // UTF-16 cannot represent invalid UTF-8
			}
			w.UseUTF16()
		}
		if err := w.Write(doc); err != nil {
			return err
		}
		again, err := readString(buf.String(), escapes)
		if err != nil {
			return fmt.Errorf("UTF-16 %v: reading written output: %v\noutput: %q", utf16, err, buf.String())
		}
		if dump(again) != dump(doc) {
			return fmt.Errorf("UTF-16 %v: tree changed:\nread    %s\nwritten %s", utf16, dump(doc), dump(again))
		}
	}
	return nil
}

func readString(s string, escapes bool) (*KeyValue, error) {
	r := NewReader(strings.NewReader(s))
	if escapes {
		r.UseEscapeSequences()
	}
	return r.Read()
}

func validUTF8(kv *KeyValue) bool {
	if !utf8.ValidString(kv.key) || !utf8.ValidString(kv.value) || !utf8.ValidString(kv.condition) {
		return false
	}
	for _, child := range kv.children {
		if !validUTF8(child) {
			return false
		}
	}
	return true
}
