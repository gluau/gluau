use mluau::Lua;

use crate::LuaVmWrapper;

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
pub extern "C" fn freeluavm(ptr: *mut LuaVmWrapper) {
    // Safety: Assume ptr is a valid, non-null pointer to a LuaVmWrapper
    // and that ownership is being transferred back to Rust to be dropped.
    unsafe {
        drop(Box::from_raw(ptr));
    }
}