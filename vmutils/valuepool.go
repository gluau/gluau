package vmutils

import (
	"errors"
	"strings"

	"github.com/koeng101/gluau/vm"
)

// A ValuePool stores a Value and allows for cloning it
// with a Close method to free both the original and cloned values.
type ValuePool struct {
	lua    *vm.Lua
	value  vm.Value
	clones []vm.Value
}

func NewValuePool(lua *vm.Lua, value vm.Value) *ValuePool {
	return &ValuePool{
		lua:    lua,
		value:  value,
		clones: make([]vm.Value, 0),
	}
}

// Value creates a clone of the stored value
func (vp *ValuePool) Value() vm.Value {
	clonedRef := vp.value.Clone()
	vp.clones = append(vp.clones, clonedRef)
	return clonedRef
}

// Close frees the original value and all cloned values
func (vp *ValuePool) Close() error {
	errs := make([]string, 0)
	for _, clone := range vp.clones {
		err := clone.Close()
		if err != nil {
			errs = append(errs, err.Error())
		}
	}
	vp.clones = nil

	if vp.value != nil {
		err := vp.value.Close()
		if err != nil {
			errs = append(errs, err.Error())
		}
		vp.value = nil
	}

	if len(errs) > 0 {
		return errors.New("multiple errors occurred while closing ValuePool: " + strings.Join(errs, "; "))
	}
	return nil
}
