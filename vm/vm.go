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

var luaVmTab = objectTab{
	dtor: func(ptr *C.void) {
		C.freeluavm((*C.struct_Lua)(unsafe.Pointer(ptr)))
	},
}

// A handle to the Lua VM.
type Lua struct {
	object *object
}

// Returns the string representation of the Lua VM.
func (l *Lua) String() string {
	if l == nil || l.object == nil {
		return "<nil Lua VM>"
	}
	pt := l.MainThread().Pointer()
	if pt == 0 {
		return "<closed Lua VM>"
	}
	return fmt.Sprintf("Lua VM: 0x%x", pt)
}

func (l *Lua) lua() (*C.struct_Lua, error) {
	ptr, err := l.object.PointerNoLock()
	if err != nil {
		return nil, err // Return error if the object is closed
	}
	return (*C.struct_Lua)(unsafe.Pointer(ptr)), nil
}

// SetCompilerOpts sets the default compiler options for the Lua VM.
//
// This is a Luau-specific feature
func (l *Lua) SetCompilerOpts(opts CompilerOpts) {
	l.object.RLock()
	defer l.object.RUnlock()

	lua, err := l.lua()
	if err != nil {
		return // No-op if the Lua VM is closed
	}

	cOpts := opts.toC()
	C.luavm_setcompileropts(lua, cOpts)
}

// SetMemoryLimit sets the memory limit for the Lua VM.
//
// Upon exceeding this limit, Luau will return a memory error
// back to the caller (which may either be in Luau still or in Go
// as a error value).
func (l *Lua) SetMemoryLimit(limit int) error {
	l.object.RLock()
	defer l.object.RUnlock()

	lua, err := l.lua()
	if err != nil {
		return err
	}
	res := C.luavm_setmemorylimit(lua, C.size_t(limit))
	if res.error != nil {
		err := moveErrorToGo(res.error)
		return err
	}
	return nil
}

// SetInstructionLimit sets an instruction limit for the Lua VM.
//
// The VM will stop execution with an error when the limit is exceeded.
// This is useful for preventing infinite loops in untrusted code.
// The counting happens in Rust with no CGO overhead per instruction.
func (l *Lua) SetInstructionLimit(limit uint64) error {
	l.object.RLock()
	defer l.object.RUnlock()

	lua, err := l.lua()
	if err != nil {
		return err
	}
	res := C.luago_set_instruction_limit(lua, C.uint64_t(limit))
	if res.error != nil {
		return moveErrorToGo(res.error)
	}
	return nil
}

// CancellationToken allows cancelling Lua execution from another goroutine.
// Must be closed with Close() when no longer needed.
type CancellationToken struct {
	ptr *C.struct_CancellationToken
}

// SetInstructionLimitWithCancel sets an instruction limit and returns a CancellationToken.
//
// The token can be used to cancel execution from another goroutine by calling Cancel().
// This provides both instruction limiting and timeout-based cancellation with graceful
// termination at the next interrupt point (loop iteration or function call).
//
// The token must be closed with Close() when no longer needed.
func (l *Lua) SetInstructionLimitWithCancel(limit uint64) (*CancellationToken, error) {
	l.object.RLock()
	defer l.object.RUnlock()

	lua, err := l.lua()
	if err != nil {
		return nil, err
	}
	ptr := C.luago_set_instruction_limit_with_cancel(lua, C.uint64_t(limit))
	if ptr == nil {
		return nil, fmt.Errorf("failed to set instruction limit with cancel")
	}
	return &CancellationToken{ptr: ptr}, nil
}

// Cancel signals the Lua VM to stop execution at the next interrupt point.
// This is safe to call from another goroutine.
func (t *CancellationToken) Cancel() {
	if t == nil || t.ptr == nil {
		return
	}
	C.luago_cancel_execution(t.ptr)
}

// Close frees the cancellation token. Must be called when done.
func (t *CancellationToken) Close() {
	if t == nil || t.ptr == nil {
		return
	}
	C.luago_free_cancellation_token(t.ptr)
	t.ptr = nil
}

// UsedMemory returns the amount of memory used by the Lua VM.
func (l *Lua) UsedMemory() int {
	l.object.RLock()
	defer l.object.RUnlock()

	lua, err := l.lua()
	if err != nil {
		return 0 // Return 0 if the Lua VM is closed
	}

	used := C.luago_used_memory(lua)
	return int(used)
}

// MemoryLimit returns the memory limit set for the Lua VM.
func (l *Lua) MemoryLimit() int {
	l.object.RLock()
	defer l.object.RUnlock()

	lua, err := l.lua()
	if err != nil {
		return 0 // Return 0 if the Lua VM is closed
	}

	limit := C.luago_memory_limit(lua)
	return int(limit)
}

type TypeMetatableType uint8

const (
	TypeMetatableTypeBool TypeMetatableType = iota
	TypeMetatableTypeLightUserData
	TypeMetatableTypeNumber
	TypeMetatableTypeVector
	TypeMetatableTypeString
	TypeMetatableTypeFunction
	TypeMetatableTypeThread
	TypeMetatableTypeBuffer
)

// SetTypeMetatable sets the metatable for a Lua builtin type.
//
// - The metatable will be shared by all values of the given type.
// - mt can be set to nil to remove the metatable
func (l *Lua) SetTypeMetatable(typ TypeMetatableType, mt *LuaTable) error {
	var mtPtr *C.struct_LuaTable = nil
	if mt != nil {
		if mt.lua != l {
			return fmt.Errorf("cannot create userdata with metatable from different Lua instance")
		}

		defer mt.object.RUnlock()
		mt.object.RLock()
		metaPtr, err := mt.innerPtr()
		if err != nil {
			return err // Return error if the metatable is closed
		}
		mtPtr = metaPtr
	}

	l.object.RLock()
	defer l.object.RUnlock()

	lua, err := l.lua()
	if err != nil {
		return err
	}

	C.luago_set_type_metatable(lua, C.uint8_t(typ), mtPtr)
	return nil
}

// SetRegistryValue sets a value on the Lua registry with a given key name
func (l *Lua) SetRegistryValue(key string, value Value) error {
	l.object.RLock()
	defer l.object.RUnlock()

	lua, err := l.lua()
	if err != nil {
		return err
	}

	valueVal, err := l.valueToC(value)
	if err != nil {
		return err // Return error if the value cannot be converted (diff lua state, closed object, etc)
	}

	if len(key) == 0 {
		res := C.luago_set_named_registry_value(lua, (*C.char)(nil), 0, valueVal)
		if res.error != nil {
			err := moveErrorToGo(res.error)
			return err
		}
		return nil
	}
	keyBytes := []byte(key)
	res := C.luago_set_named_registry_value(lua, (*C.char)(unsafe.Pointer(&keyBytes[0])), C.size_t(len(key)), valueVal)
	if res.error != nil {
		err := moveErrorToGo(res.error)
		return err
	}
	return nil
}

// RegistryValue returns a value on the Lua registry with a given key name
func (l *Lua) RegistryValue(key string) (Value, error) {
	l.object.RLock()
	defer l.object.RUnlock()

	lua, err := l.lua()
	if err != nil {
		return nil, err
	}

	if len(key) == 0 {
		res := C.luago_named_registry_value(lua, (*C.char)(nil), 0)
		if res.error != nil {
			err := moveErrorToGo(res.error)
			return nil, err
		}
		return l.valueFromC(res.value), nil
	}
	keyBytes := []byte(key)
	res := C.luago_named_registry_value(lua, (*C.char)(unsafe.Pointer(&keyBytes[0])), C.size_t(len(key)))
	if res.error != nil {
		err := moveErrorToGo(res.error)
		return nil, err
	}
	return l.valueFromC(res.value), nil
}

// RemoveRegistryValue removes a value on the Lua registry with a given key name
//
// Equivalent to SetRegistryValue with value of nil
func (l *Lua) RemoveRegistryValue(key string) error {
	return l.SetRegistryValue(key, &ValueNil{})
}

// Sandbox enables or disables the sandbox mode for the Luau VM.
//
// This method, in particular:
//
// - Set all libraries to read-only
// - Set all builtin metatables to read-only
// - Set globals to read-only (and activates safeenv)
// - Setup local environment table that performs writes locally and proxies reads to the global environment.
// - Allow only count mode in collectgarbage function.
//
// Note that this is a Luau-specific feature.
func (l *Lua) Sandbox(enabled bool) error {
	l.object.RLock()
	defer l.object.RUnlock()

	lua, err := l.lua()
	if err != nil {
		return err
	}
	res := C.luavm_sandbox(lua, C.bool(enabled))
	if res.error != nil {
		err := moveErrorToGo(res.error)
		return err
	}
	return nil
}

// Globals returns the global environment table of the Lua VM.
func (l *Lua) Globals() *LuaTable {
	l.object.RLock()
	defer l.object.RUnlock()

	lua, err := l.lua()
	if err != nil {
		return nil
	}
	globals := C.luago_globals(lua)
	if globals == nil {
		return nil // Return nil if the globals table is not available
	}
	return &LuaTable{object: newObject((*C.void)(unsafe.Pointer(globals)), tableTab), lua: l}
}

// SetGlobals sets the global environment table of the Lua VM.
//
// Note that any existing Lua functions have cached global environment and will not see the changes made by this method.
//
// To update the environment for existing Lua functions, use LuaFunction.SetEnvironment
func (l *Lua) SetGlobals(tab *LuaTable) error {
	if tab.lua != l {
		return fmt.Errorf("cannot set globals from different Lua instance")
	}

	l.object.RLock()
	defer l.object.RUnlock()

	lua, err := l.lua()
	if err != nil {
		return nil
	}
	if tab == nil {
		return errors.New("globals table cannot be nil")
	}
	defer tab.object.RUnlock()
	tab.object.RLock()
	tabPtr, err := tab.innerPtr()
	if err != nil {
		return err // Return error if the table is closed
	}
	res := C.luago_setglobals(lua, tabPtr)
	if res.error != nil {
		err := moveErrorToGo(res.error)
		return err
	}
	return nil
}

type VmState int

const (
	VmStateContinue VmState = iota
	VmStateYield            // Yield the VM execution / stop execution
)

type InterruptFn func(funcVm *CallbackLua) (VmState, error)

// Sets an interrupt function that will periodically be called by Luau VM.
//
// Any Luau code is guaranteed to call this handler “eventually” (in practice this can happen at any function call or at any loop iteration).
//
// The provided interrupt function can error, and this error will be propagated through the Luau code that was executing at the time the interrupt was triggered.
//
// Also this can be used to implement continuous execution limits by instructing Luau VM to yield by returning VmState::Yield.
func (l *Lua) SetInterrupt(callback InterruptFn) {
	l.object.RLock()
	defer l.object.RUnlock()

	lua, err := l.lua()
	if err != nil {
		return
	}

	cbWrapper := newGoCallback(func(val unsafe.Pointer) {
		cval := (*C.struct_InterruptData)(val)

		// Safety: it is undefined behavior for the callback to unwind into
		// Rust (or even C!) frames from Go, so we must recover() any panic
		// that occurs in the callback to prevent a crash.
		defer func() {
			if r := recover(); r != nil {
				// Deallocate any existing error
				if cval.error != nil {
					freeRustString(cval.error)
				}

				// Replace
				errv := moveStringToRust(fmt.Sprintf("panic in interrupt callback: %v", r))
				cval.error = errv // Rust side will deallocate it for us
			}
		}()

		callbackVm := &Lua{object: newObject((*C.void)(unsafe.Pointer(cval.lua)), luaVmTab)}
		defer callbackVm.Close() // Free the memory associated with the callback VM. TODO: Maybe switch to using a Deref API instead of Close?

		cbLua := &CallbackLua{
			mainstate: l,          // The main Lua VM that owns this callback
			cbstate:   callbackVm, // The callback Lua VM that is used to execute the callback
		}

		vmState, err := callback(cbLua)

		if err != nil {
			errv := moveStringToRust(err.Error())
			cval.error = errv // Rust side will deallocate it for us
			return
		}

		cval.vm_state = C.uint8_t(vmState)
	}, nil)

	C.luago_set_interrupt(lua, cbWrapper.ToC())
}

// Removes the interrupt function set by SetInterrupt.
func (l *Lua) RemoveInterrupt() {
	l.object.RLock()
	defer l.object.RUnlock()

	lua, err := l.lua()
	if err != nil {
		return // No-op if the Lua VM is closed
	}

	C.luago_remove_interrupt(lua)
}

// Returns the main thread of the Lua VM.
//
// Note: if you want the currently running thread from a callback, use CallbackLua.CurrentThread() instead.
func (l *Lua) MainThread() *LuaThread {
	l.object.RLock()
	defer l.object.RUnlock()

	lua, err := l.lua()
	if err != nil {
		return nil // Return nil if the Lua VM is closed
	}

	thread := C.luago_current_thread(lua)
	if thread == nil {
		return nil // Return nil if the main thread is not available
	}

	return &LuaThread{object: newObject((*C.void)(unsafe.Pointer(thread)), threadTab), lua: l}
}

// CreateString creates a Lua string from a Go string.
func (l *Lua) CreateString(s string) (*LuaString, error) {
	return l.createString([]byte(s))
}

// CreateStringBytes creates a Lua string from a byte slice.
// This is useful for creating strings from raw byte data.
func (l *Lua) CreateStringBytes(s []byte) (*LuaString, error) {
	return l.createString(s)
}

func (l *Lua) createString(s []byte) (*LuaString, error) {
	l.object.RLock()
	defer l.object.RUnlock()

	lua, err := l.lua()
	if err != nil {
		return nil, err
	}

	if len(s) == 0 {
		// Passing nil to luago_create_string creates an empty string.
		res := C.luago_create_string(lua, (*C.char)(nil), C.size_t(len(s)))
		if res.error != nil {
			return nil, moveErrorToGo(res.error)
		}
		return &LuaString{object: newObject((*C.void)(unsafe.Pointer(res.value)), stringTab), lua: l}, nil
	}

	res := C.luago_create_string(lua, (*C.char)(unsafe.Pointer(&s[0])), C.size_t(len(s)))
	if res.error != nil {
		return nil, moveErrorToGo(res.error)
	}
	return &LuaString{object: newObject((*C.void)(unsafe.Pointer(res.value)), stringTab), lua: l}, nil
}

// Create string as pointer (without any finalizer)
func (l *Lua) createStringAsPtr(s []byte) (*C.struct_LuaString, error) {
	l.object.RLock()
	defer l.object.RUnlock()

	lua, err := l.lua()
	if err != nil {
		return nil, err
	}

	if len(s) == 0 {
		// Passing nil to luago_create_string creates an empty string.
		res := C.luago_create_string(lua, (*C.char)(nil), C.size_t(len(s)))
		if res.error != nil {
			return nil, moveErrorToGo(res.error)
		}
		return res.value, nil
	}

	res := C.luago_create_string(lua, (*C.char)(unsafe.Pointer(&s[0])), C.size_t(len(s)))
	if res.error != nil {
		return nil, moveErrorToGo(res.error)
	}
	return res.value, nil
}

// CreateTable creates a new Lua table.
func (l *Lua) CreateTable() (*LuaTable, error) {
	l.object.RLock()
	defer l.object.RUnlock()

	lua, err := l.lua()
	if err != nil {
		return nil, err
	}

	res := C.luago_create_table(lua)
	if res.error != nil {
		err := moveErrorToGo(res.error)
		return nil, err
	}
	return &LuaTable{object: newObject((*C.void)(unsafe.Pointer(res.value)), tableTab), lua: l}, nil
}

// CreateTableWithCapacity creates a new Lua table with specified capacity for array and record parts.
// with narr as the number of array elements and nrec as the number of record elements.
func (l *Lua) CreateTableWithCapacity(narr, nrec int) (*LuaTable, error) {
	l.object.RLock()
	defer l.object.RUnlock()

	lua, err := l.lua()
	if err != nil {
		return nil, err
	}

	res := C.luago_create_table_with_capacity(lua, C.size_t(narr), C.size_t(nrec))
	if res.error != nil {
		err := moveErrorToGo(res.error)
		return nil, err
	}
	return &LuaTable{object: newObject((*C.void)(unsafe.Pointer(res.value)), tableTab), lua: l}, nil
}

type FunctionFn func(funcVm *CallbackLua, args []Value) ([]Value, error)

// CreateFunction creates a new Function
//
// # Note that funcVm will only be open until the callback function returns
//
// Locking behavior: All values returned by the callback function
// will be write-locked (taken ownership of). Having any sort of read-lock
// to a returned argument during a return will cause a error to be returned to Luau
func (l *Lua) CreateFunction(callback FunctionFn) (*LuaFunction, error) {
	l.object.RLock()
	defer l.object.RUnlock()

	lua, err := l.lua()
	if err != nil {
		return nil, err
	}

	cbWrapper := newGoCallback(func(val unsafe.Pointer) {
		cval := (*C.struct_FunctionCallbackData)(val)

		// Safety: it is undefined behavior for the callback to unwind into
		// Rust (or even C!) frames from Go, so we must recover() any panic
		// that occurs in the callback to prevent a crash.
		defer func() {
			if r := recover(); r != nil {
				// Deallocate any existing error
				if cval.error != nil {
					freeRustString(cval.error)
				}

				// Replace
				errv := moveStringToRust(fmt.Sprintf("panic in CreateFunction callback: %v", r))
				cval.error = errv // Rust side will deallocate it for us
			}
		}()

		// Take out args
		// mw as a object will be deallocated by the Rust side
		mw := &luaMultiValue{ptr: cval.args, lua: l}
		args := mw.take()

		callbackVm := &Lua{object: newObject((*C.void)(unsafe.Pointer(cval.lua)), luaVmTab)}
		defer callbackVm.Close() // Free the memory associated with the callback VM. TODO: Maybe switch to using a Deref API instead of Close?

		cbLua := &CallbackLua{
			mainstate: l,          // The main Lua VM that owns this callback
			cbstate:   callbackVm, // The callback Lua VM that is used to execute the callback
		}

		values, err := callback(cbLua, args)

		if err != nil {
			errv := moveStringToRust(err.Error())
			cval.error = errv // Rust side will deallocate it for us
			return
		}

		outMw, err := l.multiValueFromValues(values)
		if err != nil {
			errv := moveStringToRust(err.Error())
			cval.error = errv // Rust side will deallocate it for us
			return
		}

		cval.values = outMw.ptr // Rust will deallocate values as well
	}, nil)

	res := C.luago_create_function(lua, cbWrapper.ToC())
	if res.error != nil {
		err := moveErrorToGo(res.error)
		return nil, err
	}

	return &LuaFunction{object: newObject((*C.void)(unsafe.Pointer(res.value)), functionTab), lua: l}, nil
}

// CreateThread creates a new thread from a LuaFunction
//
// Locking behavior: Takes a read-lock on the LuaFunction object
// and the Lua VM object
func (l *Lua) CreateThread(fn *LuaFunction) (*LuaThread, error) {
	if fn == nil {
		return nil, fmt.Errorf("function cannot be nil")
	}

	if fn.lua != l {
		return nil, fmt.Errorf("cannot create thread from different Lua instance")
	}

	l.object.RLock()
	defer l.object.RUnlock()

	lua, err := l.lua()
	if err != nil {
		return nil, err
	}

	fn.object.RLock()
	defer fn.object.RUnlock()
	fnPtr, err := fn.innerPtr()
	if err != nil {
		return nil, err // Return error if the function is closed
	}

	res := C.luago_create_thread(lua, fnPtr)
	if res.error != nil {
		err := moveErrorToGo(res.error)
		return nil, err
	}
	return &LuaThread{object: newObject((*C.void)(unsafe.Pointer(res.value)), threadTab), lua: l}, nil
}

// CreateBuffer creates a LuaBuffer from a byte slice.
func (l *Lua) CreateBuffer(s []byte) (*LuaBuffer, error) {
	l.object.RLock()
	defer l.object.RUnlock()

	lua, err := l.lua()
	if err != nil {
		return nil, err
	}

	if len(s) == 0 {
		// Passing nil to luago_create_buffer creates an empty buffer.
		res := C.luago_create_buffer(lua, (*C.char)(nil), C.size_t(len(s)))
		if res.error != nil {
			return nil, moveErrorToGo(res.error)
		}
		return &LuaBuffer{object: newObject((*C.void)(unsafe.Pointer(res.value)), bufferTab), lua: l}, nil
	}

	res := C.luago_create_buffer(lua, (*C.char)(unsafe.Pointer(&s[0])), C.size_t(len(s)))
	if res.error != nil {
		return nil, moveErrorToGo(res.error)
	}
	return &LuaBuffer{object: newObject((*C.void)(unsafe.Pointer(res.value)), bufferTab), lua: l}, nil
}

// LoadChunk loads a Lua chunk from the given options.
func (l *Lua) LoadChunk(opts ChunkOpts) (*LuaFunction, error) {
	l.object.RLock()
	defer l.object.RUnlock()

	lua, err := l.lua()
	if err != nil {
		return nil, err
	}

	var env *C.struct_LuaTable
	if opts.Env != nil {
		if opts.Env.lua != l {
			return nil, fmt.Errorf("cannot set environment table from different Lua instance")
		}

		defer opts.Env.object.RUnlock()
		opts.Env.object.RLock()
		envPtr, err := opts.Env.object.PointerNoLock()
		if err == nil {
			env = (*C.struct_LuaTable)(unsafe.Pointer(envPtr))
		}
	}

	var compilerOpts *C.struct_CompilerOpts = nil
	if opts.CompilerOpts != nil {
		compilerOptsC := opts.CompilerOpts.toC()
		compilerOpts = &compilerOptsC
	}

	var name = newChunkString([]byte(opts.Name))
	var code = newChunkString([]byte(opts.Code))

	res := C.luago_load_chunk(
		lua,
		C.struct_ChunkOpts{
			name:          name,
			env:           env,
			mode:          C.uint8_t(opts.Mode),
			compiler_opts: compilerOpts,
			code:          code,
		},
	)

	if res.error != nil {
		err := moveErrorToGo(res.error)
		return nil, err
	}
	return &LuaFunction{object: newObject((*C.void)(unsafe.Pointer(res.value)), functionTab), lua: l}, nil
}

// CreateUserData creates a LuaUserData with associated data and a metatable.
func (l *Lua) CreateUserData(associatedData any, mt *LuaTable) (*LuaUserData, error) {
	if mt == nil {
		return nil, fmt.Errorf("metatable cannot be nil")
	}
	if mt.lua != l {
		return nil, fmt.Errorf("cannot create userdata with metatable from different Lua instance")
	}

	l.object.RLock()
	defer l.object.RUnlock()

	lua, err := l.lua()
	if err != nil {
		return nil, err
	}

	defer mt.object.RUnlock()
	mt.object.RLock()
	mtPtr, err := mt.innerPtr()
	if err != nil {
		return nil, err // Return error if the metatable is closed
	}

	dynData := newDynamicData(associatedData, nil)
	cDynData := dynData.ToC()
	res := C.luago_create_userdata(lua, cDynData, mtPtr)
	if res.error != nil {
		err := moveErrorToGo(res.error)
		return nil, err
	}
	return &LuaUserData{
		lua:    l,
		object: newObject((*C.void)(unsafe.Pointer(res.value)), userdataTab),
	}, nil
}

func (l *Lua) Close() error {
	if l == nil || l.object == nil {
		return nil // Nothing to close
	}

	// Close the Lua VM object
	return l.object.Close()
}

type StdLib uint32

const (
	StdLibCoroutine StdLib = 1 << 0
	StdLibTable     StdLib = 1 << 1
	StdLibOS        StdLib = 1 << 2
	StdLibString    StdLib = 1 << 3
	StdLibUtf8      StdLib = 1 << 4
	StdLibBit       StdLib = 1 << 5
	StdLibMath      StdLib = 1 << 6
	StdLibBuffer    StdLib = 1 << 7
	StdLibVector    StdLib = 1 << 8
	StdLibDebug     StdLib = 1 << 9
	StdLibAll       StdLib = 1 << 31 // All standard libraries
)

// CreateLuaVm creates a new Lua VM with the entire standard library enabled.
func CreateLuaVm() (*Lua, error) {
	return CreateLuaVmComplex(StdLibAll)
}

// CreateLuaVmComplex creates a new Lua VM with the specified standard libraries enabled.
//
// If you want the entire stdlib to be exposed to scripts, pass `StdLibAll` here
// or use the CreateLuaVm function.
func CreateLuaVmComplex(stdlib StdLib) (*Lua, error) {
	ptr := C.newluavm(C.uint32_t(stdlib))
	if ptr == nil {
		return nil, fmt.Errorf("failed to create Lua VM")
	}
	vm := &Lua{object: newObject((*C.void)(unsafe.Pointer(ptr)), luaVmTab)}
	return vm, nil
}

// A special 'borrowed' Lua VM that is passed to callbacks.
//
// Provides special context-specific data about the current Lua state
type CallbackLua struct {
	mainstate *Lua // The main Lua VM that owns this callback
	cbstate   *Lua // The callback Lua
}

// Returns the main Lua VM state
//
// Note: it is not possible to get the callback Lua state directly to avoid
// object lifetime related issues.
//
// Returns nil if the CallbackLua is closed (note that CallbackLua is closed automatically when the callback function returns).
func (c *CallbackLua) MainState() *Lua {
	if c == nil {
		return nil // No main state if the callback Lua is nil
	}
	return c.mainstate
}

// Returns the currently running thread of the Lua VM.
//
// Returns nil if the CallbackLua is closed (note that CallbackLua is closed automatically when the callback function returns).
func (c *CallbackLua) CurrentThread() *LuaThread {
	if c.mainstate == nil || c.cbstate == nil {
		return nil // No current thread if the main state or callback state is nil
	}

	c.cbstate.object.RLock()
	defer c.cbstate.object.RUnlock()
	c.mainstate.object.RLock()
	defer c.mainstate.object.RUnlock()

	lua, err := c.cbstate.lua()
	if err != nil {
		return nil // Return nil if the Lua VM is closed
	}

	thread := C.luago_current_thread(lua)
	if thread == nil {
		return nil // Return nil if the callback thread is not available
	}

	return &LuaThread{object: newObject((*C.void)(unsafe.Pointer(thread)), threadTab), lua: c.mainstate}
}

// Sets the arguments to yield the thread with.
//
// Notes:
// - the yield will only occur after return.
// - the arguments returned will be ignored internally (as such, you should just return a empty value list after calling this method).
func (c *CallbackLua) YieldWith(args []Value) error {
	if c == nil || c.cbstate == nil {
		return fmt.Errorf("callback Lua VM is closed")
	}

	c.cbstate.object.RLock()
	defer c.cbstate.object.RUnlock()

	lua, err := c.cbstate.lua()
	if err != nil {
		return err // Return error if the callback Lua VM is closed
	}

	mw, err := c.cbstate.multiValueFromValues(args)
	if err != nil {
		return err // Return error if the values cannot be converted (diff lua state, closed object, etc)
	}

	res := C.luago_yield_with(lua, mw.ptr)
	if res.error != nil {
		return moveErrorToGo(res.error) // Return error if the yield failed
	}
	return nil
}

// Closes the CallbackLua object.
//
// Note: this is automatically called when the callback function returns,
func (c *CallbackLua) Close() error {
	if c == nil || c.cbstate == nil {
		return nil // Nothing to close
	}
	// Close the callback Lua VM object
	err := c.cbstate.Close()
	if err != nil {
		return err // Return error if the callback Lua VM is closed
	}
	// Nil out the mainstate to allow GC
	c.mainstate = nil
	return nil
}
