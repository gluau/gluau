#include <stdint.h>
#include <stddef.h>
#include <stdbool.h>

struct LuaVmWrapper;

// CompilerOpts API

struct CompilerOpts {
    // The optimization level for the Lua chunk.
    uint8_t optimization_level;
    // The debug level for the Lua chunk.
    uint8_t debug_level;
    // The Luau type information level
    uint8_t type_info_level;
    // The coverage level to use
    uint8_t coverage_level;
};

struct LuaVmWrapper* newluavm();
void luavm_setcompileropts(struct LuaVmWrapper* ptr, struct CompilerOpts opts);
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
struct GoValueResult luago_table_raw_get(struct LuaTable* ptr, struct GoLuaValue key);
struct GoNoneResult luago_table_raw_insert(struct LuaTable* ptr, int64_t idx, struct GoLuaValue value);
size_t luago_table_raw_len(struct LuaTable* ptr);
struct GoValueResult luago_table_raw_pop(struct LuaTable* ptr);
struct GoNoneResult luago_table_raw_push(struct LuaTable* ptr, struct GoLuaValue value);
struct GoNoneResult luago_table_raw_remove(struct LuaTable* ptr, struct GoLuaValue key);
struct GoNoneResult luago_table_raw_set(struct LuaTable* ptr, struct GoLuaValue key, struct GoLuaValue value);
struct GoNoneResult luago_table_set(struct LuaTable* ptr, struct GoLuaValue key, struct GoLuaValue value);
struct TableForEachValueCallbackData {
    struct GoLuaValue value;
    // Go code may modify the below
    bool stop;
};
struct GoNoneResult luago_table_foreach_value(struct LuaTable* ptr, struct IGoCallback cb);
struct GoNoneResult luago_table_set_metatable(struct LuaTable* ptr, struct LuaTable* mt);
void luago_table_set_readonly(struct LuaTable* ptr, bool enabled);
void luago_table_set_safeenv(struct LuaTable* ptr, bool enabled);
uintptr_t luago_table_to_pointer(struct LuaTable* ptr);
char* luago_table_debug(struct LuaTable* ptr);
void luago_free_table(struct LuaTable* ptr);

// Functions
struct LuaFunction;
struct FunctionCallbackData {
    struct LuaVmWrapper* lua;
    struct GoMultiValue* args; // NOTE: Rust will deallocate this

    // Go side may set this to set a response
    struct GoMultiValue* values; // NOTE: Rust will deallocate this
    struct ErrorVariant *error; // NOTE: Rust will deallocate this
};
struct GoFunctionResult luago_create_function(struct LuaVmWrapper* ptr, struct IGoCallback cb);
struct GoMultiValueResult luago_function_call(struct LuaFunction* ptr, struct GoMultiValue* args);
void luago_free_function(struct LuaFunction* f);

// Userdata API
struct LuaUserData;
struct DynamicData {
    uint64_t handle; // cgo handle to the data
    DropCallback drop; // cgo drop callback
};
struct GoUserDataResult luago_create_userdata(struct LuaVmWrapper* ptr, struct DynamicData data, struct LuaTable* mt);
struct GoUsizePtrResult luago_get_userdata_handle(struct LuaUserData* ptr);
void luago_free_userdata(struct LuaUserData* ptr);

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
struct GoUsizePtrResult {
    uintptr_t value;
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
struct GoFunctionResult {
    // Pointer to the LuaTable value
    struct LuaFunction* value;
    // Pointer to a null-terminated C string for the error message
    char* error;
};
struct GoUserDataResult {
    // Pointer to the LuaUserData value
    struct LuaUserData* value;
    // Pointer to a null-terminated C string for the error message
    char* error;
};
struct GoMultiValueResult {
    struct GoMultiValue* value;
    char* error;
};

struct GoValueResult {
    // The Lua value
    struct GoLuaValue value;
    // Pointer to a null-terminated C string for the error message
    char* error;
};

// Result types end

// Multivalue handling
struct GoMultiValue;
struct GoMultiValue* luago_create_multivalue_with_capacity(size_t capacity);
void luago_multivalue_push(struct GoMultiValue* mv, struct GoLuaValue value);
size_t luago_multivalue_len(struct GoMultiValue* mv);
struct GoLuaValue luago_multivalue_pop(struct GoMultiValue* mv);
void luago_free_multivalue(struct GoMultiValue* mv);
// Multivalue handling end

// ChunkOpts API
struct ChunkString;
struct ChunkString* luago_chunk_string_new(const char* bytes, size_t len);

struct ChunkOpts {
    // The name of the chunk, used for debugging and error messages.
    struct ChunkString* name;
    // The environment table for the chunk.
    struct LuaTable* env;
    // The chunks mode (either text or binary).
    uint8_t mode;
    // The compiler options for the chunk.
    struct CompilerOpts* compiler_opts;
    // The actual code of the chunk.
    struct ChunkString* code;
};
struct GoFunctionResult luago_load_chunk(struct LuaVmWrapper* ptr, struct ChunkOpts opts);