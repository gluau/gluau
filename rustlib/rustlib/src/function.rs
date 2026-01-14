use std::ffi::{c_char, c_void, CString};

use mluau::Lua;

use crate::{IGoCallback, IGoCallbackWrapper, result::{GoBoolResult, GoLuaValueV2ArrayResult, GoValueV2Result, wrap_failable}, table::get_table, value_v2::{GoLuaValueV2, GoLuaValueV2Array}};

pub fn get_function(lua: &Lua, t: GoLuaValueV2) -> Option<mluau::Function> {
    let v = t.to_value(lua);
    match v {
        mluau::Value::Function(b) => Some(b),
        _ => None,
    }
}

#[repr(C)]
// NOTE: Aside from the Lua, Rust will deallocate everything
pub struct FunctionCallbackData {
    // Lua representing the Lua State
    // as called from Lua.
    //
    // This means that (future) API's like Lua.CurrentThread will return
    // the correct thread when using this Lua.
    pub lua: *mut mluau::Lua, // Must be deallocated by Go

    // Arguments passed to the function by Lua, must be deallocated by Go
    pub args: GoLuaValueV2Array,

    // Go side may set this to set a response
    pub values: GoLuaValueV2Array,
    pub error: *mut c_char,
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_create_function(lua: *mut Lua, cb: IGoCallback) -> GoValueV2Result  {
    wrap_failable(|| {
        // Safety: Assume ptr is a valid, non-null pointer to a Lua
        if lua.is_null() {
            return GoValueV2Result::err("Lua pointer is null".to_string());
        }

        let cb_wrapper = IGoCallbackWrapper::new(cb);

        let lua = unsafe { &*lua };
        let func = lua.create_function(move |lua, args: mluau::MultiValue| {
            let wrapper = Box::new(lua.clone());
            let lua_ptr = Box::into_raw(wrapper);
            
            let mut data = FunctionCallbackData {
                lua: lua_ptr,
                args: GoLuaValueV2Array::from_values(lua, args),
                values: GoLuaValueV2Array::empty(),
                error: std::ptr::null_mut(),
            };

            cb_wrapper.callback(&mut data as *mut FunctionCallbackData as *mut c_void);
            println!("Returned from Go callback");
            //let _lua = unsafe { Box::from_raw(lua_ptr) };

            let mw = if data.values.size > 0 {
                data.values.to_values(lua)
            } else {
                mluau::MultiValue::with_capacity(0)
            };

            if !data.error.is_null() {
                // Safety: Go must not use the error after this point
                // as it is deallocated here.
                let error = unsafe { CString::from_raw(data.error) };
                return Err(mluau::Error::external(error.to_string_lossy()));
            } else {
                return Ok(mw);
            }
        });

        match func {
            Ok(f) => GoValueV2Result::ok(GoLuaValueV2::from_value(lua, mluau::Value::Function(f))),
            Err(err) => GoValueV2Result::err(format!("{err}")),
        }
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_function_call(lua: *mut Lua, ptr: GoLuaValueV2, args: GoLuaValueV2Array) -> GoLuaValueV2ArrayResult  {
    wrap_failable(|| {
        let lua = unsafe { &*lua };
        let Some(func) = get_function(lua, ptr) else {
            return GoLuaValueV2ArrayResult::err("Provided value is not a LuaFunction".to_string());
        };

        println!("Calling Lua function from Go");
        let values = args.to_values(lua);
        let res = func.call::<mluau::MultiValue>(values);
        match res {
            Ok(mv) => GoLuaValueV2ArrayResult::ok(GoLuaValueV2Array::from_values(lua,mv)),
            Err(e) => GoLuaValueV2ArrayResult::err(format!("{e}"))
        }
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_function_deepclone(lua: *mut mluau::Lua, f: GoLuaValueV2) -> GoValueV2Result {
    wrap_failable(|| {
        let lua = unsafe { &*lua };
        let Some(f) = get_function(lua, f) else {
            return GoValueV2Result::err("Provided value is not a LuaFunction".to_string());
        };

        let cloned_fn = f.deep_clone();

        match cloned_fn {
            Ok(func) => GoValueV2Result::ok(GoLuaValueV2::from_value(lua, mluau::Value::Function(func))),
            Err(e) => GoValueV2Result::err(format!("{e}")),
        }
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_function_environment(lua: *mut mluau::Lua, f: GoLuaValueV2) -> GoLuaValueV2 {
    wrap_failable(|| {
        let lua = unsafe { &*lua };
        let Some(f) = get_function(lua, f) else {
            return GoLuaValueV2::empty();
        };

        let env = f.environment();

        match env {
            Some(table) => GoLuaValueV2::from_value(lua, mluau::Value::Table(table)),
            None => GoLuaValueV2::empty(),
        }
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_function_set_environment(lua: *mut Lua, f: GoLuaValueV2, table: GoLuaValueV2) -> GoBoolResult {
    wrap_failable(|| {
        let lua = unsafe { &*lua };
        let Some(f) = get_function(lua, f) else {
            return GoBoolResult::err("Provided value is not a LuaFunction".to_string());
        };
        let Some(t) = get_table(lua, table) else {
            return GoBoolResult::err("Provided environment is not a LuaTable".to_string());
        };

        let res = f.set_environment(t);

        match res {
            Ok(res) => GoBoolResult::ok(res),
            Err(e) => GoBoolResult::err(format!("{e}")),
        }
    })
}
