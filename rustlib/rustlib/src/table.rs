use std::ffi::{c_char, c_void, CString};

use crate::{IGoCallback, IGoCallbackWrapper, VmHandle, result::{GoBoolResult, GoI64Result, GoNoneResult, GoValueV2Result, wrap_failable}, value_v2::GoLuaValueV2};

pub fn get_table(lua: &mluau::Lua, t: GoLuaValueV2) -> Option<mluau::Table> {
    let v = t.to_value(&lua);
    match v {
        mluau::Value::Table(b) => Some(b),
        _ => None,
    }
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_create_table(lua: VmHandle) -> GoValueV2Result  {
    wrap_failable(|| {
        // Safety: Assume ptr is a valid, non-null pointer to a Lua
        let lua = lua.get();
        let res = lua.create_table();

        match res {
            Ok(r) => GoValueV2Result::ok(GoLuaValueV2::from_value(&lua, mluau::Value::Table(r))),
            Err(err) => GoValueV2Result::err(format!("{err}")),
        }
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_create_table_with_capacity(lua: VmHandle, narr: usize, nrec: usize) -> GoValueV2Result {
    wrap_failable(|| {
        // Safety: Assume ptr is a valid, non-null pointer to a Lua


        let lua = lua.get();
        let res = lua.create_table_with_capacity(narr, nrec);

        match res {
            Ok(r) => GoValueV2Result::ok(GoLuaValueV2::from_value(&lua, mluau::Value::Table(r))),
            Err(err) => GoValueV2Result::err(format!("{err}")),
        }
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_table_clear(lua: VmHandle, tab: GoLuaValueV2) -> GoNoneResult {
    wrap_failable(|| {
        // Safety: Assume table is a valid, non-null pointer to a Lua Table
        let lua = lua.get();
        let Some(tab) = get_table(&lua, tab) else {
            return GoNoneResult::err("Table pointer is null".to_string());
        };

        let res = tab.clear();

        match res {
            Ok(_) => GoNoneResult::ok(),
            Err(err) => GoNoneResult::err(format!("{err}")),
        }
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_table_contains_key(lua: VmHandle, tab: GoLuaValueV2, value: GoLuaValueV2) -> GoBoolResult {
    wrap_failable(|| {
        let lua = lua.get();
        let Some(tab) = get_table(&lua, tab) else {
            return GoBoolResult::err("Table pointer is null".to_string());
        };

        let value = value.to_value(&lua);

        let res = tab.contains_key(value);

        match res {
            Ok(r) => GoBoolResult::ok(r),
            Err(err) => GoBoolResult::err(format!("{err}")),
        }
    })
}

#[repr(C)]
pub struct TableForEachCallbackData {
    pub key: GoLuaValueV2,
    pub value: GoLuaValueV2,

    // Go code may modify the below
    pub stop: bool,
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_table_foreach(lua: VmHandle, tab: GoLuaValueV2, cb: IGoCallback) -> GoNoneResult {
    wrap_failable(|| {
        let lua = lua.get();
        let Some(tab) = get_table(&lua, tab) else {
            return GoNoneResult::err("Table pointer is null".to_string());
        };

        let cb_wrapper = IGoCallbackWrapper::new(cb);

        let mut stopped = false;
        let res = tab.for_each(|key: mluau::Value, value: mluau::Value| {
            let Some(lua) = tab.weak_lua().try_upgrade() else {
                return Err(mluau::Error::external("Table has no associated Lua state"));
            };
            let mut data = TableForEachCallbackData {
                key: GoLuaValueV2::from_value(&lua, key),
                value: GoLuaValueV2::from_value(&lua, value),
                stop: false,
            };

            cb_wrapper.callback(&mut data as *const TableForEachCallbackData as *mut c_void);

            if data.stop {
                // Use a dummy error variant to stop the iteration
                stopped = true; 
                return Err(mluau::Error::external(""));
            }

            Ok(())
        });

        match res {
            Ok(_) => GoNoneResult::ok(),
            Err(err) => {
                if stopped {
                    return GoNoneResult::ok(); // If stopped, return ok
                }
                GoNoneResult::err(format!("{err}"))
            },
        }
    })
}

#[repr(C)]
pub struct TableForEachValueCallbackData {
    pub value: GoLuaValueV2,

    // Go code may modify the below
    pub stop: bool,
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_table_foreach_value(lua: VmHandle, tab: GoLuaValueV2, cb: IGoCallback) -> GoNoneResult {
    wrap_failable(|| {
        let lua = lua.get();
        let Some(tab) = get_table(&lua, tab) else {
            return GoNoneResult::err("Table pointer is null".to_string());
        };

        let cb_wrapper = IGoCallbackWrapper::new(cb);

        let res = tab.for_each_value(|value: mluau::Value| {
            let Some(lua) = tab.weak_lua().try_upgrade() else {
                return Err(mluau::Error::external("Table has no associated Lua state"));
            };
            let mut data = TableForEachValueCallbackData {
                value: GoLuaValueV2::from_value(&lua, value),
                stop: false,
            };
            cb_wrapper.callback(&mut data as *const TableForEachValueCallbackData as *mut c_void);

            if data.stop {
                // Use a dummy error variant to stop the iteration
                return Err(mluau::Error::external("stop"));
            }

            Ok(())
        });

        match res {
            Ok(_) => GoNoneResult::ok(),
            Err(err) => GoNoneResult::err(format!("{err}")),
        }
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_table_get(lua: VmHandle, tab: GoLuaValueV2, key: GoLuaValueV2) -> GoValueV2Result {
    wrap_failable(|| {
        let lua = lua.get();
        let Some(tab) = get_table(&lua, tab) else {
            return GoValueV2Result::err("Table pointer is null".to_string());
        };

        let key = key.to_value(&lua);
        let res = tab.get::<mluau::Value>(key);
        
        match res {
            Ok(r) => GoValueV2Result::ok(GoLuaValueV2::from_value(&lua, r)),
            Err(err) => GoValueV2Result::err(format!("{err}")),
        }
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_table_is_empty(lua: VmHandle, tab: GoLuaValueV2) -> bool {
    wrap_failable(|| {
        let lua = lua.get();
        let Some(tab) = get_table(&lua, tab) else {
            return true; // If the table pointer is null, consider it empty
        };
        tab.is_empty()
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_table_is_readonly(lua: VmHandle, tab: GoLuaValueV2) -> bool {
    wrap_failable(|| {
        let lua = lua.get();
        let Some(tab) = get_table(&lua, tab) else {
            return false; // If the table pointer is null, consider it not readonly
        };
        tab.is_readonly()
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_table_len(lua: VmHandle, tab: GoLuaValueV2) -> GoI64Result {
    wrap_failable(|| {
        let lua = lua.get();
        let Some(tab) = get_table(&lua, tab) else {
            return GoI64Result::err("Table pointer is null".to_string());
        };
        match tab.len() {
            Ok(len) => GoI64Result::ok(len),
            Err(err) => GoI64Result::err(format!("{err}")),
        }
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_table_metatable(lua: VmHandle, tab: GoLuaValueV2) -> GoLuaValueV2 {
    wrap_failable(|| {
        let lua = lua.get();
        let Some(tab) = get_table(&lua, tab) else {
            return GoLuaValueV2::empty();
        };
        match tab.metatable() {
            Some(mt) => GoLuaValueV2::from_value(&lua, mluau::Value::Table(mt)),
            None => GoLuaValueV2::empty()
        }
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_table_pop(lua: VmHandle, tab: GoLuaValueV2) -> GoValueV2Result {
    wrap_failable(|| {
        let lua = lua.get();
        let Some(tab) = get_table(&lua, tab) else {
            return GoValueV2Result::err("Table pointer is null".to_string());
        };

        match tab.pop::<mluau::Value>() {
            Ok(v) => GoValueV2Result::ok(GoLuaValueV2::from_value(&lua, v)),
            Err(err) => GoValueV2Result::err(format!("{err}")),
        }
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_table_push(lua: VmHandle, tab: GoLuaValueV2, value: GoLuaValueV2) -> GoNoneResult {
    wrap_failable(|| {
        let lua = lua.get();
        let Some(tab) = get_table(&lua, tab) else {
            return GoNoneResult::err("Table pointer is null".to_string());
        };

        match tab.push(value.to_value(&lua)) {
            Ok(_) => GoNoneResult::ok(),
            Err(err) => GoNoneResult::err(format!("{err}")),
        }
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_table_raw_get(lua: VmHandle, tab: GoLuaValueV2, key: GoLuaValueV2) -> GoValueV2Result {
    wrap_failable(|| {
        let lua = lua.get();
        // Safety: Assume table is a valid, non-null pointer to a Lua Table
        let Some(tab) = get_table(&lua, tab) else {
            return GoValueV2Result::err("Table pointer is null".to_string());
        };

        match tab.raw_get::<mluau::Value>(key.to_value(&lua)) {
            Ok(v) => GoValueV2Result::ok(GoLuaValueV2::from_value(&lua, v)),
            Err(err) => GoValueV2Result::err(format!("{err}")),
        }
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_table_raw_insert(lua: VmHandle, tab: GoLuaValueV2, idx: i64, value: GoLuaValueV2) -> GoNoneResult {
    wrap_failable(|| {
        let lua = lua.get();
        let Some(tab) = get_table(&lua, tab) else {
            return GoNoneResult::err("Table pointer is null".to_string());
        };

        // .to_value(&lua) clones the handle, so it's safe to pass to Lua
        match tab.raw_insert(idx, value.to_value(&lua)) {
            Ok(_) => GoNoneResult::ok(),
            Err(err) => GoNoneResult::err(format!("{err}")),
        }
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_table_raw_len(lua: VmHandle, tab: GoLuaValueV2) -> usize {
    wrap_failable(|| {
        let lua = lua.get();
        let Some(tab) = get_table(&lua, tab) else {
            return 0; 
        };
        tab.raw_len()
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_table_raw_pop(lua: VmHandle, tab: GoLuaValueV2) -> GoValueV2Result {
    wrap_failable(|| {
        let lua = lua.get();
        let Some(tab) = get_table(&lua, tab) else {
            return GoValueV2Result::err("Table pointer is null".to_string());
        };

        match tab.raw_pop::<mluau::Value>() {
            Ok(v) => GoValueV2Result::ok(GoLuaValueV2::from_value(&lua, v)),
            Err(err) => GoValueV2Result::err(format!("{err}")),
        }
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_table_raw_push(lua: VmHandle, tab: GoLuaValueV2, value: GoLuaValueV2) -> GoNoneResult {
    wrap_failable(|| {
        let lua = lua.get();
        let Some(tab) = get_table(&lua, tab) else {
            return GoNoneResult::err("Table pointer is null".to_string());
        };

        match tab.raw_push(value.to_value(&lua)) {
            Ok(_) => GoNoneResult::ok(),
            Err(err) => GoNoneResult::err(format!("{err}")),
        }
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_table_raw_remove(lua: VmHandle, tab: GoLuaValueV2, key: GoLuaValueV2) -> GoNoneResult {
    wrap_failable(|| {
        let lua = lua.get();
        let Some(tab) = get_table(&lua, tab) else {
            return GoNoneResult::err("Table pointer is null".to_string());
        };

        match tab.raw_remove(key.to_value(&lua)) {
            Ok(_) => GoNoneResult::ok(),
            Err(err) => GoNoneResult::err(format!("{err}")),
        }
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_table_raw_set(lua: VmHandle, tab: GoLuaValueV2, key: GoLuaValueV2, value: GoLuaValueV2) -> GoNoneResult {
    wrap_failable(|| {
        let lua = lua.get();
        let Some(tab) = get_table(&lua, tab) else {
            return GoNoneResult::err("Table pointer is null".to_string());
        };

        let key = key.to_value(&lua);
        if key == mluau::Value::Nil {
            return GoNoneResult::err("table key cannot be nil".to_string());
        }

        match tab.raw_set(key, value.to_value(&lua)) {
            Ok(_) => GoNoneResult::ok(),
            Err(err) => GoNoneResult::err(format!("{err}")),
        }
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_table_set(lua: VmHandle, tab: GoLuaValueV2, key: GoLuaValueV2, value: GoLuaValueV2) -> GoNoneResult {
    wrap_failable(|| {
        let lua = lua.get();
        let Some(tab) = get_table(&lua, tab) else {
            return GoNoneResult::err("Table pointer is null".to_string());
        };

        let key = key.to_value(&lua);
        if key == mluau::Value::Nil {
            return GoNoneResult::err("table key cannot be nil".to_string());
        }

        match tab.set(key, value.to_value(&lua)) {
            Ok(_) => GoNoneResult::ok(),
            Err(err) => GoNoneResult::err(format!("{err}")),
        }
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_table_set_metatable(lua: VmHandle, tab: GoLuaValueV2, metatable: GoLuaValueV2) -> GoNoneResult {
    wrap_failable(|| {
        let lua = lua.get();
        let Some(tab) = get_table(&lua, tab) else {
            return GoNoneResult::err("Table pointer is null".to_string());
        };

        // Convert V2 to mluau::Value, then check if it is a Table
        let mt_val = metatable.to_value(&lua);
        let mt_option = match mt_val {
            mluau::Value::Table(t) => Some(t),
            mluau::Value::Nil => None,
            _ => return GoNoneResult::err("Metatable must be a table or nil".to_string()),
        };

        match tab.set_metatable(mt_option) {
            Ok(_) => GoNoneResult::ok(),
            Err(err) => GoNoneResult::err(format!("{err}")),
        }
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_table_set_readonly(lua: VmHandle, tab: GoLuaValueV2, enabled: bool) {
    wrap_failable(|| {
        let lua = lua.get();
        let Some(tab) = get_table(&lua, tab) else {
            return; 
        };
        tab.set_readonly(enabled);
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_table_set_safeenv(lua: VmHandle, tab: GoLuaValueV2, enabled: bool) {
    wrap_failable(|| {
        let lua = lua.get();
        let Some(tab) = get_table(&lua, tab) else {
            return; 
        };
        tab.set_safeenv(enabled);
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_table_debug(lua: VmHandle, tab: GoLuaValueV2) -> *mut c_char {
    wrap_failable(|| {
        let lua = lua.get();
        let Some(tab) = get_table(&lua, tab) else {
            return std::ptr::null_mut(); 
        };

        // luago_result_error_free compatible string creation
        let debug = format!("{tab:#?}");
        let error_cstr = CString::new(debug).unwrap_or_else(|_| CString::new("Invalid error string").unwrap());
        CString::into_raw(error_cstr) as *mut c_char
    })
}