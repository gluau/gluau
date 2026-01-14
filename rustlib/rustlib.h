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
struct GoLuaValueV2 luago_globals(struct Lua* ptr);
struct GoNoneResult luago_setglobals(struct Lua* ptr, struct GoLuaValueV2 globals);
struct GoLuaValueV2 luago_current_thread(struct Lua* ptr);
// given mt is a table.
void luago_set_type_metatable(struct Lua* ptr, uint8_t typ, struct GoLuaValueV2 mt);
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

// Rust String API
char* luago_string_new(const char* str, size_t len);
void luago_string_free(char* result_error_ptr);

// Returns a GoResult[LuaString]
struct GoValueV2Result luago_create_string(struct Lua* lua, const char* str, size_t len);

struct LuaStringBytes {
    // Pointer to the string data
    const char* data;
    // Length of the string data
    size_t len;
};

struct LuaStringBytes luago_string_as_bytes(struct Lua* lua, struct GoLuaValueV2 ptr);
struct LuaStringBytes luago_string_as_bytes_with_nul(struct Lua* lua, struct GoLuaValueV2 ptr);

// GoLuaValueV2 related stuff

typedef enum LuaValueTypeV2 {
    LuaValueTypeV2Nil = 0,
    LuaValueTypeV2Boolean = 1,
    LuaValueTypeV2LightUserData = 2,
    LuaValueTypeV2Integer = 3,
    LuaValueTypeV2Number = 4,
    LuaValueTypeV2Vector = 5,
    LuaValueTypeV2String = 6,
    LuaValueTypeV2Table = 7,
    LuaValueTypeV2Function = 8,
    LuaValueTypeV2Thread = 9,
    LuaValueTypeV2UserData = 10,
    LuaValueTypeV2Buffer = 11,
    LuaValueTypeV2Other = 12
} LuaValueTypeV2;

struct Handle {
    int64_t index;
    int64_t generation;
};

typedef union LuaValueDataV2 {
    bool boolean;
    int64_t integer;
    double number;
    float vector[3]; // 3d vector
    struct Handle handle; // for string, table, function, thread, userdata, buffer, other
    void* lightuserdata; // no drop needed for light userdata
} LuaValueDataV2;

struct GoLuaValueV2 {
    // The type of the Lua value 
    LuaValueTypeV2 tag;
    // The actual data of the Lua value 
    LuaValueDataV2 data;
};

void luago_valuev2_destroy(struct Lua* lua, struct GoLuaValueV2 value);
uintptr_t luago_valuev2_topointer(struct Lua* lua, struct GoLuaValueV2 ptr);
struct GoBoolResult luago_valuev2_equals(struct Lua* lua, struct GoLuaValueV2 a, struct GoLuaValueV2 b);

struct GoLuaValueV2Array {
    struct GoLuaValueV2* values;
    size_t size;
};

struct GoLuaValueV2Array luago_valuev2array_alloc(size_t size);
void luago_valuev2array_destroy(struct GoLuaValueV2Array arr);

// Table API
struct GoValueV2Result luago_create_table(struct Lua* lua);
struct GoValueV2Result luago_create_table_with_capacity(struct Lua* lua, size_t narr, size_t nrec);
struct GoNoneResult luago_table_clear(struct Lua* lua, struct GoLuaValueV2 ptr);
struct GoBoolResult luago_table_contains_key(struct Lua* lua, struct GoLuaValueV2 ptr, struct GoLuaValueV2 key);
struct TableForEachCallbackData {
    struct GoLuaValueV2 key;
    struct GoLuaValueV2 value;
    // Go code may modify the below
    bool stop;
};
struct GoNoneResult luago_table_foreach(struct Lua* lua, struct GoLuaValueV2 ptr, struct IGoCallback cb);
struct GoValueV2Result luago_table_get(struct Lua* lua, struct GoLuaValueV2 ptr, struct GoLuaValueV2 key);
bool luago_table_is_empty(struct Lua* lua, struct GoLuaValueV2 ptr);
bool luago_table_is_readonly(struct Lua* lua, struct GoLuaValueV2 ptr);
struct GoI64Result luago_table_len(struct Lua* lua, struct GoLuaValueV2 ptr);
struct GoLuaValueV2 luago_table_metatable(struct Lua* lua, struct GoLuaValueV2 ptr);
struct GoValueV2Result luago_table_pop(struct Lua* lua, struct GoLuaValueV2 ptr);
struct GoNoneResult luago_table_push(struct Lua* lua, struct GoLuaValueV2 ptr, struct GoLuaValueV2 value);
struct GoValueV2Result luago_table_raw_get(struct Lua* lua, struct GoLuaValueV2 ptr, struct GoLuaValueV2 key);
struct GoNoneResult luago_table_raw_insert(struct Lua* lua, struct GoLuaValueV2 ptr, int64_t idx, struct GoLuaValueV2 value);
size_t luago_table_raw_len(struct Lua* lua, struct GoLuaValueV2 ptr);
struct GoValueV2Result luago_table_raw_pop(struct Lua* lua, struct GoLuaValueV2 ptr);
struct GoNoneResult luago_table_raw_push(struct Lua* lua, struct GoLuaValueV2 ptr, struct GoLuaValueV2 value);
struct GoNoneResult luago_table_raw_remove(struct Lua* lua, struct GoLuaValueV2 ptr, struct GoLuaValueV2 key);
struct GoNoneResult luago_table_raw_set(struct Lua* lua, struct GoLuaValueV2 ptr, struct GoLuaValueV2 key, struct GoLuaValueV2 value);
struct GoNoneResult luago_table_set(struct Lua* lua, struct GoLuaValueV2 ptr, struct GoLuaValueV2 key, struct GoLuaValueV2 value);
struct TableForEachValueCallbackData {
    struct GoLuaValueV2 value;
    // Go code may modify the below
    bool stop;
};
struct GoNoneResult luago_table_foreach_value(struct Lua* lua, struct GoLuaValueV2 ptr, struct IGoCallback cb);
struct GoNoneResult luago_table_set_metatable(struct Lua* lua, struct GoLuaValueV2 ptr, struct GoLuaValueV2 mt);
void luago_table_set_readonly(struct Lua* lua, struct GoLuaValueV2 ptr, bool enabled);
void luago_table_set_safeenv(struct Lua* lua, struct GoLuaValueV2 ptr, bool enabled);
char* luago_table_debug(struct Lua* lua, struct GoLuaValueV2 ptr);

// Functions
struct FunctionCallbackData {
    struct Lua* lua;
    struct GoLuaValueV2Array args; // NOTE: Rust will deallocate this so go must copy this if needed

    // Go side may set this to set a response
    struct GoLuaValueV2Array values; // NOTE: Rust will deallocate this
    char* error; // NOTE: Rust will deallocate this
};
struct GoValueV2Result luago_create_function(struct Lua* lua, struct IGoCallback cb);
struct GoLuaValueV2ArrayResult luago_function_call(struct Lua* lua, struct GoLuaValueV2 ptr, struct GoLuaValueV2Array args);
struct GoValueV2Result luago_function_deepclone(struct Lua* lua, struct GoLuaValueV2 ptr);
struct GoLuaValueV2 luago_function_environment(struct Lua* lua, struct GoLuaValueV2 ptr);
struct GoBoolResult luago_function_set_environment(struct Lua* lua, struct GoLuaValueV2 ptr, struct GoLuaValueV2 table);

// Userdata API
struct DynamicData {
    uintptr_t handle; // cgo handle to the data
    DropCallback drop; // cgo drop callback
};
struct GoValueV2Result luago_create_userdata(struct Lua* lua, struct DynamicData data, struct GoLuaValueV2 mt);
struct GoUsizePtrResult luago_get_userdata_handle(struct Lua* lua, struct GoLuaValueV2 ptr);
struct GoValueV2Result luago_userdata_metatable(struct Lua* lua, struct GoLuaValueV2 ptr);

// Thread API
struct GoValueV2Result luago_create_thread(struct Lua* lua, struct GoLuaValueV2 func);
struct GoNoneResult luago_reset_thread(struct Lua* lua, struct GoLuaValueV2 ptr, struct GoLuaValueV2 func);
uint8_t luago_thread_status(struct Lua* lua, struct GoLuaValueV2 ptr);
struct GoNoneResult luago_thread_sandbox(struct Lua* lua, struct GoLuaValueV2 ptr);
struct GoLuaValueV2ArrayResult luago_thread_resume(struct Lua* lua, struct GoLuaValueV2 ptr, struct GoLuaValueV2Array args);
struct GoLuaValueV2ArrayResult luago_thread_resume_error(struct Lua* lua, struct GoLuaValueV2 ptr, struct GoLuaValueV2 error);
struct GoNoneResult luago_yield_with(struct Lua* lua, struct GoLuaValueV2Array args);

// Buffer API
struct GoValueV2Result luago_create_buffer(struct Lua* ptr, const char* s, size_t len);
struct LuaStringBytes luago_buffer_to_bytes(struct Lua* lua, struct GoLuaValueV2 ptr);
struct LuaStringBytes luago_buffer_read_bytes(struct Lua* lua, struct GoLuaValueV2 ptr, size_t offset, size_t len);
void luago_buffer_write_bytes(struct Lua* lua, struct GoLuaValueV2 ptr, size_t offset, const char* bytes, size_t len);
void luago_buffer_free_bytes(struct LuaStringBytes bytes);
size_t luago_buffer_len(struct Lua* lua, struct GoLuaValueV2 ptr);

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
struct GoValueV2Result {
    // The Lua value
    struct GoLuaValueV2 value;
    // Pointer to a null-terminated C string for the error message
    char* error;
};
struct GoLuaValueV2ArrayResult {
    // The array of Lua values
    struct GoLuaValueV2Array value;
    // Pointer to a null-terminated C string for the error message
    char* error;
};

// Result types end

// ChunkOpts API
struct ChunkString;
struct ChunkString* luago_chunk_string_new(const char* bytes, size_t len);

struct ChunkOpts {
    // The name of the chunk, used for debugging and error messages.
    struct ChunkString* name;
    // The environment table for the chunk.
    struct GoLuaValueV2 env;
    // The chunks mode (either text or binary).
    uint8_t mode;
    // The compiler options for the chunk.
    struct CompilerOpts* compiler_opts;
    // The actual code of the chunk.
    struct ChunkString* code;
};
struct GoValueV2Result luago_load_chunk(struct Lua* ptr, struct ChunkOpts opts);

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
struct GoNoneResult luago_set_named_registry_value(struct Lua* ptr, const char* key, size_t keylen, struct GoLuaValueV2 value);
struct GoValueV2Result luago_named_registry_value(struct Lua* ptr, const char* key, size_t keylen);

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
    struct GoLuaValueV2 function; // Go side may set this in response
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
struct GoValueV2Result luago_create_require_function(struct Lua* ptr, struct GoRequire gr);