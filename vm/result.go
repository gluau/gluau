package vm

/*
#include "../rustlib/rustlib.h"
*/
import "C"
import "errors"

/*
struct GoNoneResult {
    char* error;
};
struct GoBoolResult {
    bool value;
    char* error;
};
struct GoStringResult {
    // Pointer to the string value
    struct LuaString* value;
    // Pointer to a null-terminated C string for the error message
    char* error;
};
struct GoTableResult {
    // Pointer to the LuaTable value
    struct LuaTable* value;
    // Pointer to a null-terminated C string for the error message
    char* error;
};
struct GoValueResult {
    // The Lua value
    struct GoLuaValue v;
    // Pointer to a null-terminated C string for the error message
    char* error;
};
*/

func moveErrorToGo(err *C.char) string {
	if err == nil {
		return ""
	}
	errStr := C.GoString(err)
	C.luago_result_error_free(err) // Free the error string
	return errStr
}

func moveErrorToGoError(err *C.char) error {
	if err == nil {
		return nil
	}
	errStr := C.GoString(err)
	C.luago_result_error_free(err) // Free the error string
	return errors.New(errStr)
}
