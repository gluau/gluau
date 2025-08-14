#include <stdint.h>
#include <stddef.h>
#include <stdbool.h>

struct LuaVmWrapper;

struct LuaVmWrapper* newluavm();
struct GoNoneResult luavm_setmemorylimit(struct LuaVmWrapper* ptr, size_t limit);
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

// Note: only deallocates the `GoResult` error, not the value
void luago_result_error_free(char* result_error_ptr);

// Returns a GoResult[LuaString]
struct GoStringResult luago_create_string(struct LuaVmWrapper* ptr, const char* str, size_t len);
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
    struct GoLuaValue values[4];
};

// Debug API
struct DebugValue luago_dbg_value(struct LuaVmWrapper* ptr);

// Table API
struct LuaTable;
struct GoTableResult luago_create_table(struct LuaVmWrapper* ptr);
struct GoTableResult luago_create_table_with_capacity(struct LuaVmWrapper* ptr, size_t narr, size_t nrec);
struct GoNoneResult luago_table_clear(struct LuaTable* ptr);
struct GoBoolResult luago_table_contains_key(struct LuaTable* ptr, struct GoLuaValue key);
struct GoBoolResult luago_table_equals(struct LuaTable* ptr, struct LuaTable* other);
struct TableForEachCallbackData {
    struct GoLuaValue key;
    struct GoLuaValue value;
    // Go code may modify the below
    bool stop;
};
struct GoNoneResult luago_table_foreach(struct LuaTable* ptr, struct IGoCallback cb);
struct GoValueResult luago_table_get(struct LuaTable* ptr, struct GoLuaValue key);
bool luago_table_is_empty(struct LuaTable* ptr);
bool luago_table_is_readonly(struct LuaTable* ptr);
struct GoI64Result luago_table_len(struct LuaTable* ptr);
struct LuaTable* luago_table_metatable(struct LuaTable* ptr);
struct GoValueResult luago_table_pop(struct LuaTable* ptr);
struct GoNoneResult luago_table_push(struct LuaTable* ptr, struct GoLuaValue value);
void luago_free_table(struct LuaTable* ptr);

// Result types

struct GoNoneResult {
    char* error;
};
struct GoBoolResult {
    bool value;
    char* error;
};
struct GoI64Result {
    int64_t value;
    char* error;
};
struct GoStringResult {
    // Pointer to the string value
    struct LuaString* value;
    // Pointer to a null-terminated C string for the error message
    char* error;
};
struct GoTableResult {
    // Pointer to the LuaTable value
    struct LuaTable* value;
    // Pointer to a null-terminated C string for the error message
    char* error;
};
struct GoValueResult {
    // The Lua value
    struct GoLuaValue value;
    // Pointer to a null-terminated C string for the error message
    char* error;
};

// Result types end
