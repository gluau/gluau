package require

import (
	"strings"

	"github.com/koeng101/gluau/vm"
)

var (
	SUFFIXES      = [2]string{".luau", ".lua"}
	INIT_SUFFIXES = [2]string{"/init.luau", "/init.lua"}
)

type resolvedRealPath struct {
	status   *vm.NavigationResult
	realPath *string
}

type vfsNavigator struct {
	fs                 Vfs
	realPath           string
	absoluteRealPath   string
	absolutePathPrefix string
	modulePath         string
	absoluteModulePath string
}

func newVfsNavigator(fs Vfs) *vfsNavigator {
	return &vfsNavigator{
		fs:                 fs,
		realPath:           "/",
		absoluteRealPath:   "/",
		absolutePathPrefix: "",
		modulePath:         "/",
		absoluteModulePath: "/",
	}
}

func (v *vfsNavigator) getRealPath(modulePath string) *resolvedRealPath {
	found := false
	suffix := ""

	// Get the position of the last slash
	lastSlash := strings.LastIndex(modulePath, "/")
	if lastSlash == -1 {
		lastSlash = 0
	}
	lastComponent := ""
	if lastSlash != 0 {
		lastComponent = modulePath[lastSlash+1:]
	}

	if lastComponent != "init" {
		for _, potentialSuffix := range SUFFIXES {
			if vfsIsFile(v.fs, modulePath+potentialSuffix) {
				if found {
					return &resolvedRealPath{
						status:   vm.AmbiguousNavigationResult(),
						realPath: nil,
					}
				}

				suffix = potentialSuffix
				found = true
			}
		}
	}

	if vfsIsDir(v.fs, modulePath) {
		if found {
			return &resolvedRealPath{
				status:   vm.AmbiguousNavigationResult(),
				realPath: nil,
			}
		}

		for _, potentialSuffix := range INIT_SUFFIXES {
			if vfsIsFile(v.fs, modulePath+potentialSuffix) {
				if found {
					return &resolvedRealPath{
						status:   vm.AmbiguousNavigationResult(),
						realPath: nil,
					}
				}

				suffix = potentialSuffix
				found = true
			}
		}

		found = true
	}

	if !found {
		return &resolvedRealPath{
			status:   vm.NotFoundNavigationResult(),
			realPath: nil,
		}
	}

	fullPath := modulePath + suffix
	return &resolvedRealPath{
		status:   nil, // nil means success
		realPath: &fullPath,
	}
}

// getModulePath extracts the module path from a file path.
//
// Returns the modified file path and the module path.
func (v *vfsNavigator) getModulePath(filePath string) (string, string) {
	// Normalize separators: replace '\\' with '/'
	filePath = strings.ReplaceAll(filePath, "\\", "/")

	// Create a string view (slice) from the modified path
	pathView := filePath

	// Handle absolute paths
	if v.fs.IsAbsolutePath(pathView) {
		firstSlashIndex := strings.Index(pathView, "/")
		if firstSlashIndex != -1 {
			pathView = pathView[firstSlashIndex:]
		} else {
			panic("Absolute path must contain a slash")
		}
	}

	for _, suffix := range INIT_SUFFIXES {
		if strings.HasSuffix(pathView, suffix) {
			pathView = pathView[:len(pathView)-len(suffix)]

			// BUGFIX: Avoid '.' from being a module_path
			if pathView == "." {
				return filePath, ""
			}

			return filePath, pathView
		}
	}

	for _, suffix := range SUFFIXES {
		if strings.HasSuffix(pathView, suffix) {
			pathView = pathView[:len(pathView)-len(suffix)]

			// BUGFIX: Avoid '.' from being a module_path
			if pathView == "." {
				return filePath, ""
			}

			return filePath, pathView
		}
	}

	// BUGFIX: Avoid '.' from being a module_path
	if pathView == "." {
		return filePath, ""
	}

	return filePath, pathView
}

func (v *vfsNavigator) updateRealPaths() *vm.NavigationResult {
	result := v.getRealPath(v.modulePath)
	absoluteResult := v.getRealPath(v.absoluteModulePath)
	if result.status != nil || absoluteResult.status != nil {
		return result.status
	}

	if result.realPath == nil {
		return vm.OtherStringNavigationResult("result.realPath is nil")
	}
	if absoluteResult.realPath == nil {
		return vm.OtherStringNavigationResult("absoluteResult.realPath is nil")
	}
	resultRealPath := *result.realPath
	absoluteResultRealPath := *absoluteResult.realPath
	if v.fs.IsAbsolutePath(resultRealPath) {
		v.realPath = v.absolutePathPrefix + resultRealPath
	} else {
		v.realPath = resultRealPath
	}
	v.absoluteRealPath = v.absolutePathPrefix + absoluteResultRealPath
	return nil // nil means success
}

func (v *vfsNavigator) resetToStdin() *vm.NavigationResult {
	v.realPath = "./stdin"
	v.absoluteRealPath = "/stdin"
	v.modulePath = "./stdin"
	v.absoluteModulePath = "/stdin"
	v.absolutePathPrefix = ""
	return nil
}

func (v *vfsNavigator) resetToPath(path string) *vm.NavigationResult {
	var normalizedPath = v.fs.NormalizePath(path)

	if v.fs.IsAbsolutePath(normalizedPath) {
		normalizedPath, v.modulePath = v.getModulePath(normalizedPath)
		v.absoluteModulePath = v.modulePath

		firstSlashIndex := strings.Index(normalizedPath, "/")
		if firstSlashIndex == -1 {
			firstSlashIndex = 0
		}
		v.absolutePathPrefix = normalizedPath[:firstSlashIndex]
	} else {
		cwd := v.fs.Cwd()
		normalizedPath, v.modulePath = v.getModulePath(normalizedPath)
		joinedPath := v.fs.NormalizePath(v.fs.Join(cwd, normalizedPath))
		joinedPath, v.absoluteModulePath = v.getModulePath(joinedPath)

		firstSlashIndex := strings.Index(joinedPath, "/")
		if firstSlashIndex == -1 {
			firstSlashIndex = 0
		}
		v.absolutePathPrefix = joinedPath[:firstSlashIndex]
	}

	if v.modulePath == "" {
		v.modulePath = "/" // DEVIATION: Support rooted modules
	}
	if v.absoluteModulePath == "" {
		v.absoluteModulePath = "/" // DEVIATION: Support rooted modules
	}

	return v.updateRealPaths()
}

func (v *vfsNavigator) toParent() *vm.NavigationResult {
	if v.absoluteModulePath == "" {
		return vm.NotFoundNavigationResult()
	}

	// DEVIATION: Allow "" as a parent view to root dir
	if v.absoluteModulePath == "/" {
		v.modulePath = ""
		v.absoluteModulePath = ""
		return v.updateRealPaths()
	}

	numSlashes := strings.Count(v.absoluteModulePath, "/")
	if numSlashes <= 0 {
		return vm.OtherStringNavigationResult("numSlashes <= 0")
	}
	if numSlashes == 1 {
		v.modulePath = ""
		v.absoluteModulePath = ""
		return v.updateRealPaths()
	}

	v.modulePath = v.fs.NormalizePath(v.fs.Join(v.modulePath, ".."))
	v.absoluteModulePath = v.fs.NormalizePath(v.fs.Join(v.absoluteModulePath, ".."))
	return v.updateRealPaths()
}

func (v *vfsNavigator) toChild(name string) *vm.NavigationResult {
	v.modulePath = v.fs.NormalizePath(v.fs.Join(v.modulePath, name))
	v.absoluteModulePath = v.fs.NormalizePath(v.fs.Join(v.absoluteModulePath, name))
	return v.updateRealPaths()
}

func (v *vfsNavigator) getFilePath() string {
	return v.realPath
}

func (v *vfsNavigator) getAbsoluteFilePath() string {
	return v.absoluteRealPath
}

func (v *vfsNavigator) getLuaurcPath() string {
	directory := v.realPath

	for _, suffix := range INIT_SUFFIXES {
		if strings.HasSuffix(directory, suffix) {
			directory = directory[:len(directory)-len(suffix)]
			return v.fs.Join(directory, ".luaurc")
		}
	}
	for _, suffix := range SUFFIXES {
		if strings.HasSuffix(directory, suffix) {
			directory = directory[:len(directory)-len(suffix)]
			return v.fs.Join(directory, ".luaurc")
		}
	}
	return v.fs.Join(directory, ".luaurc")
}
