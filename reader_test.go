package keyvalues

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"testing/iotest"
)

// readCases are valid inputs and the tree each should produce, rendered by
// dump. They are also used as round trip and fuzz seeds.
var readCases = []struct {
	name  string
	input string
	want  string
}{
	{"block", "\"root\"\n{\n\t\"a\"\t\"b\"\n}\n", `"root"{"a"="b"}`},
	{"empty input", "", ``},
	{"whitespace only", " \t\r\n\v\f", ``},
	{"comment only", "// nothing here", ``},
	{"no trailing newline", `"a" "b"`, `"a"="b"`},
	{"last line without newline", "\"x\" \"1\"\n\"a\" \"b\"", `"x"="1" "a"="b"`},
	{"brace on key line", "\"root\" {\n\"a\" \"b\"\n}", `"root"{"a"="b"}`},
	{"brace on key line after sibling", "\"root\"\n{\n\"a\" \"b\"\n\"child\" {\n\"c\" \"d\"\n}\n}", `"root"{"a"="b" "child"{"c"="d"}}`},
	{"close brace after value", "\"root\"\n{\n\"a\" \"b\"\n\"c\" \"d\" }\n", `"root"{"a"="b" "c"="d"}`},
	{"single line block", `"root" { "a" "b" }`, `"root"{"a"="b"}`},
	{"no whitespace", `root{a b c{d e}}`, `"root"{"a"="b" "c"{"d"="e"}}`},
	{"key and value on separate lines", "\"a\"\n\"b\"", `"a"="b"`},
	{"empty block", `"a" {}`, `"a"{}`},
	{"nested empty blocks", `a { b { c { } } }`, `"a"{"b"{"c"{}}}`},
	{"multiple roots", `"r1" { } "r2" { "a" "b" }`, `"r1"{} "r2"{"a"="b"}`},
	{"top level leaves", "#base \"base.res\"\n\"root\" {}", `"#base"="base.res" "root"{}`},
	{"duplicate keys", `"a" "1" "A" "2"`, `"a"="1" "A"="2"`},
	{"braces in quoted value", `"root" { "a" "x}y{z" }`, `"root"{"a"="x}y{z"}`},
	{"comment marker in quoted value", `"root" { "url" "http://example.com" }`, `"root"{"url"="http://example.com"}`},
	{"comment marker in bare value", `url http://example.com`, `"url"="http://example.com"`},
	{"comment after value", "\"a\" \"b\" // \"c\" \"d\"\n\"e\" \"f\"", `"a"="b" "e"="f"`},
	{"comment without newline", `"a" "b" // end`, `"a"="b"`},
	{"comment between key and value", "\"a\" // note\n\"b\"", `"a"="b"`},
	{"single slash is a bare token", `a /b`, `"a"="/b"`},
	{"key with spaces", `"my key" "my value"`, `"my key"="my value"`},
	{"value contains key", `name name_1`, `"name"="name_1"`},
	{"value equals key", `"a" "a"`, `"a"="a"`},
	{"empty value", `"a" ""`, `"a"=""`},
	{"empty key", `"" "b"`, `""="b"`},
	{"tabs and repeated spaces", "\t\"a\"\t\t\t\"b  c\"\t", `"a"="b  c"`},
	{"surrounding spaces in value kept", `"a" " b "`, `"a"=" b "`},
	{"multi-line quoted value", "\"a\" \"line 1\nline 2\"", `"a"="line 1\nline 2"`},
	{"backslashes are literal", `"path" "materials\nature\tree"`, `"path"="materials\\nature\\tree"`},
	{"escaped quote is not special", `"a" "say \"hi\""`, `"a"="say \\" "hi\\"=""`},
	{"numbers kept as written", `"a" "007" "b" "1.50" "c" 1e5`, `"a"="007" "b"="1.50" "c"="1e5"`},
	{"conditional after value", `"a" "b" [$WIN32]`, `"a"="b"[$WIN32]`},
	{"conditional before value", `"a" [$WIN32] "b"`, `"a"="b"[$WIN32]`},
	{"conditional before block", `"a" [$X360] { "b" "c" }`, `"a"{"b"="c"}[$X360]`},
	{"conditional after block", `"a" { "b" "c" } [!$OSX]`, `"a"{"b"="c"}[!$OSX]`},
	{"conditional without space", `"a" "b"[$WIN32]`, `"a"="b"[$WIN32]`},
	{"quoted conditional is a value", `"a" "[$WIN32]"`, `"a"="[$WIN32]"`},
	{"bracketed value with spaces", `"uaxis" "[1 0 0 0] 0.25"`, `"uaxis"="[1 0 0 0] 0.25"`},
	{"CRLF line endings", "\"root\"\r\n{\r\n\"a\" \"b\"\r\n}\r\n", `"root"{"a"="b"}`},
	{"CR line endings", "\"root\"\r{\r\"a\" \"b\"\r}\r", `"root"{"a"="b"}`},
	{"byte order mark", "\xef\xbb\xbf\"root\" { }", `"root"{}`},
	{"byte order mark only", "\xef\xbb\xbf", ``},
}

func TestRead(t *testing.T) {
	for _, tc := range readCases {
		t.Run(tc.name, func(t *testing.T) {
			doc := read(t, tc.input)
			if got := dump(doc); got != tc.want {
				t.Errorf("got  %s\nwant %s", got, tc.want)
			}
			if doc.Key() != "" || !doc.IsBlock() || doc.Parent() != nil {
				t.Errorf("document root: key %q, block %v, parent %v", doc.Key(), doc.IsBlock(), doc.Parent())
			}
			checkParents(t, doc)
		})
	}
}

// readErrorCases are malformed inputs and where the error should be reported.
var readErrorCases = []struct {
	name   string
	input  string
	line   int
	column int
	msg    string
}{
	{"block without key", `{ "a" "b" }`, 1, 1, "unexpected '{', expected a key"},
	{"unclosed block", `"a" { "b" "c"`, 1, 1, `block "a" is never closed`},
	{"unclosed nested block", "\"a\"\n{\n\t\"b\"\n\t{\n", 3, 2, `block "b" is never closed`},
	{"extra close brace", `"a" { "b" "c" } }`, 1, 17, "unexpected '}' with no open block"},
	{"close brace at root", `"a" "b" }`, 1, 9, "unexpected '}' with no open block"},
	{"key without value at EOF", `"a" "b" "c"`, 1, 9, `key "c" has no value`},
	{"key without value in block", "\"a\"\n{\n\t\"b\"\n}", 4, 1, `unexpected '}', expected a value for key "b"`},
	{"unterminated quoted string", `"a" "b`, 1, 5, "unterminated quoted string"},
	{"conditional instead of key", `[$WIN32] "a" "b"`, 1, 1, "unexpected conditional [$WIN32], expected a key"},
	{"two conditionals before value", `"a" [$X] [$Y] "b"`, 1, 10, `unexpected conditional [$Y], expected a value for key "a"`},
	{"conditional before and after value", `"a" [$X] "b" [$Y]`, 1, 14, `key "a" has more than one conditional`},
	{"conditional before and after block", `"a" [$X] { } [$Y]`, 1, 14, `key "a" has more than one conditional`},
	{"conditional then EOF", `"a" [$X]`, 1, 1, `key "a" has no value`},
	{"column counts bytes after newline", "\"a\" \"b\"\n  }", 2, 3, "unexpected '}' with no open block"},
}

func TestRead_SyntaxErrors(t *testing.T) {
	for _, tc := range readErrorCases {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := NewReader(strings.NewReader(tc.input)).Read()
			if doc != nil {
				t.Errorf("got document %s, want nil", dump(doc))
			}
			var syntaxErr *SyntaxError
			if !errors.As(err, &syntaxErr) {
				t.Fatalf("got error %v, want a *SyntaxError", err)
			}
			want := SyntaxError{Line: tc.line, Column: tc.column, Msg: tc.msg}
			if *syntaxErr != want {
				t.Errorf("got  %+v\nwant %+v", *syntaxErr, want)
			}
			wantText := fmt.Sprintf("keyvalues: syntax error at line %d, column %d: %s", tc.line, tc.column, tc.msg)
			if err.Error() != wantText {
				t.Errorf("Error() = %q, want %q", err.Error(), wantText)
			}
		})
	}
}

func TestRead_EscapeSequences(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		want       string // with escape sequences
		wantPlain  string // without
		plainError bool
	}{
		{
			name:      "quote",
			input:     `"a" "say \"hi\""`,
			want:      `"a"="say \"hi\""`,
			wantPlain: `"a"="say \\" "hi\\"=""`,
		},
		{
			name:      "common sequences",
			input:     `"a" "1\n2\t3\\4"`,
			want:      `"a"="1\n2\t3\\4"`,
			wantPlain: `"a"="1\\n2\\t3\\\\4"`,
		},
		{
			name:      "all sequences",
			input:     `"a" "\v\b\r\f\a\?\'"`,
			want:      `"a"="\v\b\r\f\a?'"`,
			wantPlain: `"a"="\\v\\b\\r\\f\\a\\?\\'"`,
		},
		{
			name:      "unknown sequence kept",
			input:     `"a" "\x\q"`,
			want:      `"a"="\\x\\q"`,
			wantPlain: `"a"="\\x\\q"`,
		},
		{
			name:  "escaped key",
			input: `"a\"b" "c"`,
			want:  `"a\"b"="c"`,
			// Without escapes the key ends at the backslash, leaving an
			// unbalanced quote.
			plainError: true,
		},
		{
			name:      "bare strings are not unescaped",
			input:     `a b\n`,
			want:      `"a"="b\\n"`,
			wantPlain: `"a"="b\\n"`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := NewReader(strings.NewReader(tc.input))
			r.UseEscapeSequences()
			doc, err := r.Read()
			if err != nil {
				t.Fatal(err)
			}
			if got := dump(doc); got != tc.want {
				t.Errorf("with escapes:\ngot  %s\nwant %s", got, tc.want)
			}

			doc, err = NewReader(strings.NewReader(tc.input)).Read()
			if tc.plainError {
				if err == nil {
					t.Errorf("without escapes: got %s, want an error", dump(doc))
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got := dump(doc); got != tc.wantPlain {
				t.Errorf("without escapes:\ngot  %s\nwant %s", got, tc.wantPlain)
			}
		})
	}
}

func TestRead_EscapeAtEOF(t *testing.T) {
	r := NewReader(strings.NewReader(`"a" "b\`))
	r.UseEscapeSequences()
	_, err := r.Read()
	var syntaxErr *SyntaxError
	if !errors.As(err, &syntaxErr) || syntaxErr.Msg != "unterminated quoted string" {
		t.Errorf("got %v, want an unterminated quoted string error", err)
	}
}

func TestRead_ReaderErrors(t *testing.T) {
	errBoom := errors.New("boom")
	tests := []struct {
		name string
		r    io.Reader
	}{
		{"immediately", iotest.ErrReader(errBoom)},
		{"in whitespace", io.MultiReader(strings.NewReader(`"a" "b"  `), iotest.ErrReader(errBoom))},
		{"in comment", io.MultiReader(strings.NewReader(`"a" "b" // c`), iotest.ErrReader(errBoom))},
		{"after single slash", io.MultiReader(strings.NewReader(`"a" "b" /`), iotest.ErrReader(errBoom))},
		{"in quoted string", io.MultiReader(strings.NewReader(`"a" { "b" "c`), iotest.ErrReader(errBoom))},
		{"in bare string", io.MultiReader(strings.NewReader(`a { b c`), iotest.ErrReader(errBoom))},
		{"peeking for conditional", io.MultiReader(strings.NewReader(`"a" "b"`), iotest.ErrReader(errBoom))},
		{"after conditional", io.MultiReader(strings.NewReader(`"a" [$X] `), iotest.ErrReader(errBoom))},
		{"after value", io.MultiReader(strings.NewReader(`"a"`), iotest.ErrReader(errBoom))},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := NewReader(tc.r).Read()
			if !errors.Is(err, errBoom) {
				t.Errorf("got error %v, want %v", err, errBoom)
			}
			if doc != nil {
				t.Errorf("got document %s, want nil", dump(doc))
			}
		})
	}
}

func TestRead_EscapedReaderErrors(t *testing.T) {
	errBoom := errors.New("boom")
	r := NewReader(io.MultiReader(strings.NewReader(`"a" "b\`), iotest.ErrReader(errBoom)))
	r.UseEscapeSequences()
	if _, err := r.Read(); !errors.Is(err, errBoom) {
		t.Errorf("got error %v, want %v", err, errBoom)
	}
}

// TestRead_ReaderBehaviour checks parsing doesn't depend on how the
// underlying reader splits its input.
func TestRead_ReaderBehaviour(t *testing.T) {
	const input = "\"root\"\n{\n\t\"a\" \"b\" // comment\n\t\"c\" { \"d\" \"e\" } [$X]\n}"
	const want = `"root"{"a"="b" "c"{"d"="e"}[$X]}`
	readers := map[string]func(io.Reader) io.Reader{
		"one byte":   iotest.OneByteReader,
		"half":       iotest.HalfReader,
		"data + EOF": iotest.DataErrReader,
	}
	for name, wrap := range readers {
		t.Run(name, func(t *testing.T) {
			doc, err := NewReader(wrap(strings.NewReader(input))).Read()
			if err != nil {
				t.Fatal(err)
			}
			if got := dump(doc); got != want {
				t.Errorf("got  %s\nwant %s", got, want)
			}
		})
	}
}

func TestRead_GameInfo(t *testing.T) {
	doc := readFile(t, "gameinfo.txt", false)

	// Keys are matched ignoring case.
	gameInfo := find(t, doc, "gameinfo")
	if got := asString(t, find(t, gameInfo, "game")); got != "Counter-Strike Source" {
		t.Errorf("game = %q", got)
	}
	if got := asString(t, find(t, gameInfo, "developer_url")); got != "http://www.valvesoftware.com/" {
		t.Errorf("developer_url = %q", got)
	}
	if got, err := find(t, gameInfo, "nomodels").AsInt(); err != nil || got != 1 {
		t.Errorf("nomodels = %d, %v", got, err)
	}
	if got, err := find(t, gameInfo, "FileSystem", "SteamAppId").AsInt(); err != nil || got != 240 {
		t.Errorf("SteamAppId = %d, %v", got, err)
	}

	var paths []string
	for _, kv := range find(t, gameInfo, "FileSystem", "SearchPaths").FindAll("game") {
		paths = append(paths, asString(t, kv))
	}
	if got, want := strings.Join(paths, ","), "|gameinfo_path|.,cstrike,hl2"; got != want {
		t.Errorf("search paths = %s, want %s", got, want)
	}
	checkParents(t, doc)
}

func TestRead_Material(t *testing.T) {
	// material.vmt has a byte order mark, CRLF line endings, unquoted keys
	// and backslashed paths.
	doc := readFile(t, "material.vmt", false)
	if n := len(doc.Children()); n != 1 {
		t.Fatalf("got %d top-level nodes, want 1", n)
	}
	shader := find(t, doc, "LightmappedGeneric")

	want := map[string]string{
		"$basetexture": `concrete\concretefloor001a`,
		"$surfaceprop": "concrete",
		"$envmaptint":  "[ .5 .5 .5 ]",
	}
	for key, value := range want {
		if got := asString(t, find(t, shader, key)); got != value {
			t.Errorf("%s = %q, want %q", key, got, value)
		}
	}
	if got := asString(t, find(t, shader, ">=dx90", "$detail")); got != `detail\noise_detail_01` {
		t.Errorf("$detail = %q", got)
	}
	if got, err := find(t, shader, "$alpha").AsFloat(); err != nil || got != 0.75 {
		t.Errorf("$alpha = %v, %v", got, err)
	}
	if got, err := find(t, shader, "Proxies", "AnimatedTexture", "animatedtextureframerate").AsInt(); err != nil || got != 30 {
		t.Errorf("animatedtextureframerate = %v, %v", got, err)
	}
	if got := find(t, shader, "$bumpmap").Condition(); got != "$WIN32" {
		t.Errorf("$bumpmap condition = %q", got)
	}
	checkParents(t, doc)
}

func TestRead_Map(t *testing.T) {
	doc := readFile(t, "map.vmf", false)

	var keys []string
	for _, kv := range doc.Children() {
		keys = append(keys, kv.Key())
	}
	if got, want := strings.Join(keys, ","), "versioninfo,world,entity,cameras"; got != want {
		t.Errorf("top-level keys = %s, want %s", got, want)
	}

	sides := find(t, doc, "world", "solid").FindAll("side")
	if len(sides) != 2 {
		t.Fatalf("got %d sides, want 2", len(sides))
	}
	if got := asString(t, find(t, sides[1], "plane")); got != "(-64 -64 -64) (64 -64 -64) (64 64 -64)" {
		t.Errorf("plane = %q", got)
	}
	if got := asString(t, find(t, doc, "entity", "connections", "OnMapSpawn")); got != "door,Open,,0,-1" {
		t.Errorf("OnMapSpawn = %q", got)
	}
	checkParents(t, doc)
}

func TestRead_Resource(t *testing.T) {
	doc := readFile(t, "hud.res", true)

	if got := asString(t, find(t, doc, "#base")); got != "base.res" {
		t.Errorf("#base = %q", got)
	}
	label := find(t, doc, "Resource/UI/Hud.res", "Label")
	if got := asString(t, find(t, label, "labelText")); got != "Line one\nLine two" {
		t.Errorf("labelText = %q", got)
	}
	if got := asString(t, find(t, label, "tooltip")); got != `Say "hi"` {
		t.Errorf("tooltip = %q", got)
	}
	wide := label.FindAll("wide")
	if len(wide) != 2 || wide[0].Condition() != "$WIN32" || wide[1].Condition() != "$X360" {
		t.Errorf("wide = %s", dumpNodes(wide))
	}
	if got := find(t, doc, "Resource/UI/Hud.res", "Panel").Condition(); got != "!$OSX" {
		t.Errorf("Panel condition = %q", got)
	}
}

func BenchmarkRead(b *testing.B) {
	var sb strings.Builder
	sb.WriteString("world\n{\n\t\"classname\" \"worldspawn\"\n")
	for i := 0; i < 1000; i++ {
		fmt.Fprintf(&sb, "\tsolid\n\t{\n\t\t\"id\" \"%d\"\n", i)
		for j := 0; j < 6; j++ {
			fmt.Fprintf(&sb, "\t\tside\n\t\t{\n\t\t\t\"id\" \"%d\"\n\t\t\t\"plane\" \"(0 0 0) (1 1 1) (2 2 2)\"\n\t\t\t\"material\" \"TOOLS/TOOLSNODRAW\"\n\t\t}\n", j)
		}
		sb.WriteString("\t}\n")
	}
	sb.WriteString("}\n")
	input := sb.String()

	b.SetBytes(int64(len(input)))
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := NewReader(strings.NewReader(input)).Read(); err != nil {
			b.Fatal(err)
		}
	}
}
