#include <stdint.h>
#include <stddef.h>
#include <stdbool.h>

struct LuaVmWrapper;

struct LuaVmWrapper* newluavm();
struct GoResult luavm_setmemorylimit(struct LuaVmWrapper* ptr, size_t limit);
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

// Note: only deallocates the `GoResult` error, not the value
void luago_result_error_free(char* result_error_ptr);

// Returns a GoResult[LuaString]
struct GoResult luago_create_string(struct LuaVmWrapper* ptr, const char* str, size_t len);
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
void luago_free_string(struct LuaString* ptr);

struct ErrorVariant;

// GoLuaValue related stuff

typedef enum LuaValueType {
    LuaValueTypeNil = 0,
    LuaValueTypeBoolean = 1,
    LuaValueTypeLightUserData = 2,
    LuaValueTypeInteger = 3,
    LuaValueTypeNumber = 4,
    LuaValueTypeVector = 5,
    LuaValueTypeString = 6,
    LuaValueTypeTable = 7,
    LuaValueTypeFunction = 8,
    LuaValueTypeThread = 9,
    LuaValueTypeUserData = 10,
    LuaValueTypeBuffer = 11,
    LuaValueTypeError = 12,
    LuaValueTypeOther = 13
} LuaValueType;

typedef union LuaValueData {
    bool boolean;
    void* light_userdata; // no drop needed for light userdata
    int64_t integer;
    double number;
    float vector[3]; // 3d vector
    struct LuaString* string; // Pointer to LuaString
    void* table; // Pointer to LuaTable
    void* function; // Pointer to LuaFunction
    void* thread; // Pointer to LuaThread
    void* userdata; // Pointer to LuaUserData
    void* buffer; // Pointer to LuaBuffer
    struct ErrorVariant* error; // Pointer to LuaError
    void* other; // Placeholder for other types
} LuaValueData;

struct GoLuaValue {
    // The type of the Lua value
    LuaValueType tag;
    // The actual data of the Lua value 
    LuaValueData data;
};

struct GoLuaValue luago_value_clone(struct GoLuaValue value);

struct ErrorVariant* luago_error_new(const char* str, size_t len);
struct LuaStringBytes luago_error_get_string(struct ErrorVariant* ptr);
void luago_error_free(struct ErrorVariant* ptr);

struct DebugValue {
    // Array of two GoLuaValues for debugging purposes
    struct GoLuaValue values[3];
};

// Debug API
struct DebugValue luago_dbg_value(struct LuaVmWrapper* ptr);

// Table API
struct LuaTable;
struct GoResult luago_create_table(struct LuaVmWrapper* ptr);
struct GoResult luago_create_table_with_capacity(struct LuaVmWrapper* ptr, size_t narr, size_t nrec);
struct GoResult luago_table_clear(struct LuaTable* ptr);
struct GoResult luago_table_contains_key(struct LuaTable* ptr, struct GoLuaValue key);
void luago_free_table(struct LuaTable* ptr);
