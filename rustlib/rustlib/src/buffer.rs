use std::ffi::c_char;

use mluau::{Lua, Value};

use crate::{VmHandle, result::{GoValueV2Result, wrap_failable}, string::LuaStringBytes, value_v2::GoLuaValueV2};

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
pub extern "C" fn luago_buffer_to_bytes(lua: VmHandle, buf: GoLuaValueV2) -> LuaStringBytes {
    wrap_failable(|| {
        // Safety: Assume string is a valid, non-null pointer to a Lua String
        let lua = lua.get();
        let Some(buf) = get_buffer(&lua, buf) else {
            return LuaStringBytes {
                data: std::ptr::null(),
                size: 0,
            };
        };
        
        // Return a pointer to the bytes of the Lua String.
        let bytes = buf.to_vec().into_boxed_slice();
        let bytes_ptr = bytes.as_ptr();
        let bytes_len = bytes.len();
        std::mem::forget(bytes); // Prevent deallocation of the bytes
        LuaStringBytes {
            data: bytes_ptr,
            size: bytes_len,
        }
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_buffer_read_bytes(lua: VmHandle, buf: GoLuaValueV2, offset: usize, len: usize) -> LuaStringBytes {
    wrap_failable(|| {
        // Safety: Assume string is a valid, non-null pointer to a Lua String
        let lua = lua.get();
        let Some(buf) = get_buffer(&lua, buf) else {
            return LuaStringBytes {
                data: std::ptr::null(),
                size: 0,
            };
        };
        
        // Return a pointer to the bytes of the Lua String.
        let bytes = buf.read_bytes_to_vec(offset, len).into_boxed_slice();
        let bytes_ptr = bytes.as_ptr();
        let bytes_len = bytes.len();
        std::mem::forget(bytes); // Prevent deallocation of the bytes
        LuaStringBytes {
            data: bytes_ptr,
            size: bytes_len,
        }
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

#[unsafe(no_mangle)]
pub extern "C" fn luago_buffer_free_bytes(buf: LuaStringBytes) {
    wrap_failable(|| {
        if buf.data.is_null() {
            return; // Nothing to free
        }

        let s = std::ptr::slice_from_raw_parts_mut(buf.data as *mut u8, buf.size);
        unsafe {
            drop(Box::from_raw(s));
        }
    })
}
