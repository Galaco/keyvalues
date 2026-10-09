package keyvalues

import (
	"errors"
	"strconv"
	"strings"
	"testing"
)

func TestNewKeyValue(t *testing.T) {
	kv := NewKeyValue("key", "value")
	if kv.Key() != "key" || kv.IsBlock() || kv.Parent() != nil || kv.Children() != nil || kv.Condition() != "" {
		t.Errorf("got %s, parent %v", dump(kv), kv.Parent())
	}
	if got := asString(t, kv); got != "value" {
		t.Errorf("AsString() = %q", got)
	}
}

func TestKeyValue_ZeroValueIsEmptyLeaf(t *testing.T) {
	var kv KeyValue
	if kv.IsBlock() {
		t.Error("zero value is a block")
	}
	if got := asString(t, &kv); got != "" {
		t.Errorf("AsString() = %q", got)
	}
}

func TestNewBlock(t *testing.T) {
	a, b := NewKeyValue("a", "1"), NewKeyValue("b", "2")
	block := NewBlock("block", a, b)
	if got, want := dump(block), `"block"{"a"="1" "b"="2"}`; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
	if !block.IsBlock() || block.Parent() != nil {
		t.Errorf("IsBlock() = %v, Parent() = %v", block.IsBlock(), block.Parent())
	}
	checkParents(t, block)

	empty := NewBlock("empty")
	if !empty.IsBlock() || empty.Children() != nil {
		t.Errorf("empty block: IsBlock() = %v, Children() = %v", empty.IsBlock(), empty.Children())
	}
}

func TestNewBlock_MovesChildren(t *testing.T) {
	child := NewKeyValue("child", "v")
	first := NewBlock("first", child)
	second := NewBlock("second", child)
	if got := dump(first); got != `"first"{}` {
		t.Errorf("first = %s", got)
	}
	if child.Parent() != second || dump(second) != `"second"{"child"="v"}` {
		t.Errorf("second = %s, child parent = %v", dump(second), child.Parent())
	}
}

func TestNewBlock_NilChildPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("NewBlock with a nil child did not panic")
		}
	}()
	NewBlock("block", nil)
}

func TestKeyValue_Find(t *testing.T) {
	root := read(t, `"root" { "bar" "first" "BAZ" "123" "Bar" "second" }`).children[0]
	tests := []struct {
		key  string
		want string
	}{
		{"bar", `"bar"="first"`},
		{"BAR", `"bar"="first"`},
		{"baz", `"BAZ"="123"`},
	}
	for _, tc := range tests {
		got, err := root.Find(tc.key)
		if err != nil {
			t.Errorf("Find(%q): %v", tc.key, err)
			continue
		}
		if dump(got) != tc.want {
			t.Errorf("Find(%q) = %s, want %s", tc.key, dump(got), tc.want)
		}
	}
}

func TestKeyValue_FindErrors(t *testing.T) {
	root := NewBlock("root", NewKeyValue("a", "b"))

	got, err := root.Find("missing")
	if got != nil || !errors.Is(err, ErrNotFound) {
		t.Errorf("Find(missing) = %v, %v; want ErrNotFound", got, err)
	}
	if err != nil && !strings.Contains(err.Error(), `"missing"`) {
		t.Errorf("error %q does not name the key", err)
	}

	leaf := root.children[0]
	got, err = leaf.Find("x")
	if got != nil || !errors.Is(err, ErrNotBlock) {
		t.Errorf("Find on leaf = %v, %v; want ErrNotBlock", got, err)
	}

	got, err = NewBlock("empty").Find("x")
	if got != nil || !errors.Is(err, ErrNotFound) {
		t.Errorf("Find on empty block = %v, %v; want ErrNotFound", got, err)
	}
}

func TestKeyValue_FindAll(t *testing.T) {
	root := read(t, `"root" { "side" "1" "id" "9" "SIDE" "2" "Side" "3" }`).children[0]
	if got, want := dumpNodes(root.FindAll("side")), `"side"="1" "SIDE"="2" "Side"="3"`; got != want {
		t.Errorf("FindAll(side) = %s, want %s", got, want)
	}
	if got := root.FindAll("missing"); got != nil {
		t.Errorf("FindAll(missing) = %v, want nil", got)
	}
	if got := NewKeyValue("a", "b").FindAll("a"); got != nil {
		t.Errorf("FindAll on leaf = %v, want nil", got)
	}
}

func TestKeyValue_Children(t *testing.T) {
	root := NewBlock("root", NewKeyValue("a", "1"), NewKeyValue("b", "2"))
	children := root.Children()
	if got, want := dumpNodes(children), `"a"="1" "b"="2"`; got != want {
		t.Fatalf("Children() = %s, want %s", got, want)
	}

	// The returned slice is a copy.
	children[0] = NewKeyValue("x", "y")
	if got, want := dump(root), `"root"{"a"="1" "b"="2"}`; got != want {
		t.Errorf("modifying Children() changed the block: %s", got)
	}

	if got := NewKeyValue("a", "b").Children(); got != nil {
		t.Errorf("leaf Children() = %v, want nil", got)
	}
}

func TestKeyValue_AddChild(t *testing.T) {
	root := NewBlock("root")
	child := NewKeyValue("a", "1")
	if err := root.AddChild(child); err != nil {
		t.Fatal(err)
	}
	if err := root.AddChild(NewBlock("b")); err != nil {
		t.Fatal(err)
	}
	if got, want := dump(root), `"root"{"a"="1" "b"{}}`; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
	checkParents(t, root)
}

func TestKeyValue_AddChildMoves(t *testing.T) {
	from := NewBlock("from", NewKeyValue("a", "1"), NewKeyValue("b", "2"), NewKeyValue("c", "3"))
	to := NewBlock("to")
	b := from.children[1]
	if err := to.AddChild(b); err != nil {
		t.Fatal(err)
	}
	if got, want := dump(from), `"from"{"a"="1" "c"="3"}`; got != want {
		t.Errorf("from = %s, want %s", got, want)
	}
	if got, want := dump(to), `"to"{"b"="2"}`; got != want {
		t.Errorf("to = %s, want %s", got, want)
	}
	checkParents(t, from)
	checkParents(t, to)

	// Re-adding to the same block moves the child to the end.
	if err := from.AddChild(from.children[0]); err != nil {
		t.Fatal(err)
	}
	if got, want := dump(from), `"from"{"c"="3" "a"="1"}`; got != want {
		t.Errorf("from = %s, want %s", got, want)
	}
}

func TestKeyValue_AddChildErrors(t *testing.T) {
	leaf := NewKeyValue("leaf", "v")
	if err := leaf.AddChild(NewKeyValue("a", "b")); !errors.Is(err, ErrNotBlock) {
		t.Errorf("AddChild to leaf: got %v, want ErrNotBlock", err)
	}

	root := NewBlock("root")
	if err := root.AddChild(nil); err == nil {
		t.Error("AddChild(nil): got no error")
	}
	if err := root.AddChild(root); err == nil {
		t.Error("AddChild(self): got no error")
	}

	child := NewBlock("child")
	grandchild := NewBlock("grandchild")
	if err := root.AddChild(child); err != nil {
		t.Fatal(err)
	}
	if err := child.AddChild(grandchild); err != nil {
		t.Fatal(err)
	}
	if err := grandchild.AddChild(root); err == nil {
		t.Error("adding an ancestor: got no error")
	}
	if got, want := dump(root), `"root"{"child"{"grandchild"{}}}`; got != want {
		t.Errorf("failed AddChild changed the tree: %s", got)
	}
}

func TestKeyValue_RemoveChild(t *testing.T) {
	root := read(t, `"root" { "a" "1" "B" "2" "c" "3" "b" "4" }`).children[0]
	removed := root.children[1]
	if err := root.RemoveChild("b"); err != nil {
		t.Fatal(err)
	}
	if got, want := dump(root), `"root"{"a"="1" "c"="3" "b"="4"}`; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
	if removed.Parent() != nil {
		t.Error("removed child still has a parent")
	}
	checkParents(t, root)

	if err := root.RemoveChild("missing"); !errors.Is(err, ErrNotFound) {
		t.Errorf("RemoveChild(missing): got %v, want ErrNotFound", err)
	}
	if err := NewKeyValue("a", "b").RemoveChild("a"); !errors.Is(err, ErrNotBlock) {
		t.Errorf("RemoveChild on leaf: got %v, want ErrNotBlock", err)
	}
}

func TestKeyValue_AsString(t *testing.T) {
	// Values that look like numbers are still strings.
	for _, value := range []string{"hello", "", "123", "1.0", " padded "} {
		if got := asString(t, NewKeyValue("k", value)); got != value {
			t.Errorf("AsString() = %q, want %q", got, value)
		}
	}
	if _, err := NewBlock("b").AsString(); !errors.Is(err, ErrNotLeaf) {
		t.Errorf("AsString on block: got %v, want ErrNotLeaf", err)
	}
}

func TestKeyValue_AsInt(t *testing.T) {
	tests := []struct {
		value string
		want  int32
		err   error
	}{
		{"1", 1, nil},
		{"-5", -5, nil},
		{"+7", 7, nil},
		{" 42\t", 42, nil},
		{"007", 7, nil},
		{"2147483647", 2147483647, nil},
		{"-2147483648", -2147483648, nil},
		{"2147483648", 0, strconv.ErrRange},
		{"1.5", 0, strconv.ErrSyntax},
		{"1e3", 0, strconv.ErrSyntax},
		{"0x10", 0, strconv.ErrSyntax},
		{"abc", 0, strconv.ErrSyntax},
		{"", 0, strconv.ErrSyntax},
	}
	for _, tc := range tests {
		got, err := NewKeyValue("k", tc.value).AsInt()
		if got != tc.want || !errors.Is(err, tc.err) {
			t.Errorf("AsInt(%q) = %d, %v; want %d, %v", tc.value, got, err, tc.want, tc.err)
		}
	}
	if _, err := NewBlock("b").AsInt(); !errors.Is(err, ErrNotLeaf) {
		t.Errorf("AsInt on block: got %v, want ErrNotLeaf", err)
	}
}

func TestKeyValue_AsFloat(t *testing.T) {
	tests := []struct {
		value string
		want  float32
		err   error
	}{
		{"1", 1, nil},
		{".75", 0.75, nil},
		{"-2.5", -2.5, nil},
		{"1e5", 1e5, nil},
		{"1.5e3", 1500, nil},
		{" 3 ", 3, nil},
		{"1e40", 0, strconv.ErrRange},
		{"1.2.3", 0, strconv.ErrSyntax},
		{"abc", 0, strconv.ErrSyntax},
		{"", 0, strconv.ErrSyntax},
	}
	for _, tc := range tests {
		got, err := NewKeyValue("k", tc.value).AsFloat()
		if got != tc.want || !errors.Is(err, tc.err) {
			t.Errorf("AsFloat(%q) = %v, %v; want %v, %v", tc.value, got, err, tc.want, tc.err)
		}
	}
	if _, err := NewBlock("b").AsFloat(); !errors.Is(err, ErrNotLeaf) {
		t.Errorf("AsFloat on block: got %v, want ErrNotLeaf", err)
	}
}

func TestKeyValue_ConversionErrorNamesKey(t *testing.T) {
	_, err := NewKeyValue("nomodels", "yes").AsInt()
	if err == nil || !strings.Contains(err.Error(), `"nomodels"`) {
		t.Errorf("error %v does not name the key", err)
	}
}

func TestKeyValue_SetValue(t *testing.T) {
	kv := NewKeyValue("k", "old")
	if err := kv.SetValue("new"); err != nil {
		t.Fatal(err)
	}
	if got := asString(t, kv); got != "new" {
		t.Errorf("AsString() = %q, want new", got)
	}
	if err := NewBlock("b").SetValue("v"); !errors.Is(err, ErrNotLeaf) {
		t.Errorf("SetValue on block: got %v, want ErrNotLeaf", err)
	}
}

func TestKeyValue_Parent(t *testing.T) {
	doc := read(t, `"root" { "child" { "leaf" "v" } }`)
	root := find(t, doc, "root")
	child := find(t, root, "child")
	leaf := find(t, child, "leaf")
	if doc.Parent() != nil || root.Parent() != doc || child.Parent() != root || leaf.Parent() != child {
		t.Error("parents do not match the tree")
	}
}
