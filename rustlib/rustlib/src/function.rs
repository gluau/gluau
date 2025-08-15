use std::ffi::c_void;

use crate::{multivalue::GoMultiValue, result::{GoFunctionResult, GoMultiValueResult}, value::ErrorVariant, IGoCallback, IGoCallbackWrapper, LuaVmWrapper};

#[repr(C)]
// NOTE: Aside from the LuaVmWrapper, Rust will deallocate everything
pub struct FunctionCallbackData {
    // LuaVmWrapper representing the Lua State
    // as called from Lua.
    //
    // This means that (future) API's like LuaVmWrapper.CurrentThread will return
    // the correct thread when using this LuaVmWrapper.
    pub lua: *mut LuaVmWrapper,
    // Arguments passed to the function by Lua
    pub args: *mut GoMultiValue,

    // Go side may set this to set a response
    pub values: *mut GoMultiValue,
    pub error: *mut ErrorVariant,
}

#[unsafe(no_mangle)]
pub extern "C-unwind" fn luago_create_function(ptr: *mut LuaVmWrapper, cb: IGoCallback) -> GoFunctionResult  {
    // Safety: Assume ptr is a valid, non-null pointer to a LuaVmWrapper
    if ptr.is_null() {
        return GoFunctionResult::err("LuaVmWrapper pointer is null".to_string());
    }

    let cb_wrapper = IGoCallbackWrapper::new(cb);

    let lua = unsafe { &(*ptr).lua };
    let func = lua.create_function(move |lua, args: mluau::MultiValue| {
        let wrapper = Box::new(LuaVmWrapper { lua: lua.clone() });
        let lua_ptr = Box::into_raw(wrapper);
        
        let data = FunctionCallbackData {
            lua: lua_ptr,
            args: GoMultiValue::inst(args),
            values: std::ptr::null_mut(),
            error: std::ptr::null_mut(),
        };

        let ptr = Box::into_raw(Box::new(data));
        cb_wrapper.callback(ptr as *mut c_void);
        let data = unsafe { Box::from_raw(ptr) };
        unsafe { drop(Box::from_raw(data.args)) }
        
        if !data.error.is_null() {
            if !data.values.is_null() {
                // Avoid a memory leak by deallocating it
                unsafe { drop(Box::from_raw(data.values)) };
            }

            let error = unsafe { Box::from_raw(data.error) };
            return Err(mluau::Error::external(error.error.to_string_lossy()));
        } else {
            // If values is set, return them to Lua.
            if !data.values.is_null() {
                // Safety: Go side must ensure values cannot be used after it is set
                // here as a return value
                let values = unsafe { Box::from_raw(data.values) };
                let values_mv = values.values.into_inner().unwrap();
                return Ok(values_mv);
            } else {
                // If no values are set, return an empty MultiValue.
                return Ok(mluau::MultiValue::new());
            }
        }
    });

    match func {
        Ok(f) => GoFunctionResult::ok(Box::into_raw(Box::new(f))),
        Err(err) => GoFunctionResult::err(format!("{err}")),
    }
}

#[unsafe(no_mangle)]
pub extern "C-unwind" fn luago_function_call(ptr: *mut mluau::Function, args: *mut GoMultiValue) -> GoMultiValueResult  {
    if ptr.is_null() {
        return GoMultiValueResult::err("Function pointer is null".to_string());
    }

    let func = unsafe { &*ptr };
    
    // Safety: Go side must ensure values cannot be used after it is set
    // here as a return value
    let values = unsafe { Box::from_raw(args) };
    let values_mv = values.values.into_inner().unwrap();
    let res = func.call::<mluau::MultiValue>(values_mv);
    match res {
        Ok(mv) => GoMultiValueResult::ok(GoMultiValue::inst(mv)),
        Err(e) => GoMultiValueResult::err(format!("{e}"))
    }
}

#[unsafe(no_mangle)]
pub extern "C-unwind" fn luago_free_function(f: *mut mluau::Function) {
    // Safety: Assume function is a valid, non-null pointer to a Lua function
    if f.is_null() {
        return;
    }

    // Re-box the Lua function pointer to manage its memory automatically.
    unsafe { drop(Box::from_raw(f)) };
}