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

// The result of a navigation operation in a Luau require-by-string operation
type NavigationResult struct {
	ambiguous bool
	notfound  bool
	other     error
}

func (n *NavigationResult) fillC(c *C.struct_GoNavigationResult) {
	if n == nil {
		return // Nothing to fill
	}

	if n.ambiguous {
		c.ambiguous = C.bool(true)
	} else if n.notfound {
		c.not_found = C.bool(true)
	} else if n.other != nil {
		c.other = moveStringToRust(n.other.Error())
	}
}

// Returns a new ambiguous navigation result
func AmbiguousNavigationResult() *NavigationResult {
	return &NavigationResult{ambiguous: true}
}

// Returns a new not found navigation result
func NotFoundNavigationResult() *NavigationResult {
	return &NavigationResult{notfound: true}
}

// Returns a new other navigation result with the given error
func OtherNavigationResult(err error) *NavigationResult {
	return &NavigationResult{other: err}
}

// Returns a new other navigation result with the given error as a string
func OtherStringNavigationResult(err string) *NavigationResult {
	return &NavigationResult{other: errors.New(err)}
}

// Require is the interface Luau Require will use
// to resolve Luau module paths (Luau require-by-string).
type Require interface {
	// Returns true if “require” is permitted for the given chunk name.
	IsRequireAllowed(chunkName string) bool

	// Resets the internal state to point at the requirer module.
	Reset(chunkName string) *NavigationResult

	// Resets the internal state to point at an aliased module.
	//
	// This function received an exact path from a configuration file.
	//
	// It’s only called when an alias’s path cannot be resolved relative to its configuration file.
	JumpToAlias(path string) *NavigationResult

	// Asks the Require to go to the parent of the current module.
	ToParent() *NavigationResult

	// Asks the Require to go to the child of the current module.
	ToChild(name string) *NavigationResult

	// Returns whether the context is currently pointing at a module.
	HasModule() bool

	// Provides a cache key representing the current module.
	//
	// This function is only called if HasModule returns true.
	CacheKey() string

	// Returns whether a configuration (.luaurc) is present in the current context
	HasConfig() bool

	// Returns the contents of the configuration file (.luaurc) in the current context.
	//
	// This function is only called if HasConfig returns true.
	Config() ([]byte, error)

	// Returns a loader function for the current module, that when called, loads the module and returns the result.
	//
	// This function is only called if has_module returns true.
	Loader(cb *CallbackLua) (*LuaFunction, error)
}

// CreateRequireFunction creates a LuaFunction via Luau's Require By String API using the
// given requirer
func (l *Lua) CreateRequireFunction(require Require) (*LuaFunction, error) {
	l.object.RLock()
	defer l.object.RUnlock()

	lua, err := l.lua()
	if err != nil {
		return nil, err
	}

	// Common Safety Notes: it is undefined behavior for the callback to unwind into
	// Rust (or even C!) frames from Go, so we must recover() any panic
	// that occurs in the callback to prevent a crash.

	isRequireAllowed := newGoCallback(func(val unsafe.Pointer) {
		cval := (*C.struct_IsRequireAllowed)(val)
		defer func() {
			if r := recover(); r != nil {
				cval.data = false // Requires are not allowed if we panic
			}
		}()
		chunkname := moveStringToGo(cval.chunk_name)
		cval.data = C.bool(require.IsRequireAllowed(chunkname))
	}, nil)

	reset := newGoCallback(func(val unsafe.Pointer) {
		cval := (*C.struct_ResetOrJumpToAliasOrToChild)(val)
		defer func() {
			if r := recover(); r != nil {
				ne := OtherNavigationResult(fmt.Errorf("panic in Reset: %v", r))
				ne.fillC(&cval.data)
			}
		}()
		chunkname := moveStringToGo(cval.str)
		require.Reset(chunkname).fillC(&cval.data)
	}, nil)

	jumpToAlias := newGoCallback(func(val unsafe.Pointer) {
		cval := (*C.struct_ResetOrJumpToAliasOrToChild)(val)
		defer func() {
			if r := recover(); r != nil {
				ne := OtherNavigationResult(fmt.Errorf("panic in Reset: %v", r))
				ne.fillC(&cval.data)
			}
		}()
		path := moveStringToGo(cval.str)
		require.JumpToAlias(path).fillC(&cval.data)
	}, nil)

	toParent := newGoCallback(func(val unsafe.Pointer) {
		cval := (*C.struct_ToParent)(val)
		defer func() {
			if r := recover(); r != nil {
				ne := OtherNavigationResult(fmt.Errorf("panic in Reset: %v", r))
				ne.fillC(&cval.data)
			}
		}()
		require.ToParent().fillC(&cval.data)
	}, nil)

	toChild := newGoCallback(func(val unsafe.Pointer) {
		cval := (*C.struct_ResetOrJumpToAliasOrToChild)(val)
		defer func() {
			if r := recover(); r != nil {
				ne := OtherNavigationResult(fmt.Errorf("panic in Reset: %v", r))
				ne.fillC(&cval.data)
			}
		}()
		name := moveStringToGo(cval.str)
		require.ToChild(name).fillC(&cval.data)
	}, nil)

	hasModule := newGoCallback(func(val unsafe.Pointer) {
		cval := (*C.struct_HasModuleOrHasConfig)(val)
		defer func() {
			if r := recover(); r != nil {
				cval.data = C.bool(false) // If we panic, we assume no module
			}
		}()
		cval.data = C.bool(require.HasModule())
	}, nil)

	cacheKey := newGoCallback(func(val unsafe.Pointer) {
		cval := (*C.struct_CacheKey)(val)
		defer func() {
			if r := recover(); r != nil {
				cval.data = moveStringToRust("")
			}
		}()
		cval.data = moveStringToRust(require.CacheKey())
	}, nil)

	hasConfig := newGoCallback(func(val unsafe.Pointer) {
		cval := (*C.struct_HasModuleOrHasConfig)(val)
		defer func() {
			if r := recover(); r != nil {
				cval.data = C.bool(false) // If we panic, we assume no module
			}
		}()
		cval.data = C.bool(require.HasConfig())
	}, nil)

	config := newGoCallback(func(val unsafe.Pointer) {
		cval := (*C.struct_Config)(val)
		defer func() {
			if r := recover(); r != nil {
				cval.error = moveStringToRust(fmt.Sprintf("panic in config callback: %v", r))
			}
		}()
		bytes, err := require.Config()
		if err != nil {
			cval.error = moveStringToRust(err.Error())
			return
		}
		cval.data = moveBytesToRust(bytes)
	}, nil)

	loader := newGoCallback(func(val unsafe.Pointer) {
		cval := (*C.struct_Loader)(val)
		defer func() {
			if r := recover(); r != nil {
				cval.error = moveStringToRust(fmt.Sprintf("panic in config callback: %v", r))
			}
		}()

		callbackVm := &Lua{object: newObject((*C.void)(unsafe.Pointer(cval.lua)), luaVmTab)}
		defer callbackVm.Close() // Free the memory associated with the callback VM. TODO: Maybe switch to using a Deref API instead of Close?

		cbLua := &CallbackLua{
			mainstate: l,          // The main Lua VM that owns this callback
			cbstate:   callbackVm, // The callback Lua VM that is used to execute the callback
		}

		fn, err := require.Loader(cbLua)
		if err != nil {
			cval.error = moveStringToRust(err.Error())
			return
		}
		if fn == nil {
			cval.error = moveStringToRust("loader returned nil function")
			return
		}
		err = fn.object.Disarm()
		if err != nil {
			cval.error = moveStringToRust(err.Error())
			return
		}
		ptr, err := fn.object.UnsafePointer()
		if err != nil {
			cval.error = moveStringToRust(err.Error())
			return
		}

		cval.function = (*C.struct_LuaFunction)(unsafe.Pointer(ptr))
	}, nil)

	res := C.luago_create_require_function(lua, C.struct_GoRequire{
		is_require_allowed: isRequireAllowed.ToC(),
		reset:              reset.ToC(),
		jump_to_alias:      jumpToAlias.ToC(),
		to_parent:          toParent.ToC(),
		to_child:           toChild.ToC(),
		has_module:         hasModule.ToC(),
		cache_key:          cacheKey.ToC(),
		has_config:         hasConfig.ToC(),
		config:             config.ToC(),
		loader:             loader.ToC(),
	})

	if res.error != nil {
		err := moveErrorToGo(res.error)
		return nil, err
	}

	return &LuaFunction{object: newObject((*C.void)(unsafe.Pointer(res.value)), functionTab), lua: l}, nil
}
