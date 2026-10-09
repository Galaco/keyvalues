package keyvalues

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"unicode/utf16"
)

// dump renders a tree compactly for comparison in tests. Leaves are written
// "key"="value", blocks "key"{...}, and conditionals are appended as [cond].
// A document root is written as its children only.
func dump(kv *KeyValue) string {
	if kv.block && kv.key == "" && kv.parent == nil {
		return dumpNodes(kv.children)
	}
	var sb strings.Builder
	sb.WriteString(strconv.Quote(kv.key))
	if kv.block {
		sb.WriteString("{" + dumpNodes(kv.children) + "}")
	} else {
		sb.WriteString("=" + strconv.Quote(kv.value))
	}
	if kv.condition != "" {
		sb.WriteString("[" + kv.condition + "]")
	}
	return sb.String()
}

func dumpNodes(nodes []*KeyValue) string {
	parts := make([]string, len(nodes))
	for i, n := range nodes {
		parts[i] = dump(n)
	}
	return strings.Join(parts, " ")
}

// checkParents fails the test unless every node's parent is the block that
// holds it.
func checkParents(t testing.TB, kv *KeyValue) {
	t.Helper()
	for _, child := range kv.children {
		if child.parent != kv {
			t.Errorf("%q: parent is %p, want %p (%q)", child.key, child.parent, kv, kv.key)
		}
		checkParents(t, child)
	}
}

func read(t testing.TB, input string) *KeyValue {
	t.Helper()
	doc, err := NewReader(strings.NewReader(input)).Read()
	if err != nil {
		t.Fatalf("Read(%q): %v", input, err)
	}
	return doc
}

func readFile(t testing.TB, name string, escapes bool) *KeyValue {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	r := NewReader(bytes.NewReader(data))
	if escapes {
		r.UseEscapeSequences()
	}
	doc, err := r.Read()
	if err != nil {
		t.Fatalf("Read(%s): %v", name, err)
	}
	return doc
}

// find follows a path of keys from kv, failing the test if any is missing.
func find(t testing.TB, kv *KeyValue, path ...string) *KeyValue {
	t.Helper()
	for _, key := range path {
		var err error
		if kv, err = kv.Find(key); err != nil {
			t.Fatal(err)
		}
	}
	return kv
}

func asString(t testing.TB, kv *KeyValue) string {
	t.Helper()
	s, err := kv.AsString()
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// fixtures returns the contents of each file in testdata, by path.
func fixtures(t testing.TB) map[string]string {
	t.Helper()
	entries, err := os.ReadDir("testdata")
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		path := filepath.Join("testdata", entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		files[path] = string(data)
	}
	return files
}

// encodeUTF16 encodes s as UTF-16 with a byte order mark.
func encodeUTF16(s string, order binary.ByteOrder) []byte {
	return unitsToBytes(order, utf16.Encode([]rune(s)))
}

// unitsToBytes encodes UTF-16 code units with a byte order mark.
func unitsToBytes(order binary.ByteOrder, units []uint16) []byte {
	b := make([]byte, 2+2*len(units))
	order.PutUint16(b, 0xFEFF)
	for i, u := range units {
		order.PutUint16(b[2+2*i:], u)
	}
	return b
}
