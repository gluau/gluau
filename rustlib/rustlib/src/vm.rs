use mluau::Lua;

use crate::{result::GoNoneResult, LuaVmWrapper};

// Base functions

#[unsafe(no_mangle)]
pub extern "C-unwind" fn newluavm() -> *mut LuaVmWrapper {
    let lua = Lua::new_with(
        mluau::StdLib::ALL_SAFE, // TODO: Allow configuring this
        mluau::LuaOptions::new()
        .catch_rust_panics(false)
        .disable_error_userdata(true)
    ).unwrap(); // Will never error, as we are using safe libraries only.

    lua.set_on_close(|| {
        println!("Lua VM is being closed");
    });

    let wrapper = Box::new(LuaVmWrapper { lua });
    Box::into_raw(wrapper)
}

#[unsafe(no_mangle)]
pub extern "C-unwind" fn luavm_setmemorylimit(ptr: *mut LuaVmWrapper, limit: usize) -> GoNoneResult {
    // Safety: Assume the Lua VM is valid and we can set its memory limit.
    if ptr.is_null() {
        return GoNoneResult::err("LuaVmWrapper pointer is null".to_string());
    }
    let lua = unsafe { &(*ptr).lua };
    match lua.set_memory_limit(limit) {
        Ok(_) => GoNoneResult::ok(),
        Err(err) => GoNoneResult::err(format!("{err}")),
    }
}

#[unsafe(no_mangle)]
pub extern "C-unwind" fn freeluavm(ptr: *mut LuaVmWrapper) {
    // Safety: Assume ptr is a valid, non-null pointer to a LuaVmWrapper
    // and that ownership is being transferred back to Rust to be dropped.
    unsafe {
        drop(Box::from_raw(ptr));
    }
}