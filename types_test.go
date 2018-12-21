package keyvalues

import "testing"

func Test_getType(t *testing.T) {
	if getType("foo") != ValueString {
		t.Error("failed to determine string is a string")
	}
	if getType("45") != ValueInt {
		t.Error("failed to determine string is an int")
	}
	if getType("45.123") != ValueFloat {
		t.Error("failed to determine string is a float")
	}
}

func Test_isInt(t *testing.T) {
	if isInt("45") != true {
		t.Error("failed to determine string is an int")
	}
}

func Test_isFloat(t *testing.T) {
	if isFloat("45.123") != true {
		t.Error("failed to determine string is a float")
	}
}
