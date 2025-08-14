package vm

/*
#include "../rustlib/rustlib.h"
*/
import "C"
import (
	"errors"
	"fmt"
	"unsafe"
)

var tableTab = objectTab{
	dtor: func(ptr *C.void) {
		C.luago_free_table((*C.struct_LuaTable)(unsafe.Pointer(ptr)))
	},
}

// A LuaTable is an abstraction over a Lua table object.
type LuaTable struct {
	lua    *GoLuaVmWrapper // The Lua VM wrapper that owns this table
	object *object
}

func (l *LuaTable) innerPtr() (*C.struct_LuaTable, error) {
	ptr, err := l.object.PointerNoLock()
	if err != nil {
		return nil, err // Return error if the object is closed
	}
	return (*C.struct_LuaTable)(unsafe.Pointer(ptr)), nil
}

// Clear the LuaTable
func (l *LuaTable) Clear() error {
	l.object.RLock()
	defer l.object.RUnlock()

	ptr, err := l.innerPtr()
	if err != nil {
		return err // Return error if the object is closed
	}

	res := C.luago_table_clear(ptr)
	if res.error != nil {
		return moveErrorToGoError(res.error)
	}

	return nil
}

// ContainsKey checks if the LuaTable contains a key
func (l *LuaTable) ContainsKey(key Value) (bool, error) {
	l.object.RLock()
	defer l.object.RUnlock()

	ptr, err := l.innerPtr()
	if err != nil {
		return false, err // Return error if the object is closed
	}
	keyVal, err := l.lua.valueToC(key)
	if err != nil {
		return false, err // Return error if the value cannot be converted
	}

	res := C.luago_table_contains_key(ptr, keyVal)
	if res.error != nil {
		return false, moveErrorToGoError(res.error)
	}
	return bool(res.value), nil
}

// Equals checks if the LuaTable equals another LuaTable
//
// The two tables are first compared by reference. Otherwise,
// the __eq metamethod may be called to compare the two tables.
func (l *LuaTable) Equals(other *LuaTable) (bool, error) {
	l.object.RLock()
	defer l.object.RUnlock()

	ptr, err := l.innerPtr()
	if err != nil {
		return false, err // Return error if the object is closed
	}
	ptr2, err := other.innerPtr()
	if err != nil {
		return false, err // Return error if the other object is closed
	}

	res := C.luago_table_equals(ptr, ptr2)
	if res.error != nil {
		return false, moveErrorToGoError(res.error)
	}
	return bool(res.value), nil
}

type TableForEachFn = func(key, value Value) error

// ForEach iterates over the LuaTable and calls the provided function for each key-value pair.
//
// Deadlock note: the LuaTable should not be closed while inside a ForEach loop.
// Note 2: the returned error variant should not be closed
func (l *LuaTable) ForEach(fn TableForEachFn) error {
	l.object.RLock()
	defer l.object.RUnlock()

	ptr, err := l.innerPtr()
	if err != nil {
		return err // Return error if the object is closed
	}

	var errv error
	cbWrapper := newGoCallback(func(val unsafe.Pointer) {
		cval := (*C.struct_TableForEachCallbackData)(val)
		key := l.lua.valueFromC(cval.key)
		value := l.lua.valueFromC(cval.value)

		// Safety: it is undefined behavior for the callback to unwind into
		// Rust (or even C!) frames from Go, so we must recover() any panic
		// that occurs in the callback to prevent a crash.
		defer func() {
			if r := recover(); r != nil {
				errv = fmt.Errorf("panic in ForEach callback: %v", r)
				cval.stop = C.bool(true) // Stop the iteration
			}
		}()

		err := fn(key, value)
		if err != nil {
			errv = err               // Capture the error to return it later
			cval.stop = C.bool(true) // Stop the iteration
		}
	}, func() {
		fmt.Println("foreach callback is being dropped")
	})

	res := C.luago_table_foreach(ptr, cbWrapper.ToC())
	if res.error != nil {
		errStr := moveErrorToGo(res.error)
		if errStr != "stop" {
			return errors.New(errStr)
		}
	}

	return errv
}

// Get returns the value associated with the key in the LuaTable.
//
// If the key does not exist, it returns LuaValue of nil
func (l *LuaTable) Get(key Value) (Value, error) {
	l.object.RLock()
	defer l.object.RUnlock()

	ptr, err := l.innerPtr()
	if err != nil {
		return &ValueNil{}, err // Return error if the object is closed
	}
	keyVal, err := l.lua.valueToC(key)
	if err != nil {
		return &ValueNil{}, err // Return error if the value cannot be converted
	}

	res := C.luago_table_get(ptr, keyVal)
	if res.error != nil {
		return &ValueNil{}, moveErrorToGoError(res.error)
	}
	return l.lua.valueFromC(res.value), nil
}

// IsEmpty returns if the LuaTable is empty
//
// This method does not invoke any metamethods
func (l *LuaTable) IsEmpty() bool {
	l.object.RLock()
	defer l.object.RUnlock()

	ptr, err := l.innerPtr()
	if err != nil {
		return true // A closed table is considered empty
	}

	res := C.luago_table_is_empty(ptr)
	return bool(res)
}

// IsReadonly returns if the LuaTable is marked as readonly (Luau only)
func (l *LuaTable) IsReadonly() bool {
	l.object.RLock()
	defer l.object.RUnlock()

	ptr, err := l.innerPtr()
	if err != nil {
		return true // A closed table is considered readonly
	}

	res := C.luago_table_is_readonly(ptr)
	return bool(res)
}

// Len returns the length of the LuaTable
//
// This method is equivalent to the # operator in Lua
// and calls the __len metamethod if it exists.
//
// Note for those rusty with Lua: key-value pairs are not considered as part
// of the length of the table. Only array-like indices (1, 2, 3, ...) are counted.
func (l *LuaTable) Len() (int64, error) {
	l.object.RLock()
	defer l.object.RUnlock()

	ptr, err := l.innerPtr()
	if err != nil {
		return 0, err // Return error if the object is closed
	}

	res := C.luago_table_len(ptr)
	if res.error != nil {
		return 0, moveErrorToGoError(res.error)
	}
	return int64(res.value), nil
}

// Metatable returns the metatable of the LuaTable.
//
// Returns nil if the table does not have a metatable
// or is closed.
func (l *LuaTable) Metatable() *LuaTable {
	l.object.RLock()
	defer l.object.RUnlock()

	ptr, err := l.innerPtr()
	if err != nil {
		return nil
	}

	res := C.luago_table_metatable(ptr)
	if res == nil {
		return nil // No metatable or the table is closed
	}

	return &LuaTable{object: newObject((*C.void)(unsafe.Pointer(res)), tableTab), lua: l.lua}
}

// Pop removes the last element from the LuaTable
//
// This might invoke the __len and __newindex metamethods.
func (l *LuaTable) Pop() (Value, error) {
	l.object.RLock()
	defer l.object.RUnlock()

	ptr, err := l.innerPtr()
	if err != nil {
		return &ValueNil{}, err // Return error if the object is closed
	}
	res := C.luago_table_pop(ptr)
	if res.error != nil {
		return &ValueNil{}, moveErrorToGoError(res.error)
	}
	return l.lua.valueFromC(res.value), nil
}

// Push appends a value to the back of the LuaTable
//
// This might invoke the __len and __newindex metamethods.
func (l *LuaTable) Push(value Value) error {
	l.object.RLock()
	defer l.object.RUnlock()

	ptr, err := l.innerPtr()
	if err != nil {
		return err // Return error if the object is closed
	}
	valueVal, err := l.lua.valueToC(value)
	if err != nil {
		return err // Return error if the value cannot be converted
	}
	res := C.luago_table_push(ptr, valueVal)
	if res.error != nil {
		return moveErrorToGoError(res.error)
	}
	return nil
}

func (l *LuaTable) Close() {
	if l == nil || l.object == nil {
		return // Nothing to close
	}
	// Close the LuaTable object
	l.object.Close()
}
