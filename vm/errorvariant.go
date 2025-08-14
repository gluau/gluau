package vm

import (
	"unsafe"
)

/*
#include "../rustlib/rustlib.h"
*/
import "C"

var errorVariantTab = objectTab{
	dtor: func(ptr *C.void) {
		C.luago_error_free((*C.struct_ErrorVariant)(unsafe.Pointer(ptr)))
	},
}

// A ErrorVariant is an wrapper around a Rust Arc<String> that holds an error string.
type ErrorVariant struct {
	*object
}

// Returns the ErrorVariant as a byte slice
func (l *ErrorVariant) Bytes() []byte {
	ptr, err := l.object.PointerNoLock()
	if err != nil {
		return nil // Return nil if the object is closed
	}

	data := C.luago_error_get_string((*C.struct_ErrorVariant)(unsafe.Pointer(ptr)))
	goSlice := C.GoBytes(unsafe.Pointer(data.data), C.int(data.len))
	return goSlice
}

// Returns the ErrorVariant as a string
func (l *ErrorVariant) String() string {
	bytes := l.Bytes()
	if bytes == nil {
		return "" // Return empty string if the object is closed
	}
	return string(bytes)
}

func (l *ErrorVariant) Close() {
	if l == nil || l.object == nil {
		return // Nothing to close
	}
	// Close the LuaTable object
	l.object.Close()
}
