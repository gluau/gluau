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
	*object
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
	l.RLock()
	defer l.RUnlock()

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
	l.RLock()
	defer l.RUnlock()

	ptr, err := l.innerPtr()
	if err != nil {
		return false, err // Return error if the object is closed
	}
	keyVal, err := valueToC(key)
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
	l.RLock()
	defer l.RUnlock()

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
	l.RLock()
	defer l.RUnlock()

	ptr, err := l.innerPtr()
	if err != nil {
		return err // Return error if the object is closed
	}

	var errv error
	cbWrapper := newGoCallback(func(val unsafe.Pointer) {
		cval := (*C.struct_TableForEachCallbackData)(val)
		key := valueFromC(cval.key)
		value := valueFromC(cval.value)
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
		if errStr != "" && errStr != "stop" {
			return errors.New(errStr)
		}
	}

	return errv
}

func (l *LuaTable) Close() {
	if l == nil || l.object == nil {
		return // Nothing to close
	}
	// Close the LuaTable object
	l.object.Close()
}
