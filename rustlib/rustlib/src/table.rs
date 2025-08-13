use crate::{result::GoResult, LuaVmWrapper};

#[unsafe(no_mangle)]
pub extern "C" fn luago_create_table(ptr: *mut LuaVmWrapper) -> GoResult  {
    // Safety: Assume ptr is a valid, non-null pointer to a LuaVmWrapper
    // and that s points to a valid C string of length len.
    let res = unsafe {
        let lua = &mut (*ptr).lua;
        lua.create_table()
    };

    GoResult::new(res)
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_create_table_with_capacity(ptr: *mut LuaVmWrapper, narr: usize, nrec: usize) -> GoResult  {
    // Safety: Assume ptr is a valid, non-null pointer to a LuaVmWrapper
    // and that s points to a valid C string of length len.
    let res = unsafe {
        let lua = &mut (*ptr).lua;
        lua.create_table_with_capacity(narr, nrec)
    };

    GoResult::new(res)
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_table_clear(tab: *mut mluau::Table) {
    // TODO: Implement table clear
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_free_table(tab: *mut mluau::Table) {
    // Safety: Assume table is a valid, non-null pointer to a Lua Table
    if tab.is_null() {
        return;
    }

    // Re-box the Lua Table pointer to manage its memory automatically.
    unsafe { drop(Box::from_raw(tab)) };
}