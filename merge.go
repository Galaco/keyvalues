package keyvalues

import (
	"fmt"
	"strings"
)

// Reserved root keys that let Patch and Replace apply kv to a base with a
// different key.
const (
	reservedKeyPatch   = "patch"
	reservedKeyReplace = "replace"
)

// Patch returns a copy of base with any keys from kv that base lacks added to
// it. Values already in base are kept, and blocks found in both are patched
// recursively. Keys are matched ignoring case, against the first child of base
// with that key. Neither kv nor base is modified.
//
// kv must have the same key as base, or the key "patch".
func (kv *KeyValue) Patch(base *KeyValue) (*KeyValue, error) {
	return merge(kv, base, reservedKeyPatch, false)
}

// Replace is like Patch, except values in kv overwrite those in base. Where a
// key is a leaf in one tree and a block in the other, kv's node replaces
// base's.
//
// kv must have the same key as base, or the key "replace".
func (kv *KeyValue) Replace(base *KeyValue) (*KeyValue, error) {
	return merge(kv, base, reservedKeyReplace, true)
}

func merge(overlay, base *KeyValue, reservedKey string, replace bool) (*KeyValue, error) {
	if !strings.EqualFold(overlay.key, base.key) && !strings.EqualFold(overlay.key, reservedKey) {
		return nil, fmt.Errorf("keyvalues: cannot merge %q into %q: keys differ", overlay.key, base.key)
	}
	merged := base.clone()
	mergeInto(merged, overlay, replace)
	return merged, nil
}

// mergeInto merges src into dst. dst must not share nodes with src.
func mergeInto(dst, src *KeyValue, replace bool) {
	if !dst.block || !src.block {
		if replace {
			dst.value, dst.block, dst.condition = src.value, src.block, src.condition
			dst.children = nil
			for _, child := range src.children {
				dst.appendChild(child.clone())
			}
		}
		return
	}
	// Only match against dst's original children, so repeated keys in src
	// are each added rather than merged into one another.
	original := dst.children
	for _, child := range src.children {
		if match := findIn(original, child.key); match != nil {
			mergeInto(match, child, replace)
		} else {
			dst.appendChild(child.clone())
		}
	}
}
