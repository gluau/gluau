//go:build cgo

package vm

import (
	"strings"
	"testing"
	"time"
)

func TestInstructionLimitStopsInfiniteLoop(t *testing.T) {
	luavm, err := CreateLuaVm()
	if err != nil {
		t.Fatal(err)
	}
	defer luavm.Close()

	err = luavm.SetInstructionLimit(100_000)
	if err != nil {
		t.Fatal(err)
	}

	code := `while true do end`
	chunk, err := luavm.LoadChunk(ChunkOpts{Name: "infinite", Code: code})
	if err != nil {
		t.Fatal(err)
	}
	defer chunk.Close()

	_, err = chunk.Call()
	if err == nil {
		t.Fatal("expected error from instruction limit, got nil")
	}
	if !strings.Contains(err.Error(), "instruction limit exceeded") {
		t.Fatalf("expected 'instruction limit exceeded' error, got: %v", err)
	}
}

func TestInstructionLimitAllowsNormalCode(t *testing.T) {
	luavm, err := CreateLuaVm()
	if err != nil {
		t.Fatal(err)
	}
	defer luavm.Close()

	err = luavm.SetInstructionLimit(1_000_000)
	if err != nil {
		t.Fatal(err)
	}

	// Simple loop that should complete well under the limit
	code := `
		local sum = 0
		for i = 1, 1000 do
			sum = sum + i
		end
		return sum
	`
	chunk, err := luavm.LoadChunk(ChunkOpts{Name: "normal", Code: code})
	if err != nil {
		t.Fatal(err)
	}
	defer chunk.Close()

	results, err := chunk.Call()
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	defer func() {
		for _, r := range results {
			r.Close()
		}
	}()

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].String() != "ValueInteger: 500500" {
		t.Fatalf("expected 500500, got %s", results[0].String())
	}
}

// BenchmarkNoLimit measures execution without any limits
func BenchmarkNoLimit(b *testing.B) {
	code := `
		local sum = 0
		for i = 1, 10000 do
			sum = sum + i
		end
		return sum
	`
	for i := 0; i < b.N; i++ {
		luavm, _ := CreateLuaVm()
		chunk, _ := luavm.LoadChunk(ChunkOpts{Name: "bench", Code: code})
		results, _ := chunk.Call()
		for _, r := range results {
			r.Close()
		}
		chunk.Close()
		luavm.Close()
	}
}

// BenchmarkWithInstructionLimit measures execution with Rust-based instruction limit
func BenchmarkWithInstructionLimit(b *testing.B) {
	code := `
		local sum = 0
		for i = 1, 10000 do
			sum = sum + i
		end
		return sum
	`
	for i := 0; i < b.N; i++ {
		luavm, _ := CreateLuaVm()
		luavm.SetInstructionLimit(1_000_000)
		chunk, _ := luavm.LoadChunk(ChunkOpts{Name: "bench", Code: code})
		results, _ := chunk.Call()
		for _, r := range results {
			r.Close()
		}
		chunk.Close()
		luavm.Close()
	}
}

// BenchmarkWithGoInterrupt measures execution with Go-based interrupt (CGO crossing each time)
func BenchmarkWithGoInterrupt(b *testing.B) {
	code := `
		local sum = 0
		for i = 1, 10000 do
			sum = sum + i
		end
		return sum
	`
	for i := 0; i < b.N; i++ {
		luavm, _ := CreateLuaVm()
		count := 0
		luavm.SetInterrupt(func(cb *CallbackLua) (VmState, error) {
			count++
			if count > 1_000_000 {
				return VmStateContinue, nil
			}
			return VmStateContinue, nil
		})
		chunk, _ := luavm.LoadChunk(ChunkOpts{Name: "bench", Code: code})
		results, _ := chunk.Call()
		for _, r := range results {
			r.Close()
		}
		chunk.Close()
		luavm.Close()
	}
}

// BenchmarkInstructionLimitTightLoop measures overhead on a very tight loop
func BenchmarkInstructionLimitTightLoop(b *testing.B) {
	code := `
		local x = 0
		for i = 1, 100000 do
			x = x + 1
		end
		return x
	`

	b.Run("NoLimit", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			luavm, _ := CreateLuaVm()
			chunk, _ := luavm.LoadChunk(ChunkOpts{Name: "tight", Code: code})
			results, _ := chunk.Call()
			for _, r := range results {
				r.Close()
			}
			chunk.Close()
			luavm.Close()
		}
	})

	b.Run("RustLimit", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			luavm, _ := CreateLuaVm()
			luavm.SetInstructionLimit(10_000_000)
			chunk, _ := luavm.LoadChunk(ChunkOpts{Name: "tight", Code: code})
			results, _ := chunk.Call()
			for _, r := range results {
				r.Close()
			}
			chunk.Close()
			luavm.Close()
		}
	})

	b.Run("GoInterrupt", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			luavm, _ := CreateLuaVm()
			deadline := time.Now().Add(10 * time.Second)
			luavm.SetInterrupt(func(cb *CallbackLua) (VmState, error) {
				if time.Now().After(deadline) {
					return VmStateContinue, nil
				}
				return VmStateContinue, nil
			})
			chunk, _ := luavm.LoadChunk(ChunkOpts{Name: "tight", Code: code})
			results, _ := chunk.Call()
			for _, r := range results {
				r.Close()
			}
			chunk.Close()
			luavm.Close()
		}
	})
}
