#include <stdint.h>
#include <stddef.h>
#include <stdbool.h>

struct Lua;

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

struct Lua* newluavm(uint32_t stdlib);
void luavm_setcompileropts(struct Lua* ptr, struct CompilerOpts opts);
struct GoNoneResult luavm_setmemorylimit(struct Lua* ptr, size_t limit);
size_t luago_used_memory(struct Lua* ptr);
size_t luago_memory_limit(struct Lua* ptr);
struct GoNoneResult luavm_sandbox(struct Lua* ptr, bool enabled);
struct LuaTable* luago_globals(struct Lua* ptr);
struct GoNoneResult luago_setglobals(struct Lua* ptr, struct LuaTable* globals);
struct LuaThread* luago_current_thread(struct Lua* ptr);
void luago_set_type_metatable(struct Lua* ptr, uint8_t typ, struct LuaTable* mt);
struct GoNoneResult luago_set_instruction_limit(struct Lua* ptr, uint64_t limit);
struct CancellationToken;
struct CancellationToken* luago_set_instruction_limit_with_cancel(struct Lua* ptr, uint64_t limit);
void luago_cancel_execution(struct CancellationToken* token);
void luago_free_cancellation_token(struct CancellationToken* token);
void freeluavm(struct Lua* ptr);

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

// Rust String API
char* luago_string_new(const char* str, size_t len);
void luago_string_free(char* result_error_ptr);

// Returns a GoResult[LuaString]
struct GoStringResult luago_create_string(struct Lua* ptr, const char* str, size_t len);
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
bool luago_string_equals(struct LuaString* a, struct LuaString* b);
void luago_free_string(struct LuaString* ptr);

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
    LuaValueTypeOther = 12
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
    void* other; // Placeholder for other types
} LuaValueData;

struct GoLuaValue {
    // The type of the Lua value
    LuaValueType tag;
    // The actual data of the Lua value 
    LuaValueData data;
};

struct GoLuaValue luago_value_clone(struct GoLuaValue value);
void luago_value_destroy(struct GoLuaValue value);

// Table API
struct LuaTable;
struct GoTableResult luago_create_table(struct Lua* ptr);
struct GoTableResult luago_create_table_with_capacity(struct Lua* ptr, size_t narr, size_t nrec);
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
    struct Lua* lua;
    struct GoMultiValue* args; // NOTE: Rust will deallocate this

    // Go side may set this to set a response
    struct GoMultiValue* values; // NOTE: Rust will deallocate this
    char* error; // NOTE: Rust will deallocate this
};
struct GoFunctionResult luago_create_function(struct Lua* ptr, struct IGoCallback cb);
struct GoMultiValueResult luago_function_call(struct LuaFunction* ptr, struct GoMultiValue* args);
uintptr_t luago_function_to_pointer(struct LuaFunction* ptr);
struct GoFunctionResult luago_function_deepclone(struct LuaFunction* ptr);
struct LuaTable* luago_function_environment(struct LuaFunction* ptr);
struct GoBoolResult luago_function_set_environment(struct LuaFunction* ptr, struct LuaTable* table);
bool luago_function_equals(struct LuaFunction* a, struct LuaFunction* b);
void luago_free_function(struct LuaFunction* f);

// Userdata API
struct LuaUserData;
struct DynamicData {
    uint64_t handle; // cgo handle to the data
    DropCallback drop; // cgo drop callback
};
struct GoUserDataResult luago_create_userdata(struct Lua* ptr, struct DynamicData data, struct LuaTable* mt);
struct GoUsizePtrResult luago_get_userdata_handle(struct LuaUserData* ptr);
uintptr_t luago_userdata_to_pointer(struct LuaUserData* ptr);
struct GoTableResult luago_userdata_metatable(struct LuaUserData* ptr);
bool luago_userdata_equals(struct LuaUserData* a, struct LuaUserData* b);
void luago_free_userdata(struct LuaUserData* ptr);

// Thread API
struct LuaThread;
struct GoThreadResult luago_create_thread(struct Lua* ptr, struct LuaFunction* func);
struct GoNoneResult luago_reset_thread(struct LuaThread* ptr, struct LuaFunction* func);
uint8_t luago_thread_status(struct LuaThread* ptr);
struct GoNoneResult luago_thread_sandbox(struct LuaThread* ptr);
struct GoMultiValueResult luago_thread_resume(struct LuaThread* ptr, struct GoMultiValue* args);
struct GoMultiValueResult luago_thread_resume_error(struct LuaThread* ptr, struct GoLuaValue error);
uintptr_t luago_thread_to_pointer(struct LuaThread* ptr);
struct GoNoneResult luago_yield_with(struct Lua* ptr, struct GoMultiValue* args);
bool luago_thread_equals(struct LuaThread* a, struct LuaThread* b);
void luago_free_thread(struct LuaThread* ptr);

// Buffer API
struct LuaBuffer;
struct GoBufferResult luago_create_buffer(struct Lua* ptr, const char* s, size_t len);
uintptr_t luago_buffer_to_pointer(struct LuaBuffer* ptr);
bool luago_buffer_equals(struct LuaBuffer* a, struct LuaBuffer* b);
struct LuaStringBytes luago_buffer_to_bytes(struct LuaBuffer* ptr);
struct LuaStringBytes luago_buffer_read_bytes(struct LuaBuffer* ptr, size_t offset, size_t len);
void luago_buffer_write_bytes(struct LuaBuffer* ptr, size_t offset, const char* bytes, size_t len);
void luago_buffer_free_bytes(struct LuaStringBytes bytes);
size_t luago_buffer_len(struct LuaBuffer* ptr);
void luago_free_buffer(struct LuaBuffer* ptr);

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
struct GoThreadResult {
    // Pointer to the LuaThread value
    struct LuaThread* value;
    // Pointer to a null-terminated C string for the error message
    char* error;
};
struct GoBufferResult {
    // Pointer to the LuaBuffer value
    struct LuaBuffer* value;
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
struct GoFunctionResult luago_load_chunk(struct Lua* ptr, struct ChunkOpts opts);

// Interrupt API
struct InterruptData {
    // Pointer to the Lua
    struct Lua* lua;

    // Go side may set this to set a response
    uint8_t vm_state;
    const char* error; // NOTE: Rust will deallocate this
};
void luago_set_interrupt(struct Lua* ptr, struct IGoCallback cb);
void luago_remove_interrupt(struct Lua* ptr);

// Registry
struct GoNoneResult luago_set_named_registry_value(struct Lua* ptr, const char* key, size_t keylen, struct GoLuaValue value);
struct GoValueResult luago_named_registry_value(struct Lua* ptr, const char* key, size_t keylen);

// Require API
struct GoNavigationResult {
    bool not_found;
    bool ambiguous;
    char* other; // Rust will deallocate this automatically. Should be allocated with moveString
};

struct IsRequireAllowed {
    char* chunk_name; // Go will free this automatically with the moveStringToGo function
    bool data; // Go may set this to true if the require is allowed
};

struct ResetOrJumpToAliasOrToChild {
    char* str; // Go will free this automatically with the moveStringToGo function
    struct GoNavigationResult data; // Go may set this to true if the require is allowed
};

struct ToParent {
    struct GoNavigationResult data; // Go may set this to true if the require is allowed
};

struct HasModuleOrHasConfig {
    bool data; // Go may set this to true if the require is allowed
};

struct CacheKey {
    char* data; // Rust will deallocate this automatically. Should be allocated with moveStringToRust
};

struct Config {
    char* data; // Pointer to a null-terminated C string for the configuration data
    char* error; // Pointer to a null-terminated C string for the error message
};

struct Loader {
    struct Lua* lua; // Pointer to the Lua instance
    struct LuaFunction* function; // Go side may set this in response
    char* error; // Pointer to a null-terminated C string for the error message
};

struct GoRequire {
    // Checks if the require is allowed for the given chunk name
    struct IGoCallback is_require_allowed;
    // Resets the require state
    struct IGoCallback reset;
    // Jumps to an alias or to a child module
    struct IGoCallback jump_to_alias;
    // Navigates to the parent module
    struct IGoCallback to_parent;
    // Navigates to a child module
    struct IGoCallback to_child;
    // Checks if a module exists
    struct IGoCallback has_module;
    // Gets the cache key for a module
    struct IGoCallback cache_key;
    // Checks if a configuration exists
    struct IGoCallback has_config;
    // Gets the configuration data
    struct IGoCallback config;
    // Gets the loader function for the current module
    struct IGoCallback loader;
};
struct GoFunctionResult luago_create_require_function(struct Lua* ptr, struct GoRequire gr);