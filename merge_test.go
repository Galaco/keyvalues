package keyvalues

import (
	"strings"
	"testing"
)

func TestMerge(t *testing.T) {
	tests := []struct {
		name        string
		overlay     string
		base        string
		wantPatch   string
		wantReplace string
	}{
		{
			name:        "missing keys are added",
			overlay:     `"foo" { "new" "1" }`,
			base:        `"foo" { "old" "2" }`,
			wantPatch:   `"foo"{"old"="2" "new"="1"}`,
			wantReplace: `"foo"{"old"="2" "new"="1"}`,
		},
		{
			name:        "existing values",
			overlay:     `"foo" { "bar" "new" }`,
			base:        `"foo" { "bar" "old" }`,
			wantPatch:   `"foo"{"bar"="old"}`,
			wantReplace: `"foo"{"bar"="new"}`,
		},
		{
			name:        "nested blocks",
			overlay:     `"foo" { "sub" { "x" "new" "y" "added" } }`,
			base:        `"foo" { "sub" { "x" "old" } "other" "kept" }`,
			wantPatch:   `"foo"{"sub"{"x"="old" "y"="added"} "other"="kept"}`,
			wantReplace: `"foo"{"sub"{"x"="new" "y"="added"} "other"="kept"}`,
		},
		{
			name:        "leaf keys match ignoring case",
			overlay:     `"foo" { "Bar" "new" }`,
			base:        `"foo" { "bar" "old" }`,
			wantPatch:   `"foo"{"bar"="old"}`,
			wantReplace: `"foo"{"bar"="new"}`,
		},
		{
			name:        "block keys match ignoring case",
			overlay:     `"foo" { "Sub" { "x" "new" } }`,
			base:        `"foo" { "sub" { "x" "old" } }`,
			wantPatch:   `"foo"{"sub"{"x"="old"}}`,
			wantReplace: `"foo"{"sub"{"x"="new"}}`,
		},
		{
			name:        "root keys match ignoring case",
			overlay:     `"FOO" { "a" "1" }`,
			base:        `"foo" { }`,
			wantPatch:   `"foo"{"a"="1"}`,
			wantReplace: `"foo"{"a"="1"}`,
		},
		{
			name:        "block over leaf",
			overlay:     `"foo" { "a" { "x" "1" } }`,
			base:        `"foo" { "a" "leaf" }`,
			wantPatch:   `"foo"{"a"="leaf"}`,
			wantReplace: `"foo"{"a"{"x"="1"}}`,
		},
		{
			name:        "leaf over block",
			overlay:     `"foo" { "a" "leaf" }`,
			base:        `"foo" { "a" { "x" "1" } }`,
			wantPatch:   `"foo"{"a"{"x"="1"}}`,
			wantReplace: `"foo"{"a"="leaf"}`,
		},
		{
			name:        "root leaves",
			overlay:     `"foo" "new"`,
			base:        `"foo" "old"`,
			wantPatch:   `"foo"="old"`,
			wantReplace: `"foo"="new"`,
		},
		{
			name:        "repeated keys in overlay are each added",
			overlay:     `"foo" { "solid" { "id" "1" } "solid" { "id" "2" } }`,
			base:        `"foo" { }`,
			wantPatch:   `"foo"{"solid"{"id"="1"} "solid"{"id"="2"}}`,
			wantReplace: `"foo"{"solid"{"id"="1"} "solid"{"id"="2"}}`,
		},
		{
			name:        "repeated keys in base match the first",
			overlay:     `"foo" { "a" "new" }`,
			base:        `"foo" { "a" "1" "a" "2" }`,
			wantPatch:   `"foo"{"a"="1" "a"="2"}`,
			wantReplace: `"foo"{"a"="new" "a"="2"}`,
		},
		{
			name:        "conditionals go with their values",
			overlay:     `"foo" { "a" "new" [$WIN32] }`,
			base:        `"foo" { "a" "old" [$X360] }`,
			wantPatch:   `"foo"{"a"="old"[$X360]}`,
			wantReplace: `"foo"{"a"="new"[$WIN32]}`,
		},
		{
			name:        "empty overlay",
			overlay:     `"foo" { }`,
			base:        `"foo" { "a" "1" }`,
			wantPatch:   `"foo"{"a"="1"}`,
			wantReplace: `"foo"{"a"="1"}`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ops := map[string]struct {
				merge func(overlay, base *KeyValue) (*KeyValue, error)
				want  string
			}{
				"Patch":   {(*KeyValue).Patch, tc.wantPatch},
				"Replace": {(*KeyValue).Replace, tc.wantReplace},
			}
			for name, op := range ops {
				overlay := read(t, tc.overlay).children[0]
				base := read(t, tc.base).children[0]
				got, err := op.merge(overlay, base)
				if err != nil {
					t.Fatalf("%s: %v", name, err)
				}
				if dump(got) != op.want {
					t.Errorf("%s:\ngot  %s\nwant %s", name, dump(got), op.want)
				}
				checkMergeResult(t, got, overlay, base, tc.overlay, tc.base)
			}
		})
	}
}

// checkMergeResult checks a merge left its inputs unchanged and returned an
// independent, well formed tree.
func checkMergeResult(t *testing.T, merged, overlay, base *KeyValue, overlayText, baseText string) {
	t.Helper()
	if got, want := dump(overlay), dump(read(t, overlayText).children[0]); got != want {
		t.Errorf("overlay was modified: got %s, want %s", got, want)
	}
	if got, want := dump(base), dump(read(t, baseText).children[0]); got != want {
		t.Errorf("base was modified: got %s, want %s", got, want)
	}
	if merged.Parent() != nil {
		t.Error("merged root has a parent")
	}
	checkParents(t, merged)

	inputs := map[*KeyValue]bool{}
	var collect func(*KeyValue)
	collect = func(kv *KeyValue) {
		inputs[kv] = true
		for _, c := range kv.children {
			collect(c)
		}
	}
	collect(overlay)
	collect(base)
	var check func(*KeyValue)
	check = func(kv *KeyValue) {
		if inputs[kv] {
			t.Errorf("merged tree shares node %q with an input", kv.key)
		}
		for _, c := range kv.children {
			check(c)
		}
	}
	check(merged)
}

func TestMerge_ReservedKeys(t *testing.T) {
	base := read(t, `"GameInfo" { "game" "old" "nomodels" "1" }`).children[0]
	tests := []struct {
		name    string
		overlay string
		merge   func(overlay, base *KeyValue) (*KeyValue, error)
		want    string // empty if an error is expected
	}{
		{"patch", `"patch" { "game" "new" "type" "mod" }`, (*KeyValue).Patch, `"GameInfo"{"game"="old" "nomodels"="1" "type"="mod"}`},
		{"patch ignoring case", `"Patch" { "type" "mod" }`, (*KeyValue).Patch, `"GameInfo"{"game"="old" "nomodels"="1" "type"="mod"}`},
		{"replace", `"replace" { "game" "new" }`, (*KeyValue).Replace, `"GameInfo"{"game"="new" "nomodels"="1"}`},
		{"replace ignoring case", `"REPLACE" { "game" "new" }`, (*KeyValue).Replace, `"GameInfo"{"game"="new" "nomodels"="1"}`},
		{"replace key with Patch", `"replace" { "game" "new" }`, (*KeyValue).Patch, ""},
		{"patch key with Replace", `"patch" { "game" "new" }`, (*KeyValue).Replace, ""},
		{"different key", `"other" { "game" "new" }`, (*KeyValue).Replace, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			overlay := read(t, tc.overlay).children[0]
			overlayKey := overlay.Key()
			got, err := tc.merge(overlay, base)
			if tc.want == "" {
				if err == nil {
					t.Fatalf("got %s, want an error", dump(got))
				}
				if !strings.Contains(err.Error(), `"`+overlayKey+`"`) || !strings.Contains(err.Error(), `"GameInfo"`) {
					t.Errorf("error %q does not name both keys", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if dump(got) != tc.want {
				t.Errorf("got  %s\nwant %s", dump(got), tc.want)
			}
			if overlay.Key() != overlayKey {
				t.Errorf("overlay key changed from %q to %q", overlayKey, overlay.Key())
			}
		})
	}
}

func TestMerge_Documents(t *testing.T) {
	overlay := read(t, `"GameInfo" { "type" "mod" } "extra" "1"`)
	base := read(t, `"GameInfo" { "game" "Half-Life 2" }`)
	got, err := overlay.Patch(base)
	if err != nil {
		t.Fatal(err)
	}
	if want := `"GameInfo"{"game"="Half-Life 2" "type"="mod"} "extra"="1"`; dump(got) != want {
		t.Errorf("got  %s\nwant %s", dump(got), want)
	}
}
