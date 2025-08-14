use mluau::Lua;

use crate::{result::GoResult, LuaVmWrapper};

// Base functions

#[unsafe(no_mangle)]
pub extern "C" fn newluavm() -> *mut LuaVmWrapper {
    let lua = Lua::new();

    lua.set_on_close(|| {
        println!("Lua VM is being closed");
    });

    let wrapper = Box::new(LuaVmWrapper { lua });
    Box::into_raw(wrapper)
}

#[unsafe(no_mangle)]
pub extern "C" fn luavm_setmemorylimit(ptr: *mut LuaVmWrapper, limit: usize) -> GoResult {
    // Safety: Assume the Lua VM is valid and we can set its memory limit.
    if ptr.is_null() {
        return GoResult::from_error(mluau::Error::external("LuaVmWrapper pointer is null".to_string()));
    }
    let lua = unsafe { &(*ptr).lua };
    match lua.set_memory_limit(limit) {
        Ok(_) => GoResult::from_value(true),
        Err(err) => GoResult::from_error(err),
    }
}

#[unsafe(no_mangle)]
pub extern "C" fn freeluavm(ptr: *mut LuaVmWrapper) {
    // Safety: Assume ptr is a valid, non-null pointer to a LuaVmWrapper
    // and that ownership is being transferred back to Rust to be dropped.
    unsafe {
        drop(Box::from_raw(ptr));
    }
}