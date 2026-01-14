package vm

/*
#include "../rustlib/rustlib.h"
*/
import "C"
import (
	"fmt"
	"runtime"
)

// A LuaThread is an abstraction over a Lua thread object.
type LuaThread struct {
	*BaseRef
}

type ThreadStatus int

const (
	ThreadStatusUnknown ThreadStatus = iota
	ThreadStatusResumable
	ThreadStatusRunning
	ThreadStatusFinished
	ThreadStatusError
)

func (ts ThreadStatus) String() string {
	switch ts {
	case ThreadStatusResumable:
		return "resumable"
	case ThreadStatusRunning:
		return "running"
	case ThreadStatusFinished:
		return "finished"
	case ThreadStatusError:
		return "error"
	default:
		return "unknown"
	}
}

// Returns the current status of the LuaThread or ThreadStatusUnknown if the thread has been closed
//
// Locking behavior: This function acquires a read lock on the LuaThread object.
func (l *LuaThread) Status() ThreadStatus {
	return withBaseRefDefault(l.BaseRef, func(ptr C.struct_GoLuaValueV2) ThreadStatus {
		status := C.luago_thread_status(l.lua.ptr(), ptr)
		switch uint8(status) {
		case 0:
			return ThreadStatusResumable // Resumable
		case 1:
			return ThreadStatusRunning // Running
		case 2:
			return ThreadStatusFinished // Finished
		case 3:
			return ThreadStatusError // Error
		default:
			return ThreadStatusFinished // Default to finished for unknown statuses
		}
	})
}

// Resets the LuaThread to the initial state of a newly created Luau thread regardless
// of its current state and sets its function afterwards
//
// Locking behavior: Takes a read-lock on the LuaFunction object
// and the LuaThread object
func (l *LuaThread) Reset(fn *LuaFunction) error {
	return withBaseRefNoRet(l.BaseRef, func(ptr C.struct_GoLuaValueV2) error {
		return withBaseRefNoRet(fn.BaseRef, func(fn C.struct_GoLuaValueV2) error {
			res := C.luago_reset_thread(l.lua.ptr(), ptr, fn)
			if res.error != nil {
				err := moveErrorToGo(res.error)
				return err
			}
			return nil
		})
	})
}

// Sandboxes a Luau thread
//
// Under the hood replaces the global environment table with a new table, that performs writes locally and proxies reads to caller's global environment.
//
// This mode ideally should be used together with the global sandbox mode Lua.Sandbox.
//
// Please note that Luau links environment table with chunk when loading it into Lua state. Therefore you need to load chunks into a thread to link with the thread environment.
//
// Locking behavior: This function acquires a read lock on the LuaThread object.
func (l *LuaThread) Sandbox() error {
	return withBaseRefNoRet(l.BaseRef, func(ptr C.struct_GoLuaValueV2) error {
		res := C.luago_thread_sandbox(l.lua.ptr(), ptr)
		if res.error != nil {
			return moveErrorToGo(res.error) // Return error if sandboxing failed
		}
		return nil // Return nil if sandboxing was successful
	})
}

// Resume resumes a thread `th`
//
// Passes args as arguments to the thread. If the coroutine has called coroutine.yield, it will return these arguments. Otherwise, the coroutine wasn’t yet started, so the arguments are passed to its main function.
//
// If the thread is no longer resumable (meaning it has finished execution or encountered an error), this will return a coroutine unresumable error, otherwise will return as follows:
// If the thread is yielded via coroutine.yield or CallbackLua.YieldWith, returns the values passed to yield. If the thread returns values from its main function, returns those.
func (l *LuaThread) Resume(args ...any) ([]any, error) {
	return withBaseRef(l.BaseRef, func(ptr C.struct_GoLuaValueV2) ([]any, error) {
		mw, err := createMultiValue(args)
		if err != nil {
			return nil, err // Return error if the value cannot be converted
		}
		defer freeMultiValueArray(mw)

		res := C.luago_thread_resume(l.lua.ptr(), ptr, mw)
		runtime.KeepAlive(args) // ensure args are not GC'd before the call
		if res.error != nil {
			return nil, moveErrorToGo(res.error)
		}
		return copyAndFreeMultiValueArray(l.lua, res.value), nil
	})
}

// ResumeError resumes a thread `th` with an error
//
// Similar to Resume, but allows the resume to throw an error into the thread.
//
// This is a Luau specific extension
func (l *LuaThread) ResumeError(errorValue any) ([]any, error) {
	return withBaseRef(l.BaseRef, func(ptr C.struct_GoLuaValueV2) ([]any, error) {
		errorValueC, err, ref := valueToC(errorValue)
		if ref != nil {
			// Ensure error value is not closed
			if ref.closed.Load() {
				return nil, fmt.Errorf("cannot resume thread with closed value")
			}
		}
		if err != nil {
			return nil, err // Return error if the value cannot be converted
		}

		res := C.luago_thread_resume_error(l.lua.ptr(), ptr, errorValueC)
		runtime.KeepAlive(errorValue) // ensure args are not GC'd before the call
		if res.error != nil {
			return nil, moveErrorToGo(res.error)
		}
		return copyAndFreeMultiValueArray(l.lua, res.value), nil
	})
}

// String returns a string representation of the LuaThread.
func (l *LuaThread) String() string {
	status := l.Status()
	return "LuaThread(status: " + status.String() + ", pointer: " + fmt.Sprintf("%#x", l.Pointer()) + ")"
}
