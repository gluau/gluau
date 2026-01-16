use std::ffi::{c_char, c_void, CString};

use mluau::Lua;

use crate::{IGoCallback, IGoCallbackWrapper, VmHandle, compiler::CompilerOpts, result::{GoNoneResult, GoValueV2Result, GoVmHandleResult, wrap_failable}, table::get_table, value_v2::{GoLuaValueV2, GoLuaValueV2Array}};

// Represents the different standard libraries that can be loaded into the Luau VM.
bitflags::bitflags! {
    pub struct StdLib: u32 {
        const COROUTINE = 1 << 0;
        const TABLE = 1 << 1;
        const OS = 1 << 2;
        const STRING = 1 << 3;
        const UTF8 = 1 << 4;
        const BIT = 1 << 5;
        const MATH = 1 << 6;
        const BUFFER = 1 << 7;
        const VECTOR = 1 << 8;
        const DEBUG = 1 << 9;
        const ALL = 1 << 31;
    }
}

impl StdLib {
    pub fn to_mluau(self) -> mluau::StdLib {
        if self.contains(StdLib::ALL) {
            return mluau::StdLib::ALL_SAFE; // Return all safe libraries
        }
       let mut libs = mluau::StdLib::NONE;
       if self.contains(StdLib::COROUTINE) {
            libs |= mluau::StdLib::COROUTINE;
        }
        if self.contains(StdLib::TABLE) {
            libs |= mluau::StdLib::TABLE;
        }
        if self.contains(StdLib::OS) {
            libs |= mluau::StdLib::OS;
        }
        if self.contains(StdLib::STRING) {
            libs |= mluau::StdLib::STRING;
        }
        if self.contains(StdLib::UTF8) {
            libs |= mluau::StdLib::UTF8;
        }
        if self.contains(StdLib::BIT) {
            libs |= mluau::StdLib::BIT;
        }
        if self.contains(StdLib::MATH) {
            libs |= mluau::StdLib::MATH;
        }
        if self.contains(StdLib::BUFFER) {
            libs |= mluau::StdLib::BUFFER;
        }
        if self.contains(StdLib::VECTOR) {
            libs |= mluau::StdLib::VECTOR;
        }
        if self.contains(StdLib::DEBUG) {
            libs |= mluau::StdLib::DEBUG;
        }
        libs
    }
}

#[unsafe(no_mangle)]
pub extern "C" fn newluavm(stdlib: u32) -> GoVmHandleResult {
    wrap_failable(|| {
        let stdlib = StdLib::from_bits_truncate(stdlib);
        let lua = Lua::new_with(
            stdlib.to_mluau(), // TODO: Allow configuring this
            mluau::LuaOptions::new()
            .catch_rust_panics(false)
            .disable_error_userdata(true)
        ).unwrap(); // Will never error, as we are using safe libraries only.
        lua.set_on_close(|| {
            println!("Lua VM is being closed");
        });
        GoVmHandleResult::ok(VmHandle::new(lua))
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn luavm_setcompileropts(ptr: VmHandle, opts: CompilerOpts) {
    wrap_failable(|| {
        let lua = ptr.get();
        lua.set_compiler(opts.to_compiler());
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn luavm_setmemorylimit(ptr: VmHandle, limit: usize) -> GoNoneResult {
    wrap_failable(|| {
        
        let lua = ptr.get();
        match lua.set_memory_limit(limit) {
            Ok(_) => GoNoneResult::ok(),
            Err(err) => GoNoneResult::err(format!("{err}")),
        }
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn luavm_sandbox(ptr: VmHandle, enabled: bool) -> GoNoneResult {
    wrap_failable(|| {
        // Safety: Assume the Lua VM is valid and we can set its sandbox mode.
        
        let lua = ptr.get();
        match lua.sandbox(enabled) {
            Ok(_) => GoNoneResult::ok(),
            Err(err) => GoNoneResult::err(format!("{err}")),
        }
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_globals(ptr: VmHandle) -> GoLuaValueV2 {
    wrap_failable(|| {
        // Safety: Assume ptr is a valid, non-null pointer to a Lua VM
       let lua = ptr.get();
        GoLuaValueV2::from_value(&lua, mluau::Value::Table(lua.globals()))
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_setglobals(ptr: VmHandle, tab: GoLuaValueV2) -> GoNoneResult {
    wrap_failable(|| {
        let lua = ptr.get();
        let tab = match get_table(&lua, tab) {
            Some(t) => t,
            None => return GoNoneResult::err("Provided value is not a table".to_string()),
        };
        match lua.set_globals(tab) {
            Ok(_) => GoNoneResult::ok(),
            Err(err) => GoNoneResult::err(format!("{err}")),
        }
    })
}

#[repr(C)]
pub struct InterruptData {
    // mluau::Lua representing the Lua State
    // as called from Lua.
    //
    // This means that (future) API's like Lua.CurrentThread will return
    // the correct thread when using this Lua.
    pub lua: VmHandle,
   // Go side may set this to set a response
    pub vm_state: u8,
    pub error: *mut c_char,
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_set_interrupt(ptr: VmHandle, cb: IGoCallback)  {
    wrap_failable(|| {
        // Safety: Assume ptr is a valid, non-null pointer to a mluau::Lua
        let cb_wrapper = IGoCallbackWrapper::new(cb);
       let lua = ptr.get();
        lua.set_interrupt(move |lua| {
            let lua_ptr = VmHandle::new(lua.clone());
            
            let data = InterruptData {
                lua: lua_ptr,
                vm_state: 0, // Default state (Continue)
                error: std::ptr::null_mut(), // No error by default
            };
           let ptr = Box::into_raw(Box::new(data));
            cb_wrapper.callback(ptr as *mut c_void);
            let data = unsafe { Box::from_raw(ptr) };
           if !data.error.is_null() {
                let error = unsafe { CString::from_raw(data.error) };
                return Err(mluau::Error::external(error.to_string_lossy()));
            }
            
            match data.vm_state {
                0 => Ok(mluau::VmState::Continue), // Continue
                1 => Ok(mluau::VmState::Yield),    // Yield
                _ => Err(mluau::Error::external("Invalid VM state".to_string())),
            }
        });
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_remove_interrupt(ptr: VmHandle)  {
    wrap_failable(|| {
        // Safety: Assume ptr is a valid, non-null pointer to a Lua

       let lua = ptr.get();
        lua.remove_interrupt();
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_current_thread(ptr: VmHandle) -> GoLuaValueV2 {
    wrap_failable(|| {
        // Safety: Assume ptr is a valid, non-null pointer to a Lua

       let lua = ptr.get();
       GoLuaValueV2::from_value(&lua, mluau::Value::Thread(lua.current_thread()))
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_yield_with(ptr: VmHandle, args: GoLuaValueV2Array) -> GoNoneResult {
    wrap_failable(|| {
        let lua = ptr.get();
        let values = args.to_values(&lua);
       let lua = ptr.get();
       match lua.yield_with(values) {
            Ok(_) => GoNoneResult::ok(),
            Err(err) => GoNoneResult::err(format!("{err}")),
        }
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_used_memory(ptr: VmHandle) -> usize {
    wrap_failable(|| {
        let lua = ptr.get();
        lua.used_memory()
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_memory_limit(ptr: VmHandle) -> usize {
    wrap_failable(|| {
        let lua = ptr.get();
        lua.memory_limit().unwrap_or(0)
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_set_type_metatable(ptr: VmHandle, typ: u8, tab: GoLuaValueV2) {
    wrap_failable(|| {
        let lua = ptr.get();
        let tab = get_table(&lua, tab);
        match typ {
            0 => lua.set_type_metatable::<bool>(tab),
            1 => lua.set_type_metatable::<mluau::LightUserData>(tab),
            2 => lua.set_type_metatable::<mluau::Number>(tab),
            3 => lua.set_type_metatable::<mluau::Vector>(tab),
            4 => lua.set_type_metatable::<mluau::String>(tab),
            5 => lua.set_type_metatable::<mluau::Function>(tab),
            6 => lua.set_type_metatable::<mluau::Thread>(tab),
            7 => lua.set_type_metatable::<mluau::Buffer>(tab),
            _ => return
        }
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_set_named_registry_value(ptr: VmHandle, key: *const c_char, keylen: usize, value: GoLuaValueV2) -> GoNoneResult {
    wrap_failable(|| {
        
       let lua = ptr.get();
        let value = value.to_value(&lua);
        let key = if key.is_null() {
            unsafe { std::str::from_utf8_unchecked(&[]) }
        } else {
            let key = unsafe { std::slice::from_raw_parts(key as *const u8, keylen) };
            match std::str::from_utf8(key) {
                Ok(s) => s,
                Err(_) => return GoNoneResult::err("Invalid UTF-8 key".to_string()),
            }
        };
        match lua.set_named_registry_value(key, value) {
            Ok(_) => GoNoneResult::ok(),
            Err(err) => GoNoneResult::err(format!("{err}")),
        }
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_named_registry_value(ptr: VmHandle, key: *const c_char, keylen: usize) -> GoValueV2Result {
    wrap_failable(|| {
        let lua = ptr.get();
        let key = if key.is_null() {
            unsafe { std::str::from_utf8_unchecked(&[]) }
        } else {
            let key = unsafe { std::slice::from_raw_parts(key as *const u8, keylen) };
            match std::str::from_utf8(key) {
                Ok(s) => s,
                Err(_) => return GoValueV2Result::err("Invalid UTF-8 key".to_string()),
            }
        };
        match lua.named_registry_value::<mluau::Value>(key) {
            Ok(v) => GoValueV2Result::ok(GoLuaValueV2::from_value(&lua, v)),
            Err(err) => GoValueV2Result::err(format!("{err}")),
        }
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn freeluavm(ptr: VmHandle) {
    wrap_failable(|| {
        ptr.remove();
    })
}