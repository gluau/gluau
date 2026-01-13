use std::ffi::{c_char, c_void, CString};

use mluau::Lua;

use crate::{compiler::CompilerOpts, multivalue::GoMultiValue, result::{wrap_failable, GoNoneResult, GoValueResult}, value::GoLuaValue, IGoCallback, IGoCallbackWrapper};

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
pub extern "C" fn newluavm(stdlib: u32) -> *mut mluau::Lua {
    wrap_failable(|| {
        let stdlib = StdLib::from_bits_truncate(stdlib);
        let lua = Lua::new_with(
            stdlib.to_mluau(), // TODO: Allow configuring this
            mluau::LuaOptions::new()
            .catch_rust_panics(false)
            .disable_error_userdata(true)
        ).unwrap(); // Will never error, as we are using safe libraries only.

        let wrapper = Box::new(lua);
        Box::into_raw(wrapper)
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn luavm_setcompileropts(ptr: *mut mluau::Lua, opts: CompilerOpts) {
    wrap_failable(|| {
        if ptr.is_null() {
            return; // no-op if pointer is null
        }
        let lua = unsafe { &*ptr };
        lua.set_compiler(opts.to_compiler());
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn luavm_setmemorylimit(ptr: *mut mluau::Lua, limit: usize) -> GoNoneResult {
    wrap_failable(|| {
        if ptr.is_null() {
            return GoNoneResult::err("Lua pointer is null".to_string());
        }
        let lua = unsafe { &*ptr };
        match lua.set_memory_limit(limit) {
            Ok(_) => GoNoneResult::ok(),
            Err(err) => GoNoneResult::err(format!("{err}")),
        }
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn luavm_sandbox(ptr: *mut mluau::Lua, enabled: bool) -> GoNoneResult {
    wrap_failable(|| {
        // Safety: Assume the Lua VM is valid and we can set its sandbox mode.
        if ptr.is_null() {
            return GoNoneResult::err("Lua pointer is null".to_string());
        }
        let lua = unsafe { &*ptr };
        match lua.sandbox(enabled) {
            Ok(_) => GoNoneResult::ok(),
            Err(err) => GoNoneResult::err(format!("{err}")),
        }
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_globals(ptr: *mut mluau::Lua) -> *mut mluau::Table {
    wrap_failable(|| {
        // Safety: Assume ptr is a valid, non-null pointer to a Lua VM
        if ptr.is_null() {
            return std::ptr::null_mut();
        }
        let lua = unsafe { &*ptr };
        Box::into_raw(Box::new(lua.globals()))
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_setglobals(ptr: *mut mluau::Lua, tab: *mut mluau::Table) -> GoNoneResult {
    wrap_failable(|| {
        // Safety: Assume ptr is a valid, non-null pointer to a Lua VM
        if ptr.is_null() {
            return GoNoneResult::err("Lua pointer is null".to_string());
        }
        let lua = unsafe { &*ptr };
        let tab = unsafe { &*tab };
        match lua.set_globals(tab.clone()) {
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
    pub lua: *mut mluau::Lua,

    // Go side may set this to set a response
    pub vm_state: u8,
    pub error: *mut c_char,
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_set_interrupt(ptr: *mut mluau::Lua, cb: IGoCallback)  {
    wrap_failable(|| {
        // Safety: Assume ptr is a valid, non-null pointer to a mluau::Lua
        if ptr.is_null() {
            return;
        }

        let cb_wrapper = IGoCallbackWrapper::new(cb);

        let lua = unsafe { &*ptr };
        lua.set_interrupt(move |lua| {
            let wrapper = Box::new(lua.clone());
            let lua_ptr = Box::into_raw(wrapper);
            
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
pub extern "C" fn luago_remove_interrupt(ptr: *mut mluau::Lua)  {
    wrap_failable(|| {
        // Safety: Assume ptr is a valid, non-null pointer to a Lua
        if ptr.is_null() {
            return;
        }

        let lua = unsafe { &*ptr };
        lua.remove_interrupt();
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_current_thread(ptr: *mut mluau::Lua) -> *mut mluau::Thread {
    wrap_failable(|| {
        // Safety: Assume ptr is a valid, non-null pointer to a Lua
        if ptr.is_null() {
            return std::ptr::null_mut();
        }

        let lua = unsafe { &*ptr };
        Box::into_raw(Box::new(lua.current_thread()))
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_yield_with(ptr: *mut mluau::Lua, args: *mut GoMultiValue) -> GoNoneResult {
    wrap_failable(|| {
        // Safety: Assume ptr is a valid, non-null pointer to a Lua
        if ptr.is_null() {
            return GoNoneResult::err("Lua pointer is null".to_string());
        }

        let lua = unsafe { &*ptr };
        // Safety: Go side must ensure values cannot be used after it is set
        // here as a return value
        let values = unsafe { Box::from_raw(args) };
        let values_mv = values.values.into_inner().unwrap();

        match lua.yield_with(values_mv) {
            Ok(_) => GoNoneResult::ok(),
            Err(err) => GoNoneResult::err(format!("{err}")),
        }
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_used_memory(ptr: *mut mluau::Lua) -> usize {
    wrap_failable(|| {
        // Safety: Assume ptr is a valid, non-null pointer to a Lua
        if ptr.is_null() {
            return 0;
        }

        let lua = unsafe { &*ptr };
        lua.used_memory()
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_memory_limit(ptr: *mut mluau::Lua) -> usize {
    wrap_failable(|| {
        // Safety: Assume ptr is a valid, non-null pointer to a Lua
        if ptr.is_null() {
            return 0;
        }

        let lua = unsafe { &*ptr };
        lua.memory_limit().unwrap_or(0)
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_set_type_metatable(ptr: *mut mluau::Lua, typ: u8, tab: *mut mluau::Table) {
    wrap_failable(|| {
        // ptr must be non-null however tab may be null.
        if ptr.is_null() {
            return;
        }

        let lua = unsafe { &*ptr };
        let tab = if tab.is_null() {
            None
        } else {
            let tab = unsafe { &*tab };
            Some(tab.clone())
        };
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
pub extern "C" fn luago_set_named_registry_value(ptr: *mut mluau::Lua, key: *const c_char, keylen: usize, value: GoLuaValue) -> GoNoneResult {
    wrap_failable(|| {
        if ptr.is_null() {
            return GoNoneResult::err("Lua pointer is null".to_string());
        }

        let lua = unsafe { &*ptr };
        let value = value.to_value_from_owned();
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
pub extern "C" fn luago_named_registry_value(ptr: *mut mluau::Lua, key: *const c_char, keylen: usize) -> GoValueResult {
    wrap_failable(|| {
        if ptr.is_null() {
            return GoValueResult::err("Lua pointer is null".to_string());
        }

        let lua = unsafe { &*ptr };
        let key = if key.is_null() {
            unsafe { std::str::from_utf8_unchecked(&[]) }
        } else {
            let key = unsafe { std::slice::from_raw_parts(key as *const u8, keylen) };
            match std::str::from_utf8(key) {
                Ok(s) => s,
                Err(_) => return GoValueResult::err("Invalid UTF-8 key".to_string()),
            }
        };
        match lua.named_registry_value::<mluau::Value>(key) {
            Ok(v) => GoValueResult::ok(GoLuaValue::from_owned(v)),
            Err(err) => GoValueResult::err(format!("{err}")),
        }
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_set_instruction_limit(ptr: *mut mluau::Lua, limit: u64) -> GoNoneResult {
    wrap_failable(|| {
        if ptr.is_null() {
            return GoNoneResult::err("Lua pointer is null".to_string());
        }
        let lua = unsafe { &*ptr };
        let counter = std::sync::atomic::AtomicU64::new(0);

        lua.set_interrupt(move |_| {
            let count = counter.fetch_add(1, std::sync::atomic::Ordering::Relaxed);
            if count >= limit {
                return Err(mluau::Error::external("instruction limit exceeded"));
            }
            Ok(mluau::VmState::Continue)
        });
        GoNoneResult::ok()
    })
}

/// A token that can be used to cancel execution from another thread.
pub struct CancellationToken {
    cancelled: std::sync::atomic::AtomicBool,
    counter: std::sync::atomic::AtomicU64,
    limit: u64,
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_set_instruction_limit_with_cancel(
    ptr: *mut mluau::Lua,
    limit: u64,
) -> *mut CancellationToken {
    wrap_failable(|| {
        if ptr.is_null() {
            return std::ptr::null_mut();
        }
        let lua = unsafe { &*ptr };

        let token = Box::new(CancellationToken {
            cancelled: std::sync::atomic::AtomicBool::new(false),
            counter: std::sync::atomic::AtomicU64::new(0),
            limit,
        });
        let token_ptr = Box::into_raw(token);

        // Safety: token_ptr is valid for the lifetime of the interrupt
        let token_ref = unsafe { &*token_ptr };

        lua.set_interrupt(move |_| {
            if token_ref.cancelled.load(std::sync::atomic::Ordering::Relaxed) {
                return Err(mluau::Error::external("execution cancelled"));
            }
            let count = token_ref.counter.fetch_add(1, std::sync::atomic::Ordering::Relaxed);
            if count >= token_ref.limit {
                return Err(mluau::Error::external("instruction limit exceeded"));
            }
            Ok(mluau::VmState::Continue)
        });

        token_ptr
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_cancel_execution(token: *mut CancellationToken) {
    wrap_failable(|| {
        if token.is_null() {
            return;
        }
        let token = unsafe { &*token };
        token.cancelled.store(true, std::sync::atomic::Ordering::Relaxed);
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_free_cancellation_token(token: *mut CancellationToken) {
    wrap_failable(|| {
        if token.is_null() {
            return;
        }
        unsafe {
            drop(Box::from_raw(token));
        }
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn freeluavm(ptr: *mut mluau::Lua) {
    wrap_failable(|| {
        // Safety: Assume ptr is a valid, non-null pointer to a mluau::Lua
        // and that ownership is being transferred back to Rust to be dropped.
        unsafe {
            drop(Box::from_raw(ptr));
        }
    })
}