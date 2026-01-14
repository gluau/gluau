package vm

/*
#include "../rustlib/rustlib.h"
*/
import "C"
import (
	"fmt"
	"unsafe"
)

// A LuaBuffer is an abstraction over a Lua buffer object.
type LuaBuffer struct {
	*BaseRef
}

// Returns the LuaBuffer as a byte slice
//
// Returns nil if the buffer is closed.
func (l *LuaBuffer) Bytes() []byte {
	return withBaseRefDefault(l.BaseRef, func(ptr C.struct_GoLuaValueV2) []byte {
		data := C.luago_buffer_to_bytes(l.lua.ptr(), ptr)
		bytes := moveBytesToGo(data)
		C.luago_buffer_free_bytes(data) // Free the buffer's bytes in Rust
		return bytes
	})
}

// Returns the bytes from the LuaBuffer starting at the given offset
// with the specified length.
func (l *LuaBuffer) ReadBytes(offset, len uint64) (bytes []byte, err error) {
	return withBaseRef(l.BaseRef, func(ptr C.struct_GoLuaValueV2) ([]byte, error) {
		data := C.luago_buffer_read_bytes(l.lua.ptr(), ptr, C.size_t(offset), C.size_t(len))
		bytes = moveBytesToGo(data)
		C.luago_buffer_free_bytes(data) // Free the buffer's bytes in Rust
		return bytes, nil
	})
}

// Writes data into the LuaBuffer starting at the given offset
func (l *LuaBuffer) WriteBytes(offset uint64, data []byte) error {
	if len(data) == 0 {
		return nil // No data to write, return early
	}

	return withBaseRefNoRet(l.BaseRef, func(ptr C.struct_GoLuaValueV2) error {
		size := uint64(C.luago_buffer_len(l.lua.ptr(), ptr))

		if offset > size {
			return fmt.Errorf("offset %d is out of bounds for LuaBuffer of length %d", offset, size)
		}
		if offset+uint64(len(data)) > size {
			return fmt.Errorf("data length %d exceeds LuaBuffer length %d at offset %d", len(data), size, offset)
		}

		C.luago_buffer_write_bytes(ptr, C.size_t(offset), (*C.char)(unsafe.Pointer(&data[0])), C.size_t(len(data)))
		return nil
	})
}

// Returns the LuaBuffer's length
func (l *LuaBuffer) Len() (uint64, error) {
	return withBaseRef(l.BaseRef, func(ptr C.struct_GoLuaValueV2) (uint64, error) {
		size := C.luago_buffer_len(l.lua.ptr(), ptr)
		return uint64(size), nil
	})
}

// String returns a string representation of the LuaBuffer.
//
// This is currently just the pointer address of the buffer.
func (l *LuaBuffer) String() string {
	ptr := l.Pointer()
	if ptr == 0 {
		return "<closed LuaBuffer>"
	}
	return fmt.Sprintf("LuaBuffer 0x%x", ptr)
}
