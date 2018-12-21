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

func getType(val string) ValueType {
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
	if _, err := strconv.ParseFloat(val, 32); err == nil {
		if strings.Contains(val, ".") {
			return true
		}
	}
	return false
}
