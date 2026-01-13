package vmutils

import (
	"errors"
	"strconv"

	"github.com/koeng101/gluau/vm"
)

// A ValueSet is a collection of vm.Value objects
// that can be ergonomically accessed and closed at the same time.
type ValueSet struct {
	values []vm.Value
}

func NewValueSet(values []vm.Value) *ValueSet {
	return &ValueSet{values: values}
}

// Returns the values in the ValueSet.
func (vs *ValueSet) Values() []vm.Value {
	return vs.values
}

// Casts a value at the given index to a vm.Value
// Returns a (lua-like) error if the index is out of bounds.
func (vs *ValueSet) ValueAt(index int) (vm.Value, error) {
	if index < 0 || index >= len(vs.values) {
		return nil, errors.New("expected at least " + strconv.Itoa(index+1) + " values but got " + strconv.Itoa(len(vs.values)) + " values")
	}
	return vs.values[index], nil
}

// Internal helper function to create a error message for type mismatches
func TypeMismatchError(index int, expectedType string, actualType string) error {
	return errors.New("expected arg #" + strconv.Itoa(index) + " to be a " + expectedType + ", but got " + actualType)
}

// Casts a value at the given index to a nil value
//
// Returns a (lua-like) error if the index is out of bounds or if the value at that index
// is not a nil value.
func (vs *ValueSet) NilAt(index int) (*vm.ValueNil, error) {
	value, err := vs.ValueAt(index)
	if err != nil {
		return nil, err
	}

	switch v := value.(type) {
	case *vm.ValueNil:
		return v, nil
	default:
		return nil, TypeMismatchError(index, "nil", v.Type().String())
	}
}

// Casts a value at the given index to a boolean
//
// Returns a (lua-like) error if the index is out of bounds or if the value at that index
// is not a boolean.
func (vs *ValueSet) BoolAt(index int) (bool, error) {
	value, err := vs.ValueAt(index)
	if err != nil {
		return false, err
	}

	switch v := value.(type) {
	case *vm.ValueBoolean:
		return v.Value(), nil
	default:
		return false, TypeMismatchError(index, "boolean", v.Type().String())
	}
}

// Casts a value at the given index to a integer
// Returns a (lua-like) error if the index is out of bounds or if the value at that index
func (vs *ValueSet) IntegerAt(index int) (int64, error) {
	value, err := vs.ValueAt(index)
	if err != nil {
		return 0, err
	}

	switch v := value.(type) {
	case *vm.ValueInteger:
		return v.Value(), nil
	case *vm.ValueNumber:
		return int64(v.Value()), nil
	default:
		return 0, TypeMismatchError(index, "integer", v.Type().String())
	}
}

// Casts a value at the given index to a integer or number
//
// Returns a (lua-like) error if the index is out of bounds or if the value at that index
// is not a integer/number.
func (vs *ValueSet) NumberAt(index int) (float64, error) {
	value, err := vs.ValueAt(index)
	if err != nil {
		return 0, err
	}

	switch v := value.(type) {
	case *vm.ValueNumber:
		return v.Value(), nil
	case *vm.ValueInteger:
		return float64(v.Value()), nil
	default:
		return 0, TypeMismatchError(index, "number", v.Type().String())
	}
}

// Casts a value at the given index to a vector
// Returns a (lua-like) error if the index is out of bounds or if the value at that index
// is not a vector.
func (vs *ValueSet) VectorAt(index int) ([3]float32, error) {
	value, err := vs.ValueAt(index)
	if err != nil {
		return [3]float32{}, err
	}

	switch v := value.(type) {
	case *vm.ValueVector:
		return v.Value(), nil
	default:
		return [3]float32{}, TypeMismatchError(index, "vector", v.Type().String())
	}
}

// Casts a value at the given index to a string
//
// Returns a (lua-like) error if the index is out of bounds or if the value at that index
// is not a string.
func (vs *ValueSet) StringAt(index int) (string, error) {
	value, err := vs.ValueAt(index)
	if err != nil {
		return "", err
	}

	switch v := value.(type) {
	case *vm.ValueString:
		return v.Value().String(), nil
	default:
		return "", TypeMismatchError(index, "string", v.Type().String())
	}
}

// Casts a value at the given index to a table
//
// Returns a (lua-like) error if the index is out of bounds or if the value at that index
// is not a table.
func (vs *ValueSet) TableAt(index int) (*vm.LuaTable, error) {
	value, err := vs.ValueAt(index)
	if err != nil {
		return nil, err
	}

	switch v := value.(type) {
	case *vm.ValueTable:
		return v.Value(), nil
	default:
		return nil, TypeMismatchError(index, "table", v.Type().String())
	}
}

// Casts a value at the given index to a function
//
// Returns a (lua-like) error if the index is out of bounds or if the value at that index
// is not a function.
func (vs *ValueSet) FunctionAt(index int) (*vm.LuaFunction, error) {
	value, err := vs.ValueAt(index)
	if err != nil {
		return nil, err
	}

	switch v := value.(type) {
	case *vm.ValueFunction:
		return v.Value(), nil
	default:
		return nil, TypeMismatchError(index, "function", v.Type().String())
	}
}

// Casts a value at the given index to a userdata
//
// Returns a (lua-like) error if the index is out of bounds or if the value at that index
// is not a userdata.
func (vs *ValueSet) UserdataAt(index int) (*vm.LuaUserData, error) {
	value, err := vs.ValueAt(index)
	if err != nil {
		return nil, err
	}

	switch v := value.(type) {
	case *vm.ValueUserData:
		return v.Value(), nil
	default:
		return nil, TypeMismatchError(index, "userdata", v.Type().String())
	}
}

// Pushes a nil value to the ValueSet
func (vs *ValueSet) PushNil() {
	vs.values = append(vs.values, vm.NewValueNil())
}

// Pushes a boolean value to the ValueSet
func (vs *ValueSet) PushBool(value bool) {
	vs.values = append(vs.values, vm.NewValueBoolean(value))
}

// Pushes an integer value to the ValueSet
func (vs *ValueSet) PushInteger(value int64) {
	vs.values = append(vs.values, vm.NewValueInteger(value))
}

// Pushes a number value to the ValueSet
func (vs *ValueSet) PushNumber(value float64) {
	vs.values = append(vs.values, vm.NewValueNumber(value))
}

// Pushes a vector value to the ValueSet
func (vs *ValueSet) PushVector(x float32, y float32, z float32) {
	vs.values = append(vs.values, vm.NewValueVector(x, y, z))
}

// Pushes a vector value to the ValueSet
// as a [3]float32 array.
func (vs *ValueSet) PushVectorArray(value [3]float32) {
	vs.values = append(vs.values, vm.NewValueVector(value[0], value[1], value[2]))
}

// Pushes a lua owned string value to the ValueSet
func (vs *ValueSet) PushLuaString(value *vm.LuaString) {
	vs.values = append(vs.values, value.ToValue())
}

// Pushes a string value to the ValueSet
//
// The passed string will then be lazily/automatically converted to a lua string
// when sent to the Lua VM.
func (vs *ValueSet) PushString(value string) {
	vs.values = append(vs.values, vm.GoString(value))
}

// Pushes a LuaTable value to the ValueSet
func (vs *ValueSet) PushTable(value *vm.LuaTable) {
	vs.values = append(vs.values, value.ToValue())
}

// Pushes a LuaFunction value to the ValueSet
func (vs *ValueSet) PushFunction(value *vm.LuaFunction) {
	vs.values = append(vs.values, value.ToValue())
}

// Pushes a LuaThread value to the ValueSet
func (vs *ValueSet) PushThread(value *vm.LuaThread) {
	vs.values = append(vs.values, value.ToValue())
}

// Pushes a LuaUserData value to the ValueSet
func (vs *ValueSet) PushUserData(value *vm.LuaUserData) {
	vs.values = append(vs.values, value.ToValue())
}

// Closes all vm.Value objects in the ValueSet.
func (vs *ValueSet) Close() {
	for _, v := range vs.values {
		if v != nil {
			v.Close()
		}
	}
}
