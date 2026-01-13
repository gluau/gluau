// Example demonstrating Go-Luau interop with gluau
//
// Flow: Go -> Luau -> Go function -> Luau -> Go (result)
//
// Build with: go build -tags musl (on Alpine/musl systems)
// Run with: go run -tags musl main.go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/koeng101/gluau/vm"
)

func main() {
	// Create a new Luau VM
	lua, err := vm.CreateLuaVm()
	if err != nil {
		log.Fatal("Failed to create Luau VM:", err)
	}
	defer lua.Close()

	// Set resource limits for running untrusted code safely
	// Memory limit: 1MB
	err = lua.SetMemoryLimit(1 * 1024 * 1024)
	if err != nil {
		log.Fatal("Failed to set memory limit:", err)
	}
	// Instruction limit: 1 million (prevents infinite loops)
	err = lua.SetInstructionLimit(1_000_000)
	if err != nil {
		log.Fatal("Failed to set instruction limit:", err)
	}

	// Create a Go function that appends strings
	// This will be called from Luau
	appendFunc, err := lua.CreateFunction(func(cb *vm.CallbackLua, args []vm.Value) ([]vm.Value, error) {
		if len(args) < 2 {
			return nil, fmt.Errorf("append requires 2 string arguments")
		}

		// Get the first string argument
		str1, ok := args[0].(*vm.ValueString)
		if !ok {
			return nil, fmt.Errorf("argument 1 must be a string, got %s", args[0].Type())
		}

		// Get the second string argument
		str2, ok := args[1].(*vm.ValueString)
		if !ok {
			return nil, fmt.Errorf("argument 2 must be a string, got %s", args[1].Type())
		}

		// Append the strings in Go
		result := str1.Value().String() + str2.Value().String()
		fmt.Printf("[Go] Appending '%s' + '%s' = '%s'\n", str1.Value().String(), str2.Value().String(), result)

		// Return the result back to Luau
		return []vm.Value{vm.GoString(result)}, nil
	})
	if err != nil {
		log.Fatal("Failed to create Go function:", err)
	}

	// Set the function as a global in Luau
	err = lua.Globals().Set(vm.GoString("append_strings"), appendFunc.ToValue())
	if err != nil {
		log.Fatal("Failed to set global:", err)
	}

	// Luau code that:
	// 1. Calls the Go function to append strings
	// 2. Calls it again with the result
	// 3. Returns the final result back to Go
	luauCode := `
		local first = append_strings("Hello, ", "Luau")
		local second = append_strings(first, " from ")
		local final = append_strings(second, "Go!")
		return final
	`

	// Load and run the Luau code
	chunk, err := lua.LoadChunk(vm.ChunkOpts{
		Name: "example",
		Code: luauCode,
	})
	if err != nil {
		log.Fatal("Failed to load chunk:", err)
	}
	defer chunk.Close()

	fmt.Println("[Go] Executing Luau code...")
	fmt.Println()

	// Call the Luau function
	results, err := chunk.Call()
	if err != nil {
		log.Fatal("Failed to call chunk:", err)
	}

	// Get the result back in Go
	if len(results) > 0 {
		if str, ok := results[0].(*vm.ValueString); ok {
			fmt.Println()
			fmt.Printf("[Go] Final result from Luau: '%s'\n", str.Value().String())
		}
	}

	// Clean up results
	for _, r := range results {
		r.Close()
	}

	// Example: Running untrusted code with a timeout
	fmt.Println()
	fmt.Println("=== Running untrusted code with timeout ===")
	runWithTimeout()
}

// runWithTimeout demonstrates running untrusted Luau code with resource limits and a timeout
func runWithTimeout() {
	lua, err := vm.CreateLuaVm()
	if err != nil {
		log.Fatal(err)
	}
	defer lua.Close()

	// Set resource limits
	lua.SetMemoryLimit(1 * 1024 * 1024) // 1MB memory limit

	// Instruction limit with cancellation token for timeout support
	token, err := lua.SetInstructionLimitWithCancel(100_000_000)
	if err != nil {
		log.Fatal(err)
	}
	defer token.Close()

	// This code would run forever without limits
	code := `
		local i = 0
		while true do
			i = i + 1
		end
		return i
	`

	chunk, err := lua.LoadChunk(vm.ChunkOpts{Name: "untrusted", Code: code})
	if err != nil {
		log.Fatal(err)
	}
	defer chunk.Close()

	// Run with 100ms timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		_, err := chunk.Call()
		done <- err
	}()

	select {
	case err := <-done:
		fmt.Printf("Completed with: %v\n", err)
	case <-ctx.Done():
		token.Cancel() // Signal graceful stop
		err := <-done  // Wait for clean termination
		fmt.Printf("Timed out and cancelled: %v\n", err)
	}
}
