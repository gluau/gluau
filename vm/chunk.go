package vm

/*
#include "../rustlib/rustlib.h"
*/
import "C"
import "unsafe"

type ChunkMode int

const (
	ChunkModeText   ChunkMode = iota // Text mode for Lua chunks
	ChunkModeBinary                  // Binary mode for Lua chunks
)

// ChunkOpts represents the options for a Lua chunk.
type ChunkOpts struct {
	// The name of the chunk, used for debugging and error messages.
	Name string
	// The environment table for the chunk.
	//
	// This environment table is used as the _G/global table for the chunk.
	//
	// On Luau: consider using `safeenv` on Luau for better performance if possible
	Env *LuaTable
	// The chunks mode (either text or binary).
	//
	// Running binary chunks (bytecode) is dangerous. Maliciously crafted bytecode can cause
	// crashes or safety issues.
	Mode ChunkMode
	// The compiler options for the chunk.
	//
	// Not setting this will use the default compiler options.
	CompilerOpts *CompilerOpts
	// The code to run
	Code string
}

func newChunkString(s []byte) *C.struct_ChunkString {
	if len(s) == 0 {
		return nil // Return nil if the string is empty
	}
	chunkString := C.luago_chunk_string_new((*C.char)(unsafe.Pointer(&s[0])), C.size_t(len(s)))
	return chunkString
}
