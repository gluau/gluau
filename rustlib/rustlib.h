#include <stdint.h>
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