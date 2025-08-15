//! String related ops

use std::ffi::c_char;

use crate::{result::GoStringResult, LuaVmWrapper};

#[unsafe(no_mangle)]
pub extern "C-unwind" fn luago_create_string(ptr: *mut LuaVmWrapper, s: *const c_char, len: usize) -> GoStringResult  {
    // Safety: Assume ptr is a valid, non-null pointer to a LuaVmWrapper
    // and that s points to a valid C string of length len.
    let lua = unsafe { &(*ptr).lua };

    let res = if s.is_null() {
        // Create an empty string if the pointer is null.
        lua.create_string("")
    } else {
        let slice = unsafe { std::slice::from_raw_parts(s as *const u8, len) };
        lua.create_string(slice)
    };

    match res {
        Ok(str) => GoStringResult::ok(Box::into_raw(Box::new(str))),
        Err(err) => GoStringResult::err(format!("{err}"))
    }
}

#[repr(C)]
pub struct LuaStringBytes {
    // Pointer to the string data
    pub data: *const u8,
    // Length of the string data
    pub size: usize,
}

#[unsafe(no_mangle)]
pub extern "C-unwind" fn luago_string_as_bytes(string: *mut mluau::String) -> LuaStringBytes {
    // Safety: Assume string is a valid, non-null pointer to a Lua String
    if string.is_null() {
        return LuaStringBytes {
            data: std::ptr::null(),
            size: 0,
        };
    }

    let lua_string = unsafe { &*string };
    
    // Return a pointer to the bytes of the Lua String.
    let bytes = lua_string.as_bytes();
    LuaStringBytes {
        data: bytes.as_ptr(),
        size: bytes.len(),
    }
}

#[unsafe(no_mangle)]
pub extern "C-unwind" fn luago_string_as_bytes_with_nul(string: *mut mluau::String) -> LuaStringBytes {
    // Safety: Assume string is a valid, non-null pointer to a Lua String
    if string.is_null() {
        return LuaStringBytes {
            data: std::ptr::null(),
            size: 0,
        };
    }

    let lua_string = unsafe { &*string };
    
    // Return a pointer to the bytes of the Lua String.
    let bytes = lua_string.as_bytes_with_nul();
    LuaStringBytes {
        data: bytes.as_ptr(),
        size: bytes.len(),
    }
}

#[unsafe(no_mangle)]
pub extern "C-unwind" fn luago_string_to_pointer(string: *mut mluau::String) -> usize {
    // Safety: Assume string is a valid, non-null pointer to a Lua String
    if string.is_null() {
        return 0;
    }

    let lua_string = unsafe { &*string };

    let ptr = lua_string.to_pointer();

    ptr as usize
}

#[unsafe(no_mangle)]
pub extern "C-unwind" fn luago_free_string(string: *mut mluau::String) {
    // Safety: Assume string is a valid, non-null pointer to a Lua String
    if string.is_null() {
        return;
    }

    // Re-box the Lua String pointer to manage its memory automatically.
    unsafe { drop(Box::from_raw(string)) };
}