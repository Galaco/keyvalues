package keyvalues

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var (
	// ErrNotFound is returned when no child has the requested key.
	ErrNotFound = errors.New("keyvalues: key not found")
	// ErrNotBlock is returned when an operation that needs child nodes is
	// used on a leaf.
	ErrNotBlock = errors.New("keyvalues: not a block")
	// ErrNotLeaf is returned when an operation that needs a value is used on
	// a block.
	ErrNotLeaf = errors.New("keyvalues: not a leaf")
)

// KeyValue is a node in a KeyValues tree. It is either a leaf, which holds a
// string value, or a block, which holds child nodes.
//
// A node belongs to at most one block. The zero value is an empty leaf.
type KeyValue struct {
	key       string
	value     string
	children  []*KeyValue
	block     bool
	condition string
	parent    *KeyValue
}

// NewKeyValue returns a leaf holding value.
func NewKeyValue(key, value string) *KeyValue {
	return &KeyValue{key: key, value: value}
}

// NewBlock returns a block holding children, moving any child that already
// belongs to another block. It panics if a child is nil.
func NewBlock(key string, children ...*KeyValue) *KeyValue {
	kv := &KeyValue{key: key, block: true}
	for _, child := range children {
		if err := kv.AddChild(child); err != nil {
			panic(err)
		}
	}
	return kv
}

// Key returns the node's key.
func (kv *KeyValue) Key() string {
	return kv.key
}

// IsBlock reports whether the node is a block, which holds child nodes rather
// than a value. An empty block is still a block.
func (kv *KeyValue) IsBlock() bool {
	return kv.block
}

// Condition returns the conditional attached to the node, without brackets,
// such as "$WIN32" for `"key" "value" [$WIN32]`. It returns "" if there is
// none. Conditionals are not evaluated.
func (kv *KeyValue) Condition() string {
	return kv.condition
}

// Parent returns the block holding the node, or nil for a root.
func (kv *KeyValue) Parent() *KeyValue {
	return kv.parent
}

// Children returns a copy of a block's child nodes, in order. It returns nil
// for a leaf.
func (kv *KeyValue) Children() []*KeyValue {
	if len(kv.children) == 0 {
		return nil
	}
	return append([]*KeyValue(nil), kv.children...)
}

// Find returns the first child whose key matches key, ignoring case.
// It returns an error wrapping ErrNotFound if there is no such child, or
// ErrNotBlock if kv is a leaf.
func (kv *KeyValue) Find(key string) (*KeyValue, error) {
	if !kv.block {
		return nil, fmt.Errorf("%w: cannot find %q in %q", ErrNotBlock, key, kv.key)
	}
	if child := findIn(kv.children, key); child != nil {
		return child, nil
	}
	return nil, fmt.Errorf("%w: %q in %q", ErrNotFound, key, kv.key)
}

// FindAll returns every child whose key matches key, ignoring case. Keys may
// repeat, such as "solid" in a .vmf. It returns nil if there are no matches or
// kv is a leaf.
func (kv *KeyValue) FindAll(key string) []*KeyValue {
	var matches []*KeyValue
	for _, child := range kv.children {
		if strings.EqualFold(child.key, key) {
			matches = append(matches, child)
		}
	}
	return matches
}

// AddChild appends child to a block. If child already belongs to a block it
// is moved.
func (kv *KeyValue) AddChild(child *KeyValue) error {
	if child == nil {
		return errors.New("keyvalues: cannot add a nil child")
	}
	if !kv.block {
		return fmt.Errorf("%w: cannot add a child to %q", ErrNotBlock, kv.key)
	}
	for ancestor := kv; ancestor != nil; ancestor = ancestor.parent {
		if ancestor == child {
			return fmt.Errorf("keyvalues: cannot add %q to its own subtree", child.key)
		}
	}
	child.detach()
	kv.appendChild(child)
	return nil
}

// RemoveChild removes the first child whose key matches key, ignoring case.
// It returns the same errors as Find.
func (kv *KeyValue) RemoveChild(key string) error {
	child, err := kv.Find(key)
	if err != nil {
		return err
	}
	child.detach()
	return nil
}

// AsString returns a leaf's value. It returns ErrNotLeaf for a block.
func (kv *KeyValue) AsString() (string, error) {
	if kv.block {
		return "", fmt.Errorf("%w: %q has no value", ErrNotLeaf, kv.key)
	}
	return kv.value, nil
}

// AsInt parses a leaf's value as a base 10 int32, ignoring surrounding
// whitespace.
func (kv *KeyValue) AsInt() (int32, error) {
	s, err := kv.AsString()
	if err != nil {
		return 0, err
	}
	v, err := strconv.ParseInt(strings.TrimSpace(s), 10, 32)
	if err != nil {
		return 0, fmt.Errorf("keyvalues: %q: %w", kv.key, err)
	}
	return int32(v), nil
}

// AsFloat parses a leaf's value as a float32, ignoring surrounding whitespace.
func (kv *KeyValue) AsFloat() (float32, error) {
	s, err := kv.AsString()
	if err != nil {
		return 0, err
	}
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 32)
	if err != nil {
		return 0, fmt.Errorf("keyvalues: %q: %w", kv.key, err)
	}
	return float32(v), nil
}

// SetValue sets a leaf's value. It returns ErrNotLeaf for a block.
func (kv *KeyValue) SetValue(value string) error {
	if kv.block {
		return fmt.Errorf("%w: cannot set a value on %q", ErrNotLeaf, kv.key)
	}
	kv.value = value
	return nil
}

// appendChild adds child without the checks AddChild makes.
func (kv *KeyValue) appendChild(child *KeyValue) {
	child.parent = kv
	kv.children = append(kv.children, child)
}

// detach removes kv from its parent, if it has one.
func (kv *KeyValue) detach() {
	p := kv.parent
	if p == nil {
		return
	}
	for i, c := range p.children {
		if c == kv {
			copy(p.children[i:], p.children[i+1:])
			p.children[len(p.children)-1] = nil
			p.children = p.children[:len(p.children)-1]
			break
		}
	}
	kv.parent = nil
}

// clone returns a deep copy of kv with no parent.
func (kv *KeyValue) clone() *KeyValue {
	c := &KeyValue{key: kv.key, value: kv.value, block: kv.block, condition: kv.condition}
	for _, child := range kv.children {
		c.appendChild(child.clone())
	}
	return c
}

func findIn(nodes []*KeyValue, key string) *KeyValue {
	for _, n := range nodes {
		if strings.EqualFold(n.key, key) {
			return n
		}
	}
	return nil
}
