use crate::{result::{GoAnyUserDataResult, GoUsizePtrResult}, LuaVmWrapper};

/// DynamicData stores the cgo handle + callback for dynamic userdata functions.
#[repr(C)]
pub struct DynamicData {
    handle: usize, 
    drop: extern "C" fn(handle: usize),
}

impl Drop for DynamicData {
    fn drop(&mut self) {
        // Ensure the drop function is called only if the handle is not null.
        // This prevents double freeing or calling drop on an invalid handle.
        if self.handle != 0 {
            (self.drop)(self.handle);
        }
    }
} 

#[unsafe(no_mangle)]
pub extern "C-unwind" fn luago_create_userdata(ptr: *mut LuaVmWrapper, data: DynamicData, mt: *mut mluau::Table) -> GoAnyUserDataResult {
    // Safety: Create a new userdata with the provided data and metatable.
    if ptr.is_null() {
        return GoAnyUserDataResult::err("LuaVmWrapper pointer is null".to_string());
    }
    if mt.is_null() {
        return GoAnyUserDataResult::err("Metatable pointer is null".to_string());
    }
    let lua = unsafe { &(*ptr).lua };
    let mt = unsafe { &*mt };

    let res = lua.create_dynamic_userdata(data, mt);

    match res {
        Ok(userdata) => GoAnyUserDataResult::ok(Box::into_raw(Box::new(userdata))),
        Err(e) => GoAnyUserDataResult::err(e.to_string()),
    }
}

#[unsafe(no_mangle)]
pub extern "C-unwind" fn luago_get_userdata_handle(ud: *mut mluau::AnyUserData) -> GoUsizePtrResult {
    // Safety: Assume userdata is a valid, non-null pointer to a Lua Userdata
    if ud.is_null() {
        return GoUsizePtrResult::err("LuaUserData pointer is null".to_string());
    }

    let userdata = unsafe { &*ud };
    match userdata.dynamic_data::<DynamicData>() {
        Ok(data) => GoUsizePtrResult::ok(data.handle),
        Err(e) => GoUsizePtrResult::err(e.to_string()),
    }
}

#[unsafe(no_mangle)]
pub extern "C-unwind" fn luago_free_userdata(ud: *mut mluau::AnyUserData) {
    // Safety: Assume userdata is a valid, non-null pointer to a Lua Table
    if ud.is_null() {
        return;
    }

    // Re-box the Lua Userdata pointer to manage its memory automatically.
    unsafe { drop(Box::from_raw(ud)) };
}