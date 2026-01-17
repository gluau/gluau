//! String related ops

use std::ffi::c_char;

use mluau::Lua;

use crate::{VmHandle, externalstring::GoOwnedBytes, result::{GoValueV2Result, wrap_failable}, value_v2::GoLuaValueV2};

pub fn get_string(lua: &Lua, t: GoLuaValueV2) -> Option<mluau::String> {
    let v = t.to_value(lua);
    match v {
        mluau::Value::String(b) => Some(b),
        _ => None,
    }
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_create_string(ptr: VmHandle, s: *const c_char, len: usize) -> GoValueV2Result {
    wrap_failable(|| {
        let lua = ptr.get();
        let res = if s.is_null() {
            // Create an empty string if the pointer is null.
            lua.create_string("")
        } else {
            let slice = unsafe { std::slice::from_raw_parts(s as *const u8, len) };
            lua.create_string(slice)
        };

        match res {
            Ok(str) => GoValueV2Result::ok(GoLuaValueV2::from_value(&lua, mluau::Value::String(str))),
            Err(err) => GoValueV2Result::err(format!("{err}"))
        }
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_string_len(lua: VmHandle, string: GoLuaValueV2) -> usize {
    wrap_failable(|| {
        let lua = lua.get();
        let Some(string) = get_string(&lua, string) else {
            return 0
        };

        string.as_bytes().len()
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_string_as_bytes(ptr: VmHandle, string: GoLuaValueV2, buf: GoOwnedBytes) {
    wrap_failable(|| {
        let lua = ptr.get();
        let Some(string) = get_string(&lua, string) else {
            return;
        };
        
        let bytes = string.as_bytes();
        buf.copy_rust_bytes_to_go(&bytes);
    })
}
