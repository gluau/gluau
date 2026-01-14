package vm

/*
#include "../rustlib/rustlib.h"
*/
import "C"
import (
	"errors"
	"fmt"
	"runtime"
	"unsafe"
)

// A LuaTable is an abstraction over a Lua table object.
type LuaTable struct {
	*BaseRef
}

// Clear the LuaTable
func (l *LuaTable) Clear() error {
	return withBaseRefNoRet(l.BaseRef, func(ptr C.struct_GoLuaValueV2) error {
		res := C.luago_table_clear(l.lua.ptr(), ptr)
		if res.error != nil {
			err := moveErrorToGo(res.error)
			return err
		}

		return nil
	})
}

// ContainsKey checks if the LuaTable contains a key
func (l *LuaTable) ContainsKey(key any) (bool, error) {
	return withBaseRef(l.BaseRef, func(ptr C.struct_GoLuaValueV2) (bool, error) {
		keyVal, err, _ := valueToC(key)
		if err != nil {
			return false, err // Return error if the value cannot be converted (diff lua state, closed object, etc)
		}
		res := C.luago_table_contains_key(l.lua.ptr(), ptr, keyVal)

		// ensure key is not GC'd before the call
		//
		// This is critical as key may contain a finalizer that may clean the keyVal prior to contains_key
		// finishing causing use-after-free in Rust
		runtime.KeepAlive(key)
		if res.error != nil {
			return false, moveErrorToGo(res.error)
		}
		return bool(res.value), nil
	})
}

type TableForEachFn = func(key, value any) error

// ForEach iterates over the LuaTable and calls the provided function for each key-value pair.
//
// Deadlock note: the LuaTable should not be closed while inside a ForEach loop.
// Note 2: the returned error variant should not be closed
func (l *LuaTable) ForEach(fn TableForEachFn) error {
	return withBaseRefNoRet(l.BaseRef, func(ptr C.struct_GoLuaValueV2) error {
		var errv error
		cbWrapper := newGoCallback(func(val unsafe.Pointer) {
			cval := (*C.struct_TableForEachCallbackData)(val)

			// Safety: it is undefined behavior for the callback to unwind into
			// Rust (or even C!) frames from Go, so we must recover() any panic
			// that occurs in the callback to prevent a crash.
			defer func() {
				if r := recover(); r != nil {
					errv = fmt.Errorf("panic in ForEach callback: %v", r)
					cval.stop = C.bool(true) // Stop the iteration
				}
			}()

			key := castValue(l.lua, cval.key)
			value := castValue(l.lua, cval.value)

			err := fn(key, value)
			if err != nil {
				errv = err               // Capture the error to return it later
				cval.stop = C.bool(true) // Stop the iteration
			}
		}, func() {
			fmt.Println("foreach callback is being dropped")
		})

		res := C.luago_table_foreach(l.lua.ptr(), ptr, cbWrapper.ToC())
		if res.error != nil {
			err := moveErrorToGo(res.error)
			return err
		}

		return errv
	})
}

type TableForEachValueFn = func(value any) error

// ForEachValue iterates over sequence part of the LuaTable and calls the provided function for each key-value pair.
//
// Deadlock note: the LuaTable should not be closed while inside a ForEach loop.
// Note 2: the returned error variant should not be closed
func (l *LuaTable) ForEachValue(fn TableForEachValueFn) error {
	return withBaseRefNoRet(l.BaseRef, func(ptr C.struct_GoLuaValueV2) error {
		var errv error
		cbWrapper := newGoCallback(func(val unsafe.Pointer) {
			cval := (*C.struct_TableForEachValueCallbackData)(val)

			// Safety: it is undefined behavior for the callback to unwind into
			// Rust (or even C!) frames from Go, so we must recover() any panic
			// that occurs in the callback to prevent a crash.
			defer func() {
				if r := recover(); r != nil {
					errv = fmt.Errorf("panic in ForEachValue callback: %v", r)
					cval.stop = C.bool(true) // Stop the iteration
				}
			}()

			value := castValue(l.lua, cval.value)

			err := fn(value)
			if err != nil {
				errv = err               // Capture the error to return it later
				cval.stop = C.bool(true) // Stop the iteration
			}
		}, func() {
			fmt.Println("foreachvalue callback is being dropped")
		})

		res := C.luago_table_foreach_value(l.lua.ptr(), ptr, cbWrapper.ToC())
		if res.error != nil {
			errStr := moveStringToGo(res.error)
			if errStr != "stop" {
				return errors.New(errStr)
			}
		}

		return errv
	})
}

// Get returns the value associated with the key in the LuaTable.
//
// If the key does not exist, it returns LuaValue of nil
func (l *LuaTable) Get(key any) (any, error) {
	return withBaseRef(l.BaseRef, func(ptr C.struct_GoLuaValueV2) (any, error) {
		keyVal, err, _ := valueToC(key)
		if err != nil {
			return nil, err // Return error if the value cannot be converted (diff lua state, closed object, etc)
		}
		res := C.luago_table_get(l.lua.ptr(), ptr, keyVal)
		// ensure key is not GC'd before the call
		//
		// This is critical as key may contain a finalizer that may clean the keyVal prior to get
		runtime.KeepAlive(key)
		if res.error != nil {
			return nil, moveErrorToGo(res.error)
		}
		return castValue(l.lua, res.value), nil
	})
}

// IsEmpty returns if the LuaTable is empty
//
// This method does not invoke any metamethods but may error if the table
// is closed
func (l *LuaTable) IsEmpty() bool {
	return withBaseRefDefault(l.BaseRef, func(ptr C.struct_GoLuaValueV2) bool {
		res := C.luago_table_is_empty(l.lua.ptr(), ptr)
		return bool(res)
	})
}

// IsReadonly returns if the LuaTable is marked as readonly
//
// This method does not invoke any metamethods but may error if the table
// is closed
func (l *LuaTable) IsReadonly() (bool, error) {
	return withBaseRef(l.BaseRef, func(ptr C.struct_GoLuaValueV2) (bool, error) {
		res := C.luago_table_is_readonly(l.lua.ptr(), ptr)
		return bool(res), nil
	})
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
	return withBaseRef(l.BaseRef, func(ptr C.struct_GoLuaValueV2) (int64, error) {
		res := C.luago_table_len(l.lua.ptr(), ptr)
		if res.error != nil {
			return 0, moveErrorToGo(res.error)
		}
		return int64(res.value), nil
	})
}

// Metatable returns the metatable of the LuaTable.
//
// Returns nil if the table does not have a metatable
// or is closed.
func (l *LuaTable) Metatable() *LuaTable {
	return withBaseRefDefault(l.BaseRef, func(ptr C.struct_GoLuaValueV2) *LuaTable {
		res := C.luago_table_metatable(l.lua.ptr(), ptr)
		casted := castValue(l.lua, res)
		switch v := casted.(type) {
		case nil:
			return nil // No metatable
		case *LuaTable:
			return v
		default:
			return nil // Should not happen
		}
	})
}

// Pop removes the last element from the LuaTable
//
// This might invoke the __len and __newindex metamethods.
func (l *LuaTable) Pop() (any, error) {
	return withBaseRef(l.BaseRef, func(ptr C.struct_GoLuaValueV2) (any, error) {
		res := C.luago_table_pop(l.lua.ptr(), ptr)
		if res.error != nil {
			return nil, moveErrorToGo(res.error)
		}
		return castValue(l.lua, res.value), nil
	})
}

// Push appends a value to the back of the LuaTable
//
// This might invoke the __len and __newindex metamethods.
func (l *LuaTable) Push(value any) error {
	return withBaseRefNoRet(l.BaseRef, func(ptr C.struct_GoLuaValueV2) error {
		valueVal, err, _ := valueToC(value)
		if err != nil {
			return err
		}
		res := C.luago_table_push(l.lua.ptr(), ptr, valueVal)
		runtime.KeepAlive(value) // ensure value is not GC'd before the call
		if res.error != nil {
			return moveErrorToGo(res.error)
		}
		return nil
	})
}

// Gets the value associated to key without invoking metamethods.
func (l *LuaTable) RawGet(key any) (any, error) {
	return withBaseRef(l.BaseRef, func(ptr C.struct_GoLuaValueV2) (any, error) {
		keyVal, err, _ := valueToC(key)
		if err != nil {
			return nil, err // Return error if the value cannot be converted (diff lua state, closed object, etc)
		}
		res := C.luago_table_raw_get(l.lua.ptr(), ptr, keyVal)
		// ensure key is not GC'd before the call
		//
		// This is critical as key may contain a finalizer that may clean the keyVal prior to get
		runtime.KeepAlive(key)
		if res.error != nil {
			return nil, moveErrorToGo(res.error)
		}
		return castValue(l.lua, res.value), nil
	})
}

// Inserts element value at position idx to the table, shifting up the elements from table[idx].
//
// The worst case complexity is O(n), where n is the table length.
func (l *LuaTable) RawInsert(idx int64, value any) error {
	return withBaseRefNoRet(l.BaseRef, func(ptr C.struct_GoLuaValueV2) error {
		valueVal, err, _ := valueToC(value)
		if err != nil {
			return err
		}
		res := C.luago_table_raw_insert(l.lua.ptr(), ptr, C.int64_t(idx), valueVal)
		runtime.KeepAlive(value) // ensure value is not GC'd before the call
		if res.error != nil {
			return moveErrorToGo(res.error)
		}
		return nil
	})
}

// RawLen returns the result of the Lua # operator, without invoking the __len metamethod.
//
// If the table is closed, this function returns 0.
func (l *LuaTable) RawLen() uint64 {
	return withBaseRefDefault(l.BaseRef, func(ptr C.struct_GoLuaValueV2) uint64 {
		res := C.luago_table_raw_len(l.lua.ptr(), ptr)
		return uint64(res)
	})
}

// RawPop removes the last element from the LuaTable without invoking metamethods.
func (l *LuaTable) RawPop() (any, error) {
	return withBaseRef(l.BaseRef, func(ptr C.struct_GoLuaValueV2) (any, error) {
		res := C.luago_table_raw_pop(l.lua.ptr(), ptr)
		if res.error != nil {
			return nil, moveErrorToGo(res.error)
		}
		return castValue(l.lua, res.value), nil
	})
}

// RawPush appends a value to the back of the LuaTable without invoking metamethods.
func (l *LuaTable) RawPush(value any) error {
	return withBaseRefNoRet(l.BaseRef, func(ptr C.struct_GoLuaValueV2) error {
		valueVal, err, _ := valueToC(value)
		if err != nil {
			return err
		}
		res := C.luago_table_raw_push(l.lua.ptr(), ptr, valueVal)
		runtime.KeepAlive(value) // ensure value is not GC'd before the call
		if res.error != nil {
			return moveErrorToGo(res.error)
		}
		return nil
	})
}

// RawRemove removes a key from the LuaTable without invoking metamethods.
//
// If the key is an integer, all elements from table[key+1] will be shifted down.
// and table[key] will be removed with a worst case complexity of O(n),
//
// For non-integer keys, this is equivalent to a table[key] = nil operation,
func (l *LuaTable) RawRemove(key any) error {
	return withBaseRefNoRet(l.BaseRef, func(ptr C.struct_GoLuaValueV2) error {
		keyVal, err, _ := valueToC(key)
		if err != nil {
			return err
		}
		res := C.luago_table_raw_remove(l.lua.ptr(), ptr, keyVal)
		runtime.KeepAlive(key) // ensure value is not GC'd before the call
		if res.error != nil {
			return moveErrorToGo(res.error)
		}
		return nil
	})
}

// Sets a key-value pair without invoking metamethods.
//
// If value is nil, this effectively removes the key from the table.
func (l *LuaTable) RawSet(key any, value any) error {
	return withBaseRefNoRet(l.BaseRef, func(ptr C.struct_GoLuaValueV2) error {
		keyVal, err, _ := valueToC(key)
		if err != nil {
			return err
		}
		valueVal, err, _ := valueToC(value)
		if err != nil {
			return err
		}
		res := C.luago_table_raw_set(l.lua.ptr(), ptr, keyVal, valueVal)
		runtime.KeepAlive(key)   // ensure value is not GC'd before the call
		runtime.KeepAlive(value) // ensure value is not GC'd before the call
		if res.error != nil {
			return moveErrorToGo(res.error)
		}
		return nil
	})
}

// Sets a key-value pair.
//
// If value is nil, this effectively removes the key from the table.
//
// This might invoke the __newindex metamethod if it exists.
func (l *LuaTable) Set(key any, value any) error {
	return withBaseRefNoRet(l.BaseRef, func(ptr C.struct_GoLuaValueV2) error {
		keyVal, err, _ := valueToC(key)
		if err != nil {
			return err
		}
		valueVal, err, _ := valueToC(value)
		if err != nil {
			return err
		}
		res := C.luago_table_set(l.lua.ptr(), ptr, keyVal, valueVal)
		runtime.KeepAlive(key)   // ensure value is not GC'd before the call
		runtime.KeepAlive(value) // ensure value is not GC'd before the call
		if res.error != nil {
			return moveErrorToGo(res.error)
		}
		return nil
	})
}

// Sets the metatable for the LuaTable.
//
// If the metatable is nil, it removes the metatable from the table.
func (l *LuaTable) SetMetatable(mt *LuaTable) error {
	return withBaseRefNoRet(l.BaseRef, func(ptr C.struct_GoLuaValueV2) error {
		if mt == nil {
			// Drop the metatable by passing nil as mt
			res := C.luago_table_set_metatable(l.lua.ptr(), ptr, nullCValueV2())
			if res.error != nil {
				return moveErrorToGo(res.error)
			}
			return nil
		} else {
			// Set the metatable
			return withBaseRefNoRet(mt.BaseRef, func(mt C.struct_GoLuaValueV2) error {
				res := C.luago_table_set_metatable(l.lua.ptr(), ptr, mt)
				if res.error != nil {
					err := moveErrorToGo(res.error)
					return err
				}
				return nil
			})
		}
	})
}

// SetReadonly sets whether or not the LuaTable is readonly.
//
// This is a Luau-specific feature.
//
// If the table is closed, this function does nothing.
func (l *LuaTable) SetReadonly(enabled bool) {
	withBaseRefDefault(l.BaseRef, func(ptr C.struct_GoLuaValueV2) bool {
		C.luago_table_set_readonly(l.lua.ptr(), ptr, C.bool(enabled))
		return true
	})
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
	withBaseRefDefault(l.BaseRef, func(ptr C.struct_GoLuaValueV2) bool {
		C.luago_table_set_safeenv(l.lua.ptr(), ptr, C.bool(enabled))
		return true
	})
}

// Returns a debug string representation of the LuaTable
func (l *LuaTable) String() string {
	return withBaseRefDefault(l.BaseRef, func(ptr C.struct_GoLuaValueV2) string {
		str := C.luago_table_debug(l.lua.ptr(), ptr)
		return moveStringToGo(str)
	})
}
