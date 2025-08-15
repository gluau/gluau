package vm

/*
#include "../rustlib/rustlib.h"
*/
import "C"
import "errors"

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
