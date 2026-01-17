use std::ffi::c_char;

use mluau::{Lua, Value};

use crate::{VmHandle, externalstring::GoOwnedBytes, result::{GoValueV2Result, wrap_failable}, value_v2::GoLuaValueV2};

fn get_buffer(lua: &Lua, t: GoLuaValueV2) -> Option<mluau::Buffer> {
    let v = t.to_value(lua);
    match v {
        mluau::Value::Buffer(b) => Some(b),
        _ => None,
    }
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_create_buffer(ptr: VmHandle, s: *const c_char, len: usize) -> GoValueV2Result  {
    wrap_failable(|| {
        let lua = ptr.get();
        let res = if s.is_null() {
            // Create an empty string if the pointer is null.
            lua.create_buffer(&[])
        } else {
            let slice = unsafe { std::slice::from_raw_parts(s as *const u8, len) };
            lua.create_buffer(slice)
        };

        match res {
            Ok(buf) => GoValueV2Result::ok(GoLuaValueV2::from_value(&lua, Value::Buffer(buf))),
            Err(err) => GoValueV2Result::err(format!("{err}"))
        }
    })
}


#[unsafe(no_mangle)]
pub extern "C" fn luago_buffer_len(lua: VmHandle, buf: GoLuaValueV2) -> usize {
    wrap_failable(|| {
        let lua = lua.get();
        let Some(buf) = get_buffer(&lua, buf) else {
            return 0
        };

        buf.len()
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_buffer_to_bytes(lua: VmHandle, ptr: GoLuaValueV2, buf: GoOwnedBytes) -> usize {
    wrap_failable(|| {
        // Safety: Assume string is a valid, non-null pointer to a Lua String
        let lua = lua.get();
        let Some(mbuf) = get_buffer(&lua, ptr) else {
            return 0;
        };
        
        // Return a pointer to the bytes of the Lua String.
        mbuf.with_bytes(|bytes| {
            buf.copy_rust_bytes_to_go(bytes);
            bytes.len()
        })
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_buffer_read_bytes(lua: VmHandle, ptr: GoLuaValueV2, offset: usize, len: usize, buf: GoOwnedBytes) -> usize {
    wrap_failable(|| {
        // Safety: Assume string is a valid, non-null pointer to a Lua String
        let lua = lua.get();
        let Some(mbuf) = get_buffer(&lua, ptr) else {
            return 0;
        };
        
        // Return a pointer to the bytes of the Lua String.
        mbuf.with_bytes(|bytes| {
            if offset >= bytes.len() {
                return 0; // offset beyond byte length
            }
            // Clamp the end index to the actual length
            let available_len = bytes.len() - offset;
            let copy_len: usize = std::cmp::min(len, available_len);
            
            buf.copy_rust_bytes_to_go(&bytes[offset..offset + copy_len]);
            copy_len
        })
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_buffer_write_bytes(lua: VmHandle, buf: GoLuaValueV2, offset: usize, data: *const c_char, len: usize) {
    wrap_failable(|| {
        let lua = lua.get();
        let Some(buf) = get_buffer(&lua, buf) else {
            return;
        };
        let slice = unsafe { std::slice::from_raw_parts(data as *const u8, len) };
        buf.write_bytes(offset, slice);
    })
}
