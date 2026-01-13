package vmutils

import (
	"errors"
	"fmt"

	"github.com/gluau/gluau/vm"
)

// Ergonomic userdata handling
//
// WARNING: This is an experimental and completely untested feature.
type TypedUserData[T any] struct {
	mt           *vm.LuaTable                                                         // cached/shared metatable for the user data
	fields       map[string]vm.Value                                                  // fields of the user data
	fieldGetters map[string]func(*T) (vm.Value, error)                                // field getters
	fieldSetters map[string]func(*T, *vm.CallbackLua, vm.Value) error                 // field setters
	methods      map[string]func(*T, *vm.CallbackLua, []vm.Value) ([]vm.Value, error) // methods of the user data
	typename     string                                                               // type name of the user data
	metamethods  map[string]func(*T, *vm.CallbackLua, []vm.Value) ([]vm.Value, error) // metamethods
}

// Parse the first value as a TypedUserData of type T returning the data and the remaining values
func ParseSelf[T any](typeName string, values []vm.Value) (*T, []vm.Value, error) {
	if len(values) == 0 {
		return nil, nil, errors.New("expected argument #1 to be a userdata, but got nil")
	}

	self, ok := values[0].(*vm.ValueUserData)
	if !ok {
		return nil, nil, TypeMismatchError(0, "userdata", values[0].Type().String())
	}

	data, err := self.Value().AssociatedData()
	if err != nil {
		return nil, nil, err
	}
	dataT, ok := data.(*T)
	if !ok {
		return nil, nil, TypeMismatchError(0, "userdata of type "+typeName, "userdata")
	}

	return dataT, values[1:], nil
}

// Adds a field to the TypedUserData
func (tud *TypedUserData[T]) AddField(name string, value vm.Value) {
	tud.fields[name] = value
}

// Adds a field getter to the TypedUserData
func (tud *TypedUserData[T]) AddFieldGetter(name string, getter func(*T) (vm.Value, error)) {
	tud.fieldGetters[name] = getter
}

// Adds a field setter to the TypedUserData
func (tud *TypedUserData[T]) AddFieldSetter(name string, setter func(*T, *vm.CallbackLua, vm.Value) error) {
	tud.fieldSetters[name] = setter
}

// Adds a method to the TypedUserData
func (tud *TypedUserData[T]) AddMethod(name string, method func(*T, *vm.CallbackLua, []vm.Value) ([]vm.Value, error)) {
	tud.methods[name] = method
}

// Adds a metafields to the TypedUserData
func (tud *TypedUserData[T]) SetTypeName(typename string) {
	tud.typename = typename
}

// Adds a metamethod to the TypedUserData
func (tud *TypedUserData[T]) AddMetamethod(name string, method func(*T, *vm.CallbackLua, []vm.Value) ([]vm.Value, error)) {
	tud.metamethods[name] = method
}

// Returns `true` if the __index metamethod should be a table
//
// A fastpath is only allowed if there are no field getters,
func (tud *TypedUserData[T]) indexFastPath() bool {
	// if there are no field getters or setters, we can use a table for __index
	if len(tud.fieldGetters) != 0 {
		return false
	}

	// Check __index metamethod, if its a table, we can still use fastpath, otherwise we need to use a function
	if _, ok := tud.metamethods["__index"]; ok {
		return false
	}

	return true
}

// Creates a new UserData
func (tud *TypedUserData[T]) Create(lua *vm.Lua, data *T) (*vm.LuaUserData, error) {
	if tud.mt == nil {
		if tud.indexFastPath() {
			mt, err := tud.createMtFast(lua)
			if err != nil {
				return nil, err
			}
			tud.mt = mt
		} else {
			mt, err := tud.createMtSlow(lua)
			if err != nil {
				return nil, err
			}
			tud.mt = mt
		}
	}

	ud, err := lua.CreateUserData(data, tud.mt)
	if err != nil {
		return nil, err
	}
	return ud, nil
}

func (tud *TypedUserData[T]) createMtFast(lua *vm.Lua) (*vm.LuaTable, error) {
	typeName := tud.typename

	udMt, err := lua.CreateTable()
	if err != nil {
		return nil, err
	}
	err = udMt.Set(vm.GoString("__type"), vm.GoString(typeName))
	if err != nil {
		return nil, err
	}
	err = udMt.Set(vm.GoString("__metatable"), vm.NewValueBoolean(false))
	if err != nil {
		return nil, err
	}

	for key, value := range tud.metamethods {
		callback := func(funcVm *vm.CallbackLua, args []vm.Value) ([]vm.Value, error) {
			self, args, err := ParseSelf[T](typeName, args)
			if err != nil {
				return nil, err
			}
			return value(self, funcVm, args)
		}
		funct, err := lua.CreateFunction(callback)
		if err != nil {
			return nil, err
		}
		err = udMt.Set(vm.GoString(key), funct.ToValue())
		if err != nil {
			return nil, err
		}
	}

	// fastpath doesnt support field getters or setters, so we can just use a table
	indexMt, err := lua.CreateTable()
	if err != nil {
		return nil, err
	}

	for key, value := range tud.fields {
		if err := indexMt.Set(vm.GoString(key), value); err != nil {
			return nil, err
		}
	}

	for key, method := range tud.methods {
		callback := func(funcVm *vm.CallbackLua, args []vm.Value) ([]vm.Value, error) {
			self, args, err := ParseSelf[T](typeName, args)
			if err != nil {
				return nil, err
			}
			return method(self, funcVm, args)
		}
		funct, err := lua.CreateFunction(callback)
		if err != nil {
			return nil, err
		}

		if err := indexMt.Set(vm.GoString(key), funct.ToValue()); err != nil {
			return nil, err
		}
	}

	if err := udMt.Set(vm.GoString("__index"), indexMt.ToValue()); err != nil {
		return nil, err
	}

	// Handle field setters (which are handled via __newindex)
	if len(tud.fieldSetters) != 0 {
		newIndexCallback := func(funcVm *vm.CallbackLua, args []vm.Value) ([]vm.Value, error) {
			self, args, err := ParseSelf[T](typeName, args)
			if err != nil {
				return nil, err
			}
			if len(args) != 2 {
				return nil, errors.New("expected 2 arguments for __newindex, got " + fmt.Sprint(len(args)))
			}

			fieldName, ok := args[0].(vm.GoString)
			if !ok {
				return nil, TypeMismatchError(0, "string", args[0].Type().String())
			}

			value := args[1]
			setter, ok := tud.fieldSetters[string(fieldName)]
			if !ok {
				return nil, errors.New("no setter defined for field " + string(fieldName))
			}

			if err := setter(self, funcVm, value); err != nil {
				return nil, err
			}

			return nil, nil
		}

		newIndexFunc, err := lua.CreateFunction(newIndexCallback)
		if err != nil {
			return nil, err
		}

		if err := udMt.Set(vm.GoString("__newindex"), newIndexFunc.ToValue()); err != nil {
			return nil, err
		}
	}

	return udMt, nil
}

func (tud *TypedUserData[T]) createMtSlow(lua *vm.Lua) (*vm.LuaTable, error) {
	typeName := tud.typename

	udMt, err := lua.CreateTable()
	if err != nil {
		return nil, err
	}
	err = udMt.Set(vm.GoString("__type"), vm.GoString(typeName))
	if err != nil {
		return nil, err
	}
	err = udMt.Set(vm.GoString("__metatable"), vm.NewValueBoolean(false))
	if err != nil {
		return nil, err
	}

	for key, value := range tud.metamethods {
		if key == "__index" {
			continue
		}

		callback := func(funcVm *vm.CallbackLua, args []vm.Value) ([]vm.Value, error) {
			self, args, err := ParseSelf[T](typeName, args)
			if err != nil {
				return nil, err
			}
			return value(self, funcVm, args)
		}
		funct, err := lua.CreateFunction(callback)
		if err != nil {
			return nil, err
		}
		err = udMt.Set(vm.GoString(key), funct.ToValue())
		if err != nil {
			return nil, err
		}
	}

	// Create the field getter functions once
	fieldGetterFuncs := make(map[string]*vm.LuaFunction)
	for key, getter := range tud.fieldGetters {
		callback := func(funcVm *vm.CallbackLua, args []vm.Value) ([]vm.Value, error) {
			self, _, err := ParseSelf[T](typeName, args)
			if err != nil {
				return nil, err
			}

			value, err := getter(self)
			if err != nil {
				return nil, err
			}

			return []vm.Value{value}, nil
		}
		funct, err := lua.CreateFunction(callback)
		if err != nil {
			return nil, err
		}
		fieldGetterFuncs[key] = funct
	}

	var methodFuncs = make(map[string]*vm.LuaFunction)
	for key, method := range tud.methods {
		callback := func(funcVm *vm.CallbackLua, args []vm.Value) ([]vm.Value, error) {
			self, args, err := ParseSelf[T](typeName, args)
			if err != nil {
				return nil, err
			}
			return method(self, funcVm, args)
		}
		funct, err := lua.CreateFunction(callback)
		if err != nil {
			return nil, err
		}

		methodFuncs[key] = funct
	}

	indexCallback := func(funcVm *vm.CallbackLua, args []vm.Value) ([]vm.Value, error) {
		_, args, err := ParseSelf[T](typeName, args)
		if err != nil {
			return nil, err
		}

		// Check field getters first
		if len(args) < 1 {
			return nil, errors.New("expected at least 1 argument for __index, got " + fmt.Sprint(len(args)))
		}
		fieldName, ok := args[0].(*vm.ValueString)
		if !ok {
			return nil, TypeMismatchError(0, "string", args[0].Type().String())
		}
		if fieldGetter, ok := fieldGetterFuncs[fieldName.Value().String()]; ok {
			return []vm.Value{fieldGetter.ToValue()}, nil
		}

		// Check methods
		if methods, ok := methodFuncs[fieldName.Value().String()]; ok {
			return []vm.Value{methods.ToValue()}, nil
		}

		return nil, errors.New("no field or method found for " + fieldName.Value().String())
	}

	// Set index metamethod
	indexFunc, err := lua.CreateFunction(indexCallback)
	if err != nil {
		return nil, err
	}
	if err := udMt.Set(vm.GoString("__index"), indexFunc.ToValue()); err != nil {
		return nil, err
	}

	// Handle field setters (which are handled via __newindex)
	if len(tud.fieldSetters) != 0 {
		newIndexCallback := func(funcVm *vm.CallbackLua, args []vm.Value) ([]vm.Value, error) {
			self, args, err := ParseSelf[T](typeName, args)
			if err != nil {
				return nil, err
			}
			if len(args) != 2 {
				return nil, errors.New("expected 2 arguments for __newindex, got " + fmt.Sprint(len(args)))
			}

			fieldName, ok := args[0].(vm.GoString)
			if !ok {
				return nil, TypeMismatchError(0, "string", args[0].Type().String())
			}

			value := args[1]
			setter, ok := tud.fieldSetters[string(fieldName)]
			if !ok {
				return nil, errors.New("no setter defined for field " + string(fieldName))
			}

			if err := setter(self, funcVm, value); err != nil {
				return nil, err
			}

			return nil, nil
		}

		newIndexFunc, err := lua.CreateFunction(newIndexCallback)
		if err != nil {
			return nil, err
		}

		if err := udMt.Set(vm.GoString("__newindex"), newIndexFunc.ToValue()); err != nil {
			return nil, err
		}
	}

	return udMt, nil
}

// Creates a new TypedUserData which can be used to ergonomically build user data
func NewTypedUserData[T any]() *TypedUserData[T] {
	return &TypedUserData[T]{
		fields:       make(map[string]vm.Value),
		fieldGetters: make(map[string]func(*T) (vm.Value, error)),
		fieldSetters: make(map[string]func(*T, *vm.CallbackLua, vm.Value) error),
		methods:      make(map[string]func(*T, *vm.CallbackLua, []vm.Value) ([]vm.Value, error)),
		typename:     "",
		metamethods:  make(map[string]func(*T, *vm.CallbackLua, []vm.Value) ([]vm.Value, error)),
	}
}
