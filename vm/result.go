package vm

/*
#include "../rustlib/rustlib.h"
*/
import "C"
import (
	"bytes"
	"errors"
	"unsafe"
)

func moveStringToRust(s string) *C.char {
	if len(s) == 0 {
		return C.luago_string_new(nil, 0) // Return empty char array if the string is empty
	}
	return moveBytesToRust([]byte(s))
}

func moveBytesToRust(b []byte) *C.char {
	if b == nil {
		return C.luago_string_new(nil, 0) // Return empty char array if the byte slice is nil
	}
	b = bytes.ReplaceAll(b, []byte{0}, []byte{}) // Remove any null bytes from the byte slice
	return C.luago_string_new((*C.char)(unsafe.Pointer(&b[0])), C.size_t(len(b)))
}

func freeRustString(s *C.char) {
	if s == nil {
		return // Nothing to free
	}
	C.luago_string_free(s) // Free the Rust string
}

func moveStringToGo(err *C.char) string {
	if err == nil {
		return ""
	}
	errStr := C.GoString(err)
	C.luago_string_free(err) // Free the string
	return errStr
}

func moveErrorToGo(err *C.char) error {
	if err == nil {
		return nil
	}
	errStr := C.GoString(err)
	C.luago_string_free(err) // Free the error string
	return errors.New(errStr)
}
