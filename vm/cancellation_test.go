//go:build cgo

package vm

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestCancellationStopsExecution(t *testing.T) {
	luavm, err := CreateLuaVm()
	if err != nil {
		t.Fatal(err)
	}
	defer luavm.Close()

	token, err := luavm.SetInstructionLimitWithCancel(100_000_000) // High limit
	if err != nil {
		t.Fatal(err)
	}
	defer token.Close()

	code := `while true do end`
	chunk, err := luavm.LoadChunk(ChunkOpts{Name: "infinite", Code: code})
	if err != nil {
		t.Fatal(err)
	}
	defer chunk.Close()

	// Cancel after a short delay
	go func() {
		time.Sleep(10 * time.Millisecond)
		token.Cancel()
	}()

	_, err = chunk.Call()
	if err == nil {
		t.Fatal("expected error from cancellation, got nil")
	}
	if !strings.Contains(err.Error(), "cancelled") {
		t.Fatalf("expected 'cancelled' error, got: %v", err)
	}
}

func TestCancellationWithInstructionLimit(t *testing.T) {
	luavm, err := CreateLuaVm()
	if err != nil {
		t.Fatal(err)
	}
	defer luavm.Close()

	token, err := luavm.SetInstructionLimitWithCancel(100_000) // Low limit
	if err != nil {
		t.Fatal(err)
	}
	defer token.Close()

	code := `while true do end`
	chunk, err := luavm.LoadChunk(ChunkOpts{Name: "infinite", Code: code})
	if err != nil {
		t.Fatal(err)
	}
	defer chunk.Close()

	// Don't cancel - let instruction limit trigger
	_, err = chunk.Call()
	if err == nil {
		t.Fatal("expected error from instruction limit, got nil")
	}
	if !strings.Contains(err.Error(), "instruction limit exceeded") {
		t.Fatalf("expected 'instruction limit exceeded' error, got: %v", err)
	}
}

func TestCancellationAllowsNormalCode(t *testing.T) {
	luavm, err := CreateLuaVm()
	if err != nil {
		t.Fatal(err)
	}
	defer luavm.Close()

	token, err := luavm.SetInstructionLimitWithCancel(1_000_000)
	if err != nil {
		t.Fatal(err)
	}
	defer token.Close()

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

// RunUntrusted demonstrates the recommended pattern for running untrusted code
func RunUntrusted(code string, timeout time.Duration, maxInstructions uint64) ([]Value, error) {
	luavm, err := CreateLuaVm()
	if err != nil {
		return nil, err
	}
	defer luavm.Close()

	// Memory limit
	luavm.SetMemoryLimit(1 * 1024 * 1024)

	// Instruction limit with cancellation
	token, err := luavm.SetInstructionLimitWithCancel(maxInstructions)
	if err != nil {
		return nil, err
	}
	defer token.Close()

	chunk, err := luavm.LoadChunk(ChunkOpts{Name: "untrusted", Code: code})
	if err != nil {
		return nil, err
	}
	defer chunk.Close()

	// Run with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	type result struct {
		values []Value
		err    error
	}
	done := make(chan result, 1)

	go func() {
		values, err := chunk.Call()
		done <- result{values, err}
	}()

	select {
	case r := <-done:
		return r.values, r.err
	case <-ctx.Done():
		token.Cancel()
		r := <-done // Wait for graceful termination
		return r.values, r.err
	}
}

func TestRunUntrustedNormal(t *testing.T) {
	code := `
		local sum = 0
		for i = 1, 1000 do
			sum = sum + i
		end
		return sum
	`
	results, err := RunUntrusted(code, 5*time.Second, 1_000_000)
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	defer func() {
		for _, r := range results {
			r.Close()
		}
	}()
	if results[0].String() != "ValueInteger: 500500" {
		t.Fatalf("expected 500500, got %s", results[0].String())
	}
}

func TestRunUntrustedTimeout(t *testing.T) {
	code := `while true do end`
	_, err := RunUntrusted(code, 50*time.Millisecond, 100_000_000)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "cancelled") {
		t.Fatalf("expected 'cancelled' error, got: %v", err)
	}
}

func TestRunUntrustedInstructionLimit(t *testing.T) {
	code := `while true do end`
	_, err := RunUntrusted(code, 5*time.Second, 100_000)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "instruction limit exceeded") {
		t.Fatalf("expected 'instruction limit exceeded' error, got: %v", err)
	}
}

// Benchmarks

func BenchmarkCancellationToken(b *testing.B) {
	code := `
		local sum = 0
		for i = 1, 10000 do
			sum = sum + i
		end
		return sum
	`

	b.Run("NoLimit", func(b *testing.B) {
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
	})

	b.Run("RustLimitWithCancel", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			luavm, _ := CreateLuaVm()
			token, _ := luavm.SetInstructionLimitWithCancel(10_000_000)
			chunk, _ := luavm.LoadChunk(ChunkOpts{Name: "bench", Code: code})
			results, _ := chunk.Call()
			for _, r := range results {
				r.Close()
			}
			chunk.Close()
			token.Close()
			luavm.Close()
		}
	})

	b.Run("RustLimitOnly", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			luavm, _ := CreateLuaVm()
			luavm.SetInstructionLimit(10_000_000)
			chunk, _ := luavm.LoadChunk(ChunkOpts{Name: "bench", Code: code})
			results, _ := chunk.Call()
			for _, r := range results {
				r.Close()
			}
			chunk.Close()
			luavm.Close()
		}
	})
}
