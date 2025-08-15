package vm

/*
#include "../rustlib/rustlib.h"
*/
import "C"

type OptimizationLevel int

const (
	OptimizationLevelNone  OptimizationLevel = iota // No optimization
	OptimizationLevelBasic                          // Basic optimization
	OptimizationLevelFull                           // Full optimization (may impact debugging)
)

type DebugLevel int

const (
	DebugLevelNone     DebugLevel = iota // No debug support
	DebugLevelLineInfo                   // Line info + function names only
	DebugLevelFull                       // Full debug info with local + upvalues
)

type TypeInfoLevel int

const (
	TypeInfoLevelNativeModules TypeInfoLevel = iota // Generate type info for native modules only
	TypeInfoLevelAllModules                         // Generate type info for all modules
)

type CoverageLevel int

const (
	CoverageLevelNone  CoverageLevel = iota // No coverage information
	CoverageLevelBasic                      // Basic coverage information (statement coverage)
	CoverageLevelFull                       // Full coverage information (statement + expression coverage)
)

// CompilerOpts represents the options for compiling a Lua chunk.
//
// Not all Luau compiler options are supported yet.
type CompilerOpts struct {
	// The optimization level for the Lua chunk.
	// 0 is no optimization, 1 is basic optimization, 2 is full optimization (which may impact debugging)
	OptimizationLevel OptimizationLevel

	// The debug level for the Lua chunk.
	// 0 = no debug support, 1 = line info + func names only, 2 = full debug info with local+upvalues
	DebugLevel DebugLevel

	// The Luau type information level
	//
	// 0 = generate for native modules, 1 = generate for all modules
	//
	// Not very useful in gluau
	TypeInfoLevel TypeInfoLevel

	// The coverage level to use
	//
	// 0 = no coverage information, 1 = basic coverage information (statement coverage), 2 = full coverage information (statement + expression coverage)
	CoverageLevel CoverageLevel
}

// Converts CompilerOpts to C struct
func (opts *CompilerOpts) toC() C.struct_CompilerOpts {
	return C.struct_CompilerOpts{
		optimization_level: C.uint8_t(opts.OptimizationLevel),
		debug_level:        C.uint8_t(opts.DebugLevel),
		type_info_level:    C.uint8_t(opts.TypeInfoLevel),
		coverage_level:     C.uint8_t(opts.CoverageLevel),
	}
}
