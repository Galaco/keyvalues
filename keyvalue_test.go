package keyvalues

import (
	"testing"
)

func TestKeyValue_Key(t *testing.T) {
	key := "foo"
	kv := KeyValue{
		key:       "foo",
		valueType: ValueString,
		value:     nil,
	}

	if kv.Key() != key {
		t.Error("returned key does not match expected")
	}
}

func TestKeyValue_Type(t *testing.T) {
	valueType := ValueString
	kv := KeyValue{
		key:       "foo",
		valueType: ValueString,
		value:     nil,
	}

	if kv.Type() != valueType {
		t.Error("returned value type does not match expected")
	}
}

func TestKeyValue_HasChildren(t *testing.T) {
	kv := KeyValue{
		key:       "foo",
		valueType: ValueArray,
		value:     nil,
	}

	if kv.HasChildren() != true {
		t.Error("returned value type does not match expected")
	}

	kv.valueType = ValueInt
	if kv.HasChildren() != false {
		t.Error("returned value type does not match expected")
	}
}

func TestKeyValue_Find(t *testing.T) {
	kv := KeyValue{
		key:       "foo",
		valueType: ValueArray,
		value: append(make([]interface{}, 0),
			&KeyValue{
				key:       "bar",
				valueType: ValueString,
				value:     append(make([]interface{}, 0), "hello"),
			},
			&KeyValue{
				key:       "baz",
				valueType: ValueInt,
				value:     append(make([]interface{}, 0), 123),
			}),
	}

	if val, err := kv.Find("bar"); err != nil || val == nil {
		t.Error(err)
		if val != nil {
			if val.Key() != "bar" {
				t.Error("unexpected keyvalue returned")
			}
		}
	}
}

func TestKeyValue_FindAll(t *testing.T) {
	kv := KeyValue{
		key:       "foo",
		valueType: ValueArray,
		value: append(make([]interface{}, 0),
			&KeyValue{
				key:       "bar",
				valueType: ValueString,
				value:     append(make([]interface{}, 0), "hello"),
			},
			&KeyValue{
				key:       "baz",
				valueType: ValueInt,
				value:     append(make([]interface{}, 0), 123),
			},
			&KeyValue{
				key:       "bar",
				valueType: ValueFloat,
				value:     append(make([]interface{}, 0), 345631.12312),
			}),
	}

	if vals, err := kv.FindAll("bar"); err != nil || vals == nil {
		t.Error(err)
		if len(vals) != 2 {
			t.Error("unexpected number of keyvalues found")
		} else {
			if v, _ := vals[0].AsString(); v != "hello" {
				t.Error("unexpected keyvalue returned")
			}
			if v, _ := vals[1].AsFloat(); v != 345631.12312 {
				t.Error("unexpected keyvalue returned")
			}
		}
	}
}

func TestKeyValue_Children(t *testing.T) {

}

func TestKeyValue_AsString(t *testing.T) {
	value := "ika!"
	kv := KeyValue{
		key:       "foo",
		valueType: ValueString,
		value:     append(make([]interface{}, 0), value),
	}
	if val, err := kv.AsString(); err != nil || val != value {
		if err != nil {
			t.Error(err)
		} else {
			t.Error("returned value type does not match expected")
		}
	}
}

func TestKeyValue_AsInt(t *testing.T) {
	value := int32(3456123)
	kv := KeyValue{
		key:       "foo",
		valueType: ValueInt,
		value:     append(make([]interface{}, 0), value),
	}
	if val, err := kv.AsInt(); err != nil || val != value {
		if err != nil {
			t.Error(err)
		} else {
			t.Error("returned value type does not match expected")
		}
	}
}

func TestKeyValue_AsFloat(t *testing.T) {
	value := float32(23424.1233123)
	kv := KeyValue{
		key:       "foo",
		valueType: ValueFloat,
		value:     append(make([]interface{}, 0), value),
	}
	if val, err := kv.AsFloat(); err != nil || val != value {
		if err != nil {
			t.Error(err)
		} else {
			t.Error("returned value type does not match expected")
		}
	}
}

func TestKeyValue_MergeInto(t *testing.T) {
	a := &KeyValue{
		key: "foo",
		valueType: ValueArray,
		value: []interface{}{
			&KeyValue{
				key: "bar",
				valueType: ValueString,
				value: []interface{}{
					"bar",
				},
			},
			&KeyValue{
				key: "baz",
				valueType: ValueString,
				value: []interface{}{
					"baz",
				},
			},
			&KeyValue{
				key: "bat",
				valueType: ValueString,
				value: []interface{}{
					"bat",
				},
			},
		},
	}
	b := &KeyValue{
		key: "foo",
		valueType: ValueArray,
		value: []interface{}{
			&KeyValue{
				key: "bar",
				valueType: ValueString,
				value: []interface{}{
					"cart",
				},
			},
			&KeyValue{
				key: "egg",
				valueType: ValueString,
				value: []interface{}{
					"bat",
				},
			},
		},
	}

	result,err := a.MergeInto(b)
	if err != nil {
		t.Error(err)
	}

	actual,err := result.Find("bar")
	if actual == nil {
		t.Error(err)
	}
	actualVal,err := actual.AsString()
	if actualVal != "bar" {
		if actualVal == "cart" {
			t.Error("keyvalue was not overwritten during merge")
		} else {
			t.Errorf("unexpected value for associated key. expected bar, received: %s", actualVal)
		}
	}
	actual,err = result.Find("baz")
	if actual == nil {
		t.Error(err)
	}
	actual,err = result.Find("bat")
	if actual == nil {
		t.Error(err)
	}
	actual,err = result.Find("egg")
	if actual == nil {
		t.Error(err)
	}
}
