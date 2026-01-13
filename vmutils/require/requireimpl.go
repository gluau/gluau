package require

import (
	"fmt"
	"io"

	"github.com/koeng101/gluau/vm"
)

type SimpleRequirer struct {
	cachePrefix string
	vfs         *vfsNavigator
	globalTable *vm.LuaTable
	debug       bool
}

func NewSimpleRequirer(cachePrefix string, globalTable *vm.LuaTable, vfs Vfs, debug bool) *SimpleRequirer {
	return &SimpleRequirer{
		cachePrefix: cachePrefix,
		vfs:         newVfsNavigator(vfs),
		globalTable: globalTable,
		debug:       debug,
	}
}

func (r *SimpleRequirer) debugPrint(lines ...any) {
	if r.debug {
		fmt.Println(lines...)
	}
}

func (r *SimpleRequirer) IsRequireAllowed(chunkName string) bool {
	return true
}

func (r *SimpleRequirer) Reset(chunkName string) *vm.NavigationResult {
	r.debugPrint("Resetting require with chunk name:", chunkName)
	if chunkName == "=repl" {
		return r.vfs.resetToStdin()
	}

	return r.vfs.resetToPath(chunkName)
}

func (r *SimpleRequirer) JumpToAlias(path string) *vm.NavigationResult {
	r.debugPrint("Jumping to alias:", path)
	if !r.vfs.fs.IsAbsolutePath(path) {
		return vm.NotFoundNavigationResult()
	}

	return r.vfs.resetToPath(path)
}

func (r *SimpleRequirer) ToParent() *vm.NavigationResult {
	r.debugPrint("Navigating to parent directory")
	return r.vfs.toParent()
}

func (r *SimpleRequirer) ToChild(name string) *vm.NavigationResult {
	r.debugPrint("Navigating to child:", name)
	return r.vfs.toChild(name)
}

func (r *SimpleRequirer) HasModule() bool {
	r.debugPrint("Checking if module exists at:", r.vfs.getFilePath())
	return vfsIsFile(r.vfs.fs, r.vfs.getFilePath())
}

func (r *SimpleRequirer) CacheKey() string {
	r.debugPrint("Generating cache key for:", r.vfs.getAbsoluteFilePath())
	return r.cachePrefix + "@" + r.vfs.getAbsoluteFilePath()
}

func (r *SimpleRequirer) HasConfig() bool {
	r.debugPrint("Checking if config exists at:", r.vfs.getLuaurcPath())
	return vfsIsFile(r.vfs.fs, r.vfs.getLuaurcPath())
}

func (r *SimpleRequirer) Config() ([]byte, error) {
	r.debugPrint("Reading config from:", r.vfs.getLuaurcPath())
	file, err := r.vfs.fs.Open(r.vfs.getLuaurcPath())
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return io.ReadAll(file)
}

func (r *SimpleRequirer) Loader(cb *vm.CallbackLua) (*vm.LuaFunction, error) {
	r.debugPrint("Loading module from:", r.vfs.getFilePath())
	chunkname := r.vfs.getAbsoluteFilePath()
	file, err := r.vfs.fs.Open(chunkname)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	content, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	return cb.MainState().LoadChunk(vm.ChunkOpts{
		Name: chunkname,
		Code: string(content),
		Mode: vm.ChunkModeText,
		Env:  r.globalTable,
	})
}
