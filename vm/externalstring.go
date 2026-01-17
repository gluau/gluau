package vm

/*
#include "../rustlib/rustlib.h"
*/
import "C"

// Creates a GoOwnedBytes struct from a byte slice
func createGoOwnedBytes(data []byte) C.struct_GoOwnedBytes {
	if len(data) == 0 {
		return C.struct_GoOwnedBytes{
			data: nil,
			size: 0,
		}
	}
	return C.struct_GoOwnedBytes{
		data: (*C.uint8_t)(&data[0]),
		size: C.size_t(len(data)),
	}
}
