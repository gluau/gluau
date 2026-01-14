//! String related ops

use std::ffi::c_char;

use mluau::Lua;

use crate::{result::{Errorable, GoValueV2Result, wrap_failable}, value_v2::GoLuaValueV2};

pub fn get_string(lua: &Lua, t: GoLuaValueV2) -> Option<mluau::String> {
    let v = t.to_value(lua);
    match v {
        mluau::Value::String(b) => Some(b),
        _ => None,
    }
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_create_string(ptr: *mut mluau::Lua, s: *const c_char, len: usize) -> GoValueV2Result {
    wrap_failable(|| {
        // Safety: Assume ptr is a valid, non-null pointer to a Lua
        // and that s points to a valid C string of length len.
        let lua = unsafe { &*ptr };

        let res = if s.is_null() {
            // Create an empty string if the pointer is null.
            lua.create_string("")
        } else {
            let slice = unsafe { std::slice::from_raw_parts(s as *const u8, len) };
            lua.create_string(slice)
        };

        match res {
            Ok(str) => GoValueV2Result::ok(GoLuaValueV2::from_value(lua, mluau::Value::String(str))),
            Err(err) => GoValueV2Result::err(format!("{err}"))
        }
    })
}

#[repr(C)]
pub struct LuaStringBytes {
    // Pointer to the string data
    pub data: *const u8,
    // Length of the string data
    pub size: usize,
}

impl Errorable for LuaStringBytes {
    fn error_variant(_s: String) -> Self {
        Self {
            data: std::ptr::null(),
            size: 0,
        }
    }
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_string_as_bytes(lua: *mut Lua, string: GoLuaValueV2) -> LuaStringBytes {
    wrap_failable(|| {
        let lua = unsafe { &*lua };
        let Some(string) = get_string(lua, string) else {
            return LuaStringBytes {
                data: std::ptr::null(),
                size: 0,
            };
        };
        
        // Return a pointer to the bytes of the Lua String.
        let bytes = string.as_bytes();
        LuaStringBytes {
            data: bytes.as_ptr(),
            size: bytes.len(),
        }
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_string_as_bytes_with_nul(lua: *mut Lua, string: GoLuaValueV2) -> LuaStringBytes {
    wrap_failable(|| {
        let lua = unsafe { &*lua };
        let Some(string) = get_string(lua, string) else {
            return LuaStringBytes {
                data: std::ptr::null(),
                size: 0,
            };
        };
        
        // Return a pointer to the bytes of the Lua String.
        let bytes = string.as_bytes_with_nul();
        LuaStringBytes {
            data: bytes.as_ptr(),
            size: bytes.len(),
        }
    })
}
