use std::ffi::c_void;

use crate::{result::{GoBoolResult, GoI64Result, GoNoneResult, GoTableResult, GoValueResult}, value::GoLuaValue, IGoCallback, IGoCallbackWrapper, LuaVmWrapper};

#[unsafe(no_mangle)]
pub extern "C-unwind" fn luago_create_table(ptr: *mut LuaVmWrapper) -> GoTableResult  {
    // Safety: Assume ptr is a valid, non-null pointer to a LuaVmWrapper
    // and that s points to a valid C string of length len.
    let res = unsafe {
        let lua = &(*ptr).lua;
        lua.create_table()
    };

    match res {
        Ok(r) => GoTableResult::ok(Box::into_raw(Box::new(r))),
        Err(err) => GoTableResult::err(err),
    }
}

#[unsafe(no_mangle)]
pub extern "C-unwind" fn luago_create_table_with_capacity(ptr: *mut LuaVmWrapper, narr: usize, nrec: usize) -> GoTableResult  {
    // Safety: Assume ptr is a valid, non-null pointer to a LuaVmWrapper
    // and that s points to a valid C string of length len.
    let res = unsafe {
        let lua = &(*ptr).lua;
        lua.create_table_with_capacity(narr, nrec)
    };

    match res {
        Ok(r) => GoTableResult::ok(Box::into_raw(Box::new(r))),
        Err(err) => GoTableResult::err(err),
    }
}

#[unsafe(no_mangle)]
pub extern "C-unwind" fn luago_table_clear(tab: *mut mluau::Table) -> GoNoneResult {
    // Safety: Assume table is a valid, non-null pointer to a Lua Table
    if tab.is_null() {
        return GoNoneResult::err(mluau::Error::external("Table pointer is null".to_string()));
    }

    let tab = unsafe { &*tab };

    let res = tab.clear();

    match res {
        Ok(_) => GoNoneResult::ok(),
        Err(err) => GoNoneResult::err(err),
    }
}

#[unsafe(no_mangle)]
pub extern "C-unwind" fn luago_table_contains_key(tab: *mut mluau::Table, value: GoLuaValue) -> GoBoolResult {
    // Safety: Assume table is a valid, non-null pointer to a Lua Table
    if tab.is_null() {
        return GoBoolResult::err(mluau::Error::external("Table pointer is null".to_string()));
    }

    let tab = unsafe { &*tab };
    let value = value.to_value_from_owned();

    let res = tab.contains_key(value);

    match res {
        Ok(r) => GoBoolResult::ok(r),
        Err(err) => GoBoolResult::err(err),
    }
}

#[unsafe(no_mangle)]
pub extern "C-unwind" fn luago_table_equals(tab: *mut mluau::Table, tab2: *mut mluau::Table) -> GoBoolResult {
    // Safety: Assume table is a valid, non-null pointer to a Lua Table
    if tab.is_null() || tab2.is_null() {
        return GoBoolResult::err(mluau::Error::external("Table pointer is null".to_string()));
    }

    let tab = unsafe { &*tab };
    let tab2 = unsafe { &*tab2 };

    let res = tab.equals(tab2);

    match res {
        Ok(r) => GoBoolResult::ok(r),
        Err(err) => GoBoolResult::err(err),
    }
}

#[repr(C)]
pub struct TableForEachCallbackData {
    pub key: GoLuaValue,
    pub value: GoLuaValue,

    // Go code may modify the below
    pub stop: bool,
}

#[unsafe(no_mangle)]
pub extern "C-unwind" fn luago_table_foreach(tab: *mut mluau::Table, cb: IGoCallback) -> GoNoneResult {
    // Safety: Assume table is a valid, non-null pointer to a Lua Table
    if tab.is_null() {
        return GoNoneResult::err(mluau::Error::external("Table pointer is null".to_string()));
    }

    let tab = unsafe { &*tab };
    let cb_wrapper = IGoCallbackWrapper::new(cb);

    let res = tab.for_each(|key: mluau::Value, value: mluau::Value| {
        let data = TableForEachCallbackData {
            key: GoLuaValue::from_owned(key),
            value: GoLuaValue::from_owned(value),
            stop: false,
        };
        // TODO: Avoid the pointer allocation if possible
        let ptr = Box::into_raw(Box::new(data));
        cb_wrapper.callback(ptr as *mut c_void);
        let data = unsafe { Box::from_raw(ptr) };

        if data.stop {
            // Use a dummy error variant to stop the iteration
            return Err(mluau::Error::BindError);
        }

        Ok(())
    });

    match res {
        Ok(_) => GoNoneResult::ok(),
        Err(err) => GoNoneResult::err(err),
    }
}

#[unsafe(no_mangle)]
pub extern "C-unwind" fn luago_table_get(tab: *mut mluau::Table, key: GoLuaValue) -> GoValueResult {
    // Safety: Assume table is a valid, non-null pointer to a Lua Table
    if tab.is_null() {
        return GoValueResult::err(mluau::Error::external("Table pointer is null".to_string()));
    }

    let tab = unsafe { &*tab };
    let key = key.to_value_from_owned();
    let res = tab.get::<mluau::Value>(key);
    
    match res {
        Ok(r) => GoValueResult::ok(GoLuaValue::from_owned(r)),
        Err(err) => GoValueResult::err(err),
    }
}

#[unsafe(no_mangle)]
pub extern "C-unwind" fn luago_table_is_empty(tab: *mut mluau::Table) -> bool {
    // Safety: Assume table is a valid, non-null pointer to a Lua Table
    if tab.is_null() {
        return true; // If the table pointer is null, consider it empty
    }

    let tab = unsafe { &*tab };
    tab.is_empty()
}

#[unsafe(no_mangle)]
pub extern "C-unwind" fn luago_table_is_readonly(tab: *mut mluau::Table) -> bool {
    // Safety: Assume table is a valid, non-null pointer to a Lua Table
    if tab.is_null() {
        return true; // If the table pointer is null, consider it readonly
    }

    let tab = unsafe { &*tab };
    tab.is_readonly()
}

#[unsafe(no_mangle)]
pub extern "C-unwind" fn luago_table_len(tab: *mut mluau::Table) -> GoI64Result {
    // Safety: Assume table is a valid, non-null pointer to a Lua Table
    if tab.is_null() {
        return GoI64Result::err(mluau::Error::external("Table pointer is null".to_string()));
    }

    let tab = unsafe { &*tab };
    match tab.len() {
        Ok(len) => GoI64Result::ok(len),
        Err(err) => GoI64Result::err(err),
    }
}

#[unsafe(no_mangle)]
pub extern "C-unwind" fn luago_table_metatable(tab: *mut mluau::Table) -> *mut mluau::Table {
    // Safety: Assume table is a valid, non-null pointer to a Lua Table
    if tab.is_null() {
        return std::ptr::null_mut(); // If the table pointer is null, return null
    }

    let tab = unsafe { &*tab };
    match tab.metatable() {
        Some(mt) => Box::into_raw(Box::new(mt)),
        None => std::ptr::null_mut(), // If no metatable, return null
    }
}

#[unsafe(no_mangle)]
pub extern "C-unwind" fn luago_table_pop(tab: *mut mluau::Table) -> GoValueResult {
    // Safety: Assume table is a valid, non-null pointer to a Lua Table
    if tab.is_null() {
        return GoValueResult::err(mluau::Error::external("Table pointer is null".to_string()));
    }

    let tab = unsafe { &*tab };
    match tab.pop::<mluau::Value>() {
        Ok(v) => GoValueResult::ok(GoLuaValue::from_owned(v)),
        Err(err) => GoValueResult::err(err),
    }
}

#[unsafe(no_mangle)]
pub extern "C-unwind" fn luago_table_push(tab: *mut mluau::Table, value: GoLuaValue) -> GoNoneResult {
    // Safety: Assume table is a valid, non-null pointer to a Lua Table
    if tab.is_null() {
        return GoNoneResult::err(mluau::Error::external("Table pointer is null".to_string()));
    }

    let tab = unsafe { &*tab };
    match tab.push(value.to_value_from_owned()) {
        Ok(_) => GoNoneResult::ok(),
        Err(err) => GoNoneResult::err(err),
    }
}

#[unsafe(no_mangle)]
pub extern "C-unwind" fn luago_free_table(tab: *mut mluau::Table) {
    // Safety: Assume table is a valid, non-null pointer to a Lua Table
    if tab.is_null() {
        return;
    }

    // Re-box the Lua Table pointer to manage its memory automatically.
    unsafe { drop(Box::from_raw(tab)) };
}