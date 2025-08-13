#include <stdint.h>
#include <stddef.h>
struct LuaVmWrapper;

struct LuaVmWrapper* newluavm();
void freeluavm(struct LuaVmWrapper* ptr);

typedef void (*Callback)(void* val, uintptr_t handle);
typedef void (*DropCallback)(uintptr_t handle);

struct IGoCallback {
    // Callback function pointer
    Callback callback;
    // Drop function pointer
    DropCallback drop;
    // Handle to pass to the callback
    uintptr_t handle;
};

// Test callbacks
void test_callback(struct IGoCallback* cb, void* val);

// Represents a result from Rust that can be handled by a C-compatible language.
// It contains either a value or an error.
struct GoResult {
    // A generic pointer to the successful result value.
    // If this is not NULL, the operation was successful.
    void* value;

    // A pointer to a null-terminated C string for the error message.
    // If this is not NULL, the operation failed.
    char* error;
};

// Note: only deallocates the `GoResult` struct and error, not the value
void luago_result_free(struct GoResult* ptr);

// Returns a GoResult[LuaString]
struct GoResult* luago_create_string(struct LuaVmWrapper* ptr, const char* str, size_t len);
struct LuaString;

struct LuaStringBytes {
    // Pointer to the string data
    const char* data;
    // Length of the string data
    size_t len;
};

struct LuaStringBytes luago_string_as_bytes(struct LuaString* ptr);
struct LuaStringBytes luago_string_as_bytes_with_nul(struct LuaString* ptr);
uintptr_t luago_string_to_pointer(struct LuaString* ptr);

// Free's a LuaString
void luago_free_string(struct LuaString* ptr);