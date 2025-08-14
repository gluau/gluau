use crate::{result::GoResult, value::GoLuaValue, LuaVmWrapper};

#[unsafe(no_mangle)]
pub extern "C-unwind" fn luago_create_table(ptr: *mut LuaVmWrapper) -> GoResult  {
    // Safety: Assume ptr is a valid, non-null pointer to a LuaVmWrapper
    // and that s points to a valid C string of length len.
    let res = unsafe {
        let lua = &(*ptr).lua;
        lua.create_table()
    };

    GoResult::new(res)
}

#[unsafe(no_mangle)]
pub extern "C-unwind" fn luago_create_table_with_capacity(ptr: *mut LuaVmWrapper, narr: usize, nrec: usize) -> GoResult  {
    // Safety: Assume ptr is a valid, non-null pointer to a LuaVmWrapper
    // and that s points to a valid C string of length len.
    let res = unsafe {
        let lua = &(*ptr).lua;
        lua.create_table_with_capacity(narr, nrec)
    };

    GoResult::new(res)
}

#[unsafe(no_mangle)]
pub extern "C-unwind" fn luago_table_clear(tab: *mut mluau::Table) -> GoResult {
    // Safety: Assume table is a valid, non-null pointer to a Lua Table
    if tab.is_null() {
        return GoResult::from_error(mluau::Error::external("Table pointer is null".to_string()));
    }

    let tab = unsafe { &*tab };

    let res = tab.clear();

    match res {
        Ok(_) => GoResult::from_value(true),
        Err(err) => GoResult::from_error(err),
    }
}

#[unsafe(no_mangle)]
pub extern "C-unwind" fn luago_table_contains_key(tab: *mut mluau::Table, value: GoLuaValue) -> GoResult {
    // Safety: Assume table is a valid, non-null pointer to a Lua Table
    if tab.is_null() {
        return GoResult::from_error(mluau::Error::external("Table pointer is null".to_string()));
    }

    let tab = unsafe { &*tab };
    let value = value.to_value_from_owned();

    let res = tab.contains_key(value);

    GoResult::new(res)
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