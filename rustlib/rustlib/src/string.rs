//! String related ops

use std::ffi::c_char;

use crate::{result::GoResult, LuaVmWrapper};

#[unsafe(no_mangle)]
pub extern "C" fn luago_create_string(ptr: *mut LuaVmWrapper, s: *const c_char, len: usize) -> *mut GoResult  {
    // Safety: Assume ptr is a valid, non-null pointer to a LuaVmWrapper
    // and that s points to a valid C string of length len.
    let res = unsafe {
        let lua = &mut (*ptr).lua;
        let slice = std::slice::from_raw_parts(s as *const u8, len);
        lua.create_string(slice)
    };

    GoResult::new_ptr(res)
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_free_string(string: *mut mluau::String) {
    // Safety: Assume string is a valid, non-null pointer to a Lua String
    if string.is_null() {
        return;
    }

    // Re-box the Lua String pointer to manage its memory automatically.
    unsafe { drop(Box::from_raw(string)) };
}