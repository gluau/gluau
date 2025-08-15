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
		err := moveErrorToGoError(res.error)
		return err
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
		err := moveErrorToGoError(res.error)
		return err
	}

	return errv
}

type TableForEachValueFn = func(value Value) error

// ForEachValue iterates over sequence part of the LuaTable and calls the provided function for each key-value pair.
//
// Deadlock note: the LuaTable should not be closed while inside a ForEach loop.
// Note 2: the returned error variant should not be closed
func (l *LuaTable) ForEachValue(fn TableForEachValueFn) error {
	l.object.RLock()
	defer l.object.RUnlock()

	ptr, err := l.innerPtr()
	if err != nil {
		return err // Return error if the object is closed
	}

	var errv error
	cbWrapper := newGoCallback(func(val unsafe.Pointer) {
		cval := (*C.struct_TableForEachValueCallbackData)(val)
		value := l.lua.valueFromC(cval.value)

		// Safety: it is undefined behavior for the callback to unwind into
		// Rust (or even C!) frames from Go, so we must recover() any panic
		// that occurs in the callback to prevent a crash.
		defer func() {
			if r := recover(); r != nil {
				errv = fmt.Errorf("panic in ForEachValue callback: %v", r)
				cval.stop = C.bool(true) // Stop the iteration
			}
		}()

		err := fn(value)
		if err != nil {
			errv = err               // Capture the error to return it later
			cval.stop = C.bool(true) // Stop the iteration
		}
	}, func() {
		fmt.Println("foreachvalue callback is being dropped")
	})

	res := C.luago_table_foreach_value(ptr, cbWrapper.ToC())
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
//
// To avoid invoking the __len metamethod, use RawLen instead.
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
		err := moveErrorToGoError(res.error)
		return err
	}
	return nil
}

// Gets the value associated to key without invoking metamethods.
func (l *LuaTable) RawGet(key Value) (Value, error) {
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

	res := C.luago_table_raw_get(ptr, keyVal)
	if res.error != nil {
		return &ValueNil{}, moveErrorToGoError(res.error)
	}
	return l.lua.valueFromC(res.value), nil
}

// Inserts element value at position idx to the table, shifting up the elements from table[idx].
//
// The worst case complexity is O(n), where n is the table length.
func (l *LuaTable) RawInsert(idx int64, value Value) error {
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

	res := C.luago_table_raw_insert(ptr, C.int64_t(idx), valueVal)
	if res.error != nil {
		err := moveErrorToGoError(res.error)
		return err
	}
	return nil
}

// RawLen returns the result of the Lua # operator, without invoking the __len metamethod.
func (l *LuaTable) RawLen() uint64 {
	l.object.RLock()
	defer l.object.RUnlock()

	ptr, err := l.innerPtr()
	if err != nil {
		return 0 // Return 0 if the object is closed
	}

	res := C.luago_table_raw_len(ptr)
	return uint64(res)
}

// RawPop removes the last element from the LuaTable without invoking metamethods.
func (l *LuaTable) RawPop() (Value, error) {
	l.object.RLock()
	defer l.object.RUnlock()

	ptr, err := l.innerPtr()
	if err != nil {
		return &ValueNil{}, err // Return error if the object is closed
	}
	res := C.luago_table_raw_pop(ptr)
	if res.error != nil {
		return &ValueNil{}, moveErrorToGoError(res.error)
	}
	return l.lua.valueFromC(res.value), nil
}

// RawPush appends a value to the back of the LuaTable without invoking metamethods.
func (l *LuaTable) RawPush(value Value) error {
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
	res := C.luago_table_raw_push(ptr, valueVal)
	if res.error != nil {
		err := moveErrorToGoError(res.error)
		return err
	}
	return nil
}

// RawRemove removes a key from the LuaTable without invoking metamethods.
//
// If the key is an integer, all elements from table[key+1] will be shifted down.
// and table[key] will be removed with a worst case complexity of O(n),
//
// For non-integer keys, this is equivalent to a table[key] = nil operation,
func (l *LuaTable) RawRemove(key Value) error {
	l.object.RLock()
	defer l.object.RUnlock()

	ptr, err := l.innerPtr()
	if err != nil {
		return err // Return error if the object is closed
	}
	keyVal, err := l.lua.valueToC(key)
	if err != nil {
		return err // Return error if the value cannot be converted
	}
	res := C.luago_table_raw_remove(ptr, keyVal)
	if res.error != nil {
		err := moveErrorToGoError(res.error)
		return err
	}
	return nil
}

// Sets a key-value pair without invoking metamethods.
//
// If value is nil, this effectively removes the key from the table.
func (l *LuaTable) RawSet(key Value, value Value) error {
	l.object.RLock()
	defer l.object.RUnlock()

	ptr, err := l.innerPtr()
	if err != nil {
		return err // Return error if the object is closed
	}
	keyVal, err := l.lua.valueToC(key)
	if err != nil {
		return err // Return error if the value cannot be converted
	}
	valueVal, err := l.lua.valueToC(value)
	if err != nil {
		return err // Return error if the value cannot be converted
	}
	res := C.luago_table_raw_set(ptr, keyVal, valueVal)
	if res.error != nil {
		err := moveErrorToGoError(res.error)
		return err
	}
	return nil
}

// Sets a key-value pair.
//
// If value is nil, this effectively removes the key from the table.
//
// This might invoke the __newindex metamethod if it exists.
func (l *LuaTable) Set(key Value, value Value) error {
	l.object.RLock()
	defer l.object.RUnlock()

	ptr, err := l.innerPtr()
	if err != nil {
		return err // Return error if the object is closed
	}
	keyVal, err := l.lua.valueToC(key)
	if err != nil {
		return err // Return error if the value cannot be converted
	}
	valueVal, err := l.lua.valueToC(value)
	if err != nil {
		return err // Return error if the value cannot be converted
	}
	res := C.luago_table_set(ptr, keyVal, valueVal)
	if res.error != nil {
		err := moveErrorToGoError(res.error)
		return err
	}
	return nil
}

// Sets the metatable for the LuaTable.
//
// If the metatable is nil, it removes the metatable from the table.
func (l *LuaTable) SetMetatable(mt *LuaTable) error {
	l.object.RLock()
	defer l.object.RUnlock()

	ptr, err := l.innerPtr()
	if err != nil {
		return err // Return error if the object is closed
	}

	if mt == nil {
		// Drop the metatable by passing nil as mt
		res := C.luago_table_set_metatable(ptr, nil)
		if res.error != nil {
			return moveErrorToGoError(res.error)
		}
		return nil
	} else {
		// Set the metatable
		ptr2, err := mt.innerPtr()
		if err != nil {
			return err // Return error if the other object is closed
		}
		res := C.luago_table_set_metatable(ptr, ptr2)
		if res.error != nil {
			err := moveErrorToGoError(res.error)
			return err
		}
		return nil
	}
}

// SetReadonly sets whether or not the LuaTable is readonly.
//
// This is a Luau-specific feature.
//
// If the table is closed, this function does nothing.
func (l *LuaTable) SetReadonly(enabled bool) {
	l.object.RLock()
	defer l.object.RUnlock()

	ptr, err := l.innerPtr()
	if err != nil {
		return // No-op if the object is closed
	}

	C.luago_table_set_readonly(ptr, C.bool(enabled))
}

// SetSafeEnv sets whether or not the LuaTable is safeenv.
//
// Safeenv provides special performance optimizations for Lua tables
// used as the environment of a Luau chunk such as optimizing table
// accesses, fastpaths for iteration and fastpaths for fastcall optimization
// at the expense of breaking some metamethods and making the table de-facto
// readonly.
//
// This is a Luau-specific feature.
//
// If the table is closed, this function does nothing.
func (l *LuaTable) SetSafeEnv(enabled bool) {
	l.object.RLock()
	defer l.object.RUnlock()

	ptr, err := l.innerPtr()
	if err != nil {
		return // No-op if the object is closed
	}

	C.luago_table_set_safeenv(ptr, C.bool(enabled))
}

// Returns a 'pointer' to a Lua-owned table
//
// This pointer is only useful for hashing/debugging
// and cannot be converted back to the original Lua table object.
func (l *LuaTable) Pointer() uint64 {
	l.object.RLock()
	defer l.object.RUnlock()
	lptr, err := l.object.PointerNoLock()
	if err != nil {
		return 0 // Return 0 if the object is closed
	}

	ptr := C.luago_table_to_pointer((*C.struct_LuaTable)(unsafe.Pointer(lptr)))
	return uint64(ptr)
}

// Returns a debug string representation of the LuaTable
func (l *LuaTable) String() string {
	l.object.RLock()
	defer l.object.RUnlock()
	lptr, err := l.object.PointerNoLock()
	if err != nil {
		return "" // Return empty string if the object is closed
	}

	str := C.luago_table_debug((*C.struct_LuaTable)(unsafe.Pointer(lptr)))
	return moveErrorToGo(str)
}

func (l *LuaTable) Close() {
	if l == nil || l.object == nil {
		return // Nothing to close
	}
	// Close the LuaTable object
	l.object.Close()
}
