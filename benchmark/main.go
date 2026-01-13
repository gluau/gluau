// Benchmark comparing gluau (Luau) vs gopher-lua performance
//
// Run with: go run -tags musl main.go
//
// Results (Alpine Linux, x86_64):
//
//	| Benchmark                    | Gluau  | Gopher-Lua | Winner                 |
//	|------------------------------|--------|------------|------------------------|
//	| Fibonacci(30)                | 99ms   | 311ms      | Gluau 3.1x faster      |
//	| Loop (10M iter)              | 151ms  | 877ms      | Gluau 5.8x faster      |
//	| String naive (s=s.."x") 20K  | 212ms  | 70ms       | Gopher-Lua 3x faster   |
//	| String table.concat 20K      | 2ms    | 5ms        | Gluau 2.4x faster      |
//	| Table ops (100K)             | 5ms    | 23ms       | Gluau 4.4x faster      |
//
// Summary: Gluau is 3-6x faster for compute-heavy tasks due to Luau's
// optimized bytecode compiler.
//
// String concatenation: Naive (s = s .. "x") favors gopher-lua because strings
// are immutable - each concat allocates a new string, and Go's allocator is
// faster than Rust's for this pattern. Using table.concat (the idiomatic
// approach), gluau wins at scale. Note: gopher-lua needs RegistrySize increased
// for large tables to avoid "registry overflow" errors.
package main

import (
	"fmt"
	"time"

	"github.com/koeng101/gluau/vm"
	lua "github.com/yuin/gopher-lua"
)

func main() {
	fmt.Println("=== Gluau vs Gopher-Lua Benchmark ===")
	fmt.Println()

	// Benchmark 1: Fibonacci (recursive, tests function call overhead)
	benchmarkFibonacci()

	// Benchmark 2: Loop with arithmetic (tests basic operations)
	benchmarkLoop()

	// Benchmark 3: String concatenation (naive - demonstrates immutability cost)
	benchmarkStringsNaive()

	// Benchmark 4: String concatenation (efficient - using table.concat)
	benchmarkStringsEfficient()

	// Benchmark 5: Table operations
	benchmarkTables()
}

func benchmarkFibonacci() {
	fmt.Println("--- Fibonacci(30) recursive ---")

	code := `
		local function fib(n)
			if n < 2 then return n end
			return fib(n-1) + fib(n-2)
		end
		return fib(30)
	`

	// Gluau
	start := time.Now()
	result := runGluau(code)
	gluauTime := time.Since(start)
	fmt.Printf("Gluau:      %v (result: %s)\n", gluauTime, result)

	// Gopher-lua
	start = time.Now()
	result = runGopherLua(code)
	gopherTime := time.Since(start)
	fmt.Printf("Gopher-Lua: %v (result: %s)\n", gopherTime, result)

	printSpeedup(gluauTime, gopherTime)
	fmt.Println()
}

func benchmarkLoop() {
	fmt.Println("--- Loop with arithmetic (10M iterations) ---")

	code := `
		local sum = 0
		for i = 1, 10000000 do
			sum = sum + i * 2 - 1
		end
		return sum
	`

	// Gluau
	start := time.Now()
	result := runGluau(code)
	gluauTime := time.Since(start)
	fmt.Printf("Gluau:      %v (result: %s)\n", gluauTime, result)

	// Gopher-lua
	start = time.Now()
	result = runGopherLua(code)
	gopherTime := time.Since(start)
	fmt.Printf("Gopher-Lua: %v (result: %s)\n", gopherTime, result)

	printSpeedup(gluauTime, gopherTime)
	fmt.Println()
}

func benchmarkStringsNaive() {
	fmt.Println("--- String naive s=s..\"x\" (20K iterations) ---")

	code := `
		local s = ""
		for i = 1, 20000 do
			s = s .. "x"
		end
		return #s
	`

	// Gluau
	start := time.Now()
	result := runGluau(code)
	gluauTime := time.Since(start)
	fmt.Printf("Gluau:      %v (result: %s)\n", gluauTime, result)

	// Gopher-lua
	start = time.Now()
	result = runGopherLua(code)
	gopherTime := time.Since(start)
	fmt.Printf("Gopher-Lua: %v (result: %s)\n", gopherTime, result)

	printSpeedup(gluauTime, gopherTime)
	fmt.Println()
}

func benchmarkStringsEfficient() {
	fmt.Println("--- String table.concat (20K iterations) ---")

	code := `
		local t = {}
		for i = 1, 20000 do
		    t[i] = "x"
		end
		return #table.concat(t)
	`

	// Gluau
	start := time.Now()
	result := runGluau(code)
	gluauTime := time.Since(start)
	fmt.Printf("Gluau:      %v (result: %s)\n", gluauTime, result)

	// Gopher-lua
	start = time.Now()
	result = runGopherLua(code)
	gopherTime := time.Since(start)
	fmt.Printf("Gopher-Lua: %v (result: %s)\n", gopherTime, result)

	printSpeedup(gluauTime, gopherTime)
	fmt.Println()
}

func benchmarkTables() {
	fmt.Println("--- Table insert/access (100K operations) ---")

	code := `
		local t = {}
		for i = 1, 100000 do
			t[i] = i * 2
		end
		local sum = 0
		for i = 1, 100000 do
			sum = sum + t[i]
		end
		return sum
	`

	// Gluau
	start := time.Now()
	result := runGluau(code)
	gluauTime := time.Since(start)
	fmt.Printf("Gluau:      %v (result: %s)\n", gluauTime, result)

	// Gopher-lua
	start = time.Now()
	result = runGopherLua(code)
	gopherTime := time.Since(start)
	fmt.Printf("Gopher-Lua: %v (result: %s)\n", gopherTime, result)

	printSpeedup(gluauTime, gopherTime)
	fmt.Println()
}

func runGluau(code string) string {
	luavm, err := vm.CreateLuaVm()
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}
	defer luavm.Close()

	chunk, err := luavm.LoadChunk(vm.ChunkOpts{
		Name: "benchmark",
		Code: code,
	})
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}
	defer chunk.Close()

	results, err := chunk.Call()
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}

	if len(results) > 0 {
		defer results[0].Close()
		return results[0].String()
	}
	return "nil"
}

func runGopherLua(code string) string {
	L := lua.NewState(lua.Options{RegistrySize: 1024 * 64})
	defer L.Close()

	if err := L.DoString(code); err != nil {
		return fmt.Sprintf("error: %v", err)
	}

	result := L.Get(-1)
	L.Pop(1)

	return result.String()
}

func printSpeedup(gluauTime, gopherTime time.Duration) {
	if gluauTime < gopherTime {
		speedup := float64(gopherTime) / float64(gluauTime)
		fmt.Printf("=> Gluau is %.2fx faster\n", speedup)
	} else {
		speedup := float64(gluauTime) / float64(gopherTime)
		fmt.Printf("=> Gopher-Lua is %.2fx faster\n", speedup)
	}
}
