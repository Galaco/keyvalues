package keyvalues

import (
	"errors"
	"strings"
)

// A KeyValue object, that may hold multiple Values
type KeyValue struct {
	key       string
	valueType ValueType
	value     []interface{}
}

// key is the identifier for a stored value
// it should be unique, and keys are not case sensitive, yet seem
// to be inconsistent all over the place for file formats based on
// the kv format ( e.g. GameInfo.txt has inconsistent casing)

// Key returns KeyValues's key
func (node *KeyValue) Key() string {
	return node.key
}

// Type returns type of this key's value
func (node *KeyValue) Type() ValueType {
	return node.valueType
}

// Find returns a keyvalue pair where the key matches input
// It will return the first found KeyValue in cases where the key is defined
// multiple times
func (node *KeyValue) Find(key string) (*KeyValue, error) {
	children, err := node.FindAll(key)
	if err != nil {
		return nil, err
	}

	return children[0], err
}

// FindAll returns all children of a given type for a node.
// This is different from properties, as a property is a string:<primitive>
// This will return an array of all KeyValues that match a given
// key, even though there should be only one.
func (node *KeyValue) FindAll(key string) (children []*KeyValue, err error) {
	searchKey := strings.ToLower(key)
	for idx := range node.value {
		n, _ := node.value[idx].(*KeyValue)
		if strings.ToLower(n.key) == searchKey {
			children = append(children, n)
		}
	}
	if len(children) == 0 {
		return nil, errors.New("could not find key: " + key)
	}

	return children, nil
}

// HasChildren returns if this Key has KeyValues as its own
// value.
func (node *KeyValue) HasChildren() bool {
	return node.Type() == ValueArray
}

// GetChildren gets all node child values
// This is used for keys that contain 1 or more children as its value
// rather than a basic type
func (node *KeyValue) Children() (children []*KeyValue, err error) {
	if !node.HasChildren() {
		return nil, errors.New("keyvalue has no children")
	}
	for idx := range node.value {
		n, _ := node.value[idx].(*KeyValue)
		children = append(children, n)
	}
	return children, nil
}

// AsString returns value as a string, assuming it is of string type
func (node *KeyValue) AsString() (string, error) {
	if node.valueType != ValueString {
		return "", errors.New("value is not of type string")
	}
	return (node.value[0]).(string), nil
}

// AsInt returns value as an int32, assuming it is of integer type
func (node *KeyValue) AsInt() (int32, error) {
	if node.valueType != ValueInt {
		return -1, errors.New("value is not of type integer")
	}
	return (node.value[0]).(int32), nil
}

// AsFloat returns value as an int32, assuming it is of float type
func (node *KeyValue) AsFloat() (float32, error) {
	if node.valueType != ValueFloat {
		return -1, errors.New("value is not of type float")
	}
	return (node.value[0]).(float32), nil
}

// Add adds a new KeyValue pair to an existing Key
// Existing key's value must be an Array type
func (node *KeyValue) AddChild(value *KeyValue) error {
	if !node.HasChildren() {
		return errors.New("parent does not accept child keys")
	}
	node.value = append(node.value, value)
	return nil
}
