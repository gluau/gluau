use mluau::Lua;

use crate::{function::get_function, result::{GoLuaValueV2ArrayResult, GoNoneResult, GoValueV2Result, wrap_failable}, value_v2::{GoLuaValueV2, GoLuaValueV2Array}};

pub fn get_thread(lua: &Lua, t: GoLuaValueV2) -> Option<mluau::Thread> {
    let v = t.to_value(lua);
    match v {
        mluau::Value::Thread(b) => Some(b),
        _ => None,
    }
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_create_thread(lua: *mut mluau::Lua, func: GoLuaValueV2) -> GoValueV2Result {
    wrap_failable(|| {
        if lua.is_null() {
            return GoValueV2Result::err("Lua pointer or function pointer is null".to_string());
        }

        let lua = unsafe { &*lua };
        let Some(func) = get_function(lua, func) else {
            return GoValueV2Result::err("Provided value is not a function".to_string());
        };

        match lua.create_thread(func) {
            Ok(thread) => GoValueV2Result::ok(GoLuaValueV2::from_value(lua, mluau::Value::Thread(thread))),
            Err(e) => GoValueV2Result::err(format!("{e}")),
        }
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_reset_thread(lua: *mut mluau::Lua, th: GoLuaValueV2, func: GoLuaValueV2) -> GoNoneResult {
    wrap_failable(|| {
        let lua = unsafe { &*lua };
        let Some(th) = get_thread(lua, th) else {
            return GoNoneResult::err("Provided thread value is not a LuaThread".to_string());
        };
        let Some(func) = get_function(lua, func) else {
            return GoNoneResult::err("Provided function value is not a LuaFunction".to_string());
        };

        match th.reset(func) {
            Ok(_) => GoNoneResult::ok(),
            Err(e) => GoNoneResult::err(format!("{e}")),
        }
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_thread_status(lua: *mut mluau::Lua, th: GoLuaValueV2) -> u8 {
    wrap_failable(|| {
        let lua = unsafe { &*lua };
        let Some(t) = get_thread(lua, th) else {
            return 2; // Consider non-threads as finished
        };

        // Get the status of the Lua thread
        match t.status() {
            mluau::ThreadStatus::Resumable => 0,
            mluau::ThreadStatus::Running => 1,
            mluau::ThreadStatus::Finished => 2,
            mluau::ThreadStatus::Error => 3,
        }
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_thread_sandbox(lua: *mut mluau::Lua, t: GoLuaValueV2) -> GoNoneResult {
    wrap_failable(|| {
        let lua = unsafe { &*lua };
        let Some(t) = get_thread(lua, t) else {
            return GoNoneResult::err("Provided thread value is not a LuaThread".to_string());
        };

        // Get the status of the Lua thread
        match t.sandbox() {
            Ok(_) => GoNoneResult::ok(),
            Err(e) => GoNoneResult::err(format!("{e}")),
        }
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_thread_resume(lua: *mut mluau::Lua, th: GoLuaValueV2, args: GoLuaValueV2Array) -> GoLuaValueV2ArrayResult {
    wrap_failable(|| {
        let lua = unsafe { &*lua };
        let Some(th) = get_thread(lua, th) else {
            return GoLuaValueV2ArrayResult::err("Function pointer is null".to_string());
        };
        let mw = args.to_values(lua);
        
        let res = th.resume::<mluau::MultiValue>(mw);
        match res {
            Ok(mv) => GoLuaValueV2ArrayResult::ok(GoLuaValueV2Array::from_values(lua, mv)),
            Err(e) => GoLuaValueV2ArrayResult::err(format!("{e}"))
        }
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_thread_resume_error(lua: *mut mluau::Lua, th: GoLuaValueV2, error: GoLuaValueV2) -> GoLuaValueV2ArrayResult  {
    wrap_failable(|| {
        let lua = unsafe { &*lua };
        let Some(th) = get_thread(lua, th) else {
            return GoLuaValueV2ArrayResult::err("Function pointer is null".to_string());
        };
        let error = error.to_value(lua);
        let res = th.resume_error::<mluau::MultiValue>(error);
        match res {
            Ok(mv) => GoLuaValueV2ArrayResult::ok(GoLuaValueV2Array::from_values(lua, mv)),
            Err(e) => GoLuaValueV2ArrayResult::err(format!("{e}"))
        }
    })
}
