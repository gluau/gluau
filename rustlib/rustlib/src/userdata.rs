use mluau::Lua;
use crate::{result::{GoUsizePtrResult, GoValueV2Result, wrap_failable}, table::get_table, value_v2::GoLuaValueV2};

pub fn get_userdata(lua: &Lua, t: GoLuaValueV2) -> Option<mluau::AnyUserData> {
    let v = t.to_value(lua);
    match v {
        mluau::Value::UserData(b) => Some(b),
        _ => None,
    }
}

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
pub extern "C" fn luago_create_userdata(lua: *mut mluau::Lua, data: DynamicData, mt: GoLuaValueV2) -> GoValueV2Result {
    wrap_failable(|| {
        // Safety: Create a new userdata with the provided data and metatable.
        let lua = unsafe { &*lua };
        let Some(mt) = get_table(lua, mt) else {
            return GoValueV2Result::err("Metatable is not a table".to_string());
        };

        let res = lua.create_dynamic_userdata(data, &mt);

        match res {
            Ok(userdata) => GoValueV2Result::ok(GoLuaValueV2::from_value(lua, mluau::Value::UserData(userdata))),
            Err(e) => GoValueV2Result::err(e.to_string()),
        }
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_get_userdata_handle(lua: *mut mluau::Lua, ud: GoLuaValueV2) -> GoUsizePtrResult {
    wrap_failable(|| {
        // Safety: Assume userdata is a valid, non-null pointer to a Lua Userdata
        let lua = unsafe { &*lua };
        let Some(ud) = get_userdata(lua, ud) else {
            return GoUsizePtrResult::err("Value is not a LuaUserData".to_string());
        };

        match ud.dynamic_data::<DynamicData>() {
            Ok(data) => GoUsizePtrResult::ok(data.handle),
            Err(e) => GoUsizePtrResult::err(e.to_string()),
        }
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_userdata_metatable(lua: *mut mluau::Lua, userdata: GoLuaValueV2) -> GoValueV2Result {
    wrap_failable(|| {
        let lua = unsafe { &*lua };
        let Some(userdata) = get_userdata(lua, userdata) else {
            return GoValueV2Result::err("Value is not a LuaUserData".to_string());
        };

        // SAFETY: Luau does not have __gc metamethod, so this is safe to call here.
        let mt = unsafe { userdata.underlying_metatable() };

        match mt {
            Ok(mt) => GoValueV2Result::ok(GoLuaValueV2::from_value(lua, mluau::Value::Table(mt))),
            Err(e) => GoValueV2Result::err(e.to_string()),
        }
    })
}
