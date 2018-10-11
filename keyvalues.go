package keyvalues

import (
	"strconv"
	"strings"
)

type ValueType string

const ValueString = ValueType("string")
const ValueInt = ValueType("integer")
const ValueArray = ValueType("array")
const ValuePtr = ValueType("ptr")
const ValueFloat = ValueType("float")

var typeList = [5]ValueType{
	ValueString,
	ValueInt,
	ValueArray,
	ValuePtr,
	ValueFloat,
}

// A KeyValue object, that may hold multiple Values
type KeyValue struct {
	key string
	valueType ValueType
	value []interface{}
}

// Get node key
func (node *KeyValue) GetKey() *string {
	return &node.key
}

// Check if KeyValue has a property defined
func (node *KeyValue) HasKey(name string) bool {
	v := node.FindByKey(name)
	return v != nil
}

// Get node key
func (node *KeyValue) GetType() *ValueType {
	return &node.valueType
}

// Get all node values
func (node *KeyValue) GetAllValues() *[]interface{} {
	return &node.value
}

// Find a a keyvalue pair
func (node *KeyValue) FindByKey(key string) *KeyValue {
	for _,child := range node.value {
		n,_ := child.(KeyValue)
		if n.key == key {
			return &n
		}
	}
	return nil
}

// Find a value of a key, when the type is a string
func (node *KeyValue) FindStringByKey(key string) string {
	for _,child := range node.value {
		n,_ := child.(KeyValue)
		if n.key == key {
			if n.valueType != ValueString {
				return ""
			}
			return (n.value[0]).(string)
		}
	}
	return ""
}

// Find a value of a key, when the type is an integer
func (node *KeyValue) FindIntegerByKey(key string) int32 {
	for _,child := range node.value {
		n,_ := child.(KeyValue)
		if n.key == key {
			if n.valueType != ValueInt {
				return -1
			}
			return (n.value[0]).(int32)
		}
	}
	return -1
}

// Find a value of a key, when the type is an float
func (node *KeyValue) FindFloatByKey(key string) float32 {
	for _,child := range node.value {
		n,_ := child.(KeyValue)
		if n.key == key {
			if n.valueType != ValueFloat {
				return -1
			}
			return (n.value[0]).(float32)
		}
	}
	return -1
}

// Return all children of a given type for a node.
// This is different from properties, as a property is a string:<primitive>
func (node *KeyValue) FindArrayByKey(key string) (children []KeyValue) {
	for _,child := range node.value {
		n,_ := child.(KeyValue)
		if n.key == key {
			children = append(children, n)
		}
	}

	return children
}

func getType(val string) ValueType{
	switch true {
	case isFloat(val):
		return ValueFloat
	case isInt(val):
		return ValueInt
	default:
		return ValueString
	}
}

func isInt(val string) bool {
	if _, err := strconv.Atoi(val); err == nil {
		return true
	}
	return false
}

func isFloat(val string) bool {
	if _, err := strconv.Atoi(val); err == nil {
		if strings.Contains(val, ".") {
			return true
		}
	}
	return false
}