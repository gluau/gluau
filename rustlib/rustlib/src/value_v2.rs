use mluau::Lua;

use crate::{result::{GoBoolResult, wrap_failable}, valuearena::{Arena, Handle}};

#[repr(C)]
pub enum LuaValueTypeV2 {
    Nil = 0,
    Boolean = 1,
    LightUserData = 2,
    Integer = 3,
    Number = 4,
    Vector = 5,
    String = 6,
    Table = 7,
    Function = 8,
    Thread = 9,
    UserData = 10,
    Buffer = 11,
    Other = 12,
}

impl LuaValueTypeV2 {
    pub fn from_value(value: &mluau::Value) -> Self {
        match value {
            mluau::Value::Nil => LuaValueTypeV2::Nil,
            mluau::Value::Boolean(_) => LuaValueTypeV2::Boolean,
            mluau::Value::LightUserData(_) => LuaValueTypeV2::LightUserData,
            mluau::Value::Integer(_) => LuaValueTypeV2::Integer,
            mluau::Value::Number(_) => LuaValueTypeV2::Number,
            mluau::Value::Vector(_) => LuaValueTypeV2::Vector,
            mluau::Value::String(_) => LuaValueTypeV2::String,
            mluau::Value::Table(_) => LuaValueTypeV2::Table,
            mluau::Value::Function(_) => LuaValueTypeV2::Function,
            mluau::Value::Thread(_) => LuaValueTypeV2::Thread,
            mluau::Value::UserData(_) => LuaValueTypeV2::UserData,
            mluau::Value::Buffer(_) => LuaValueTypeV2::Buffer,
            mluau::Value::Error(_) => LuaValueTypeV2::Other,
            mluau::Value::Other(_) => LuaValueTypeV2::Other, // TODO: Handle other types
        }
    }
}

#[repr(C)]
#[derive(Clone, Copy)]
pub union LuaValueDataV2 {
    boolean: bool,
    integer: i64,
    number: f64,
    vector: [f32; 3], 
    handle: Handle,
    lightuserdata: *mut std::ffi::c_void,
}

#[repr(C)]
pub struct GoLuaValueV2 {
    tag: LuaValueTypeV2,
    data: LuaValueDataV2,
}

impl GoLuaValueV2 {
    pub fn empty() -> Self {
        GoLuaValueV2 {
            tag: LuaValueTypeV2::Nil,
            data: LuaValueDataV2 { integer: 0 },
        }
    }

    pub fn from_value(lua: &Lua, value: mluau::Value) -> Self {
        let tag = LuaValueTypeV2::from_value(&value);
        let data = match value {
            mluau::Value::Boolean(b) => LuaValueDataV2 { boolean: b },
            mluau::Value::Integer(i) => LuaValueDataV2 { integer: i },
            mluau::Value::Number(n) => LuaValueDataV2 { number: n },
            mluau::Value::Vector(v) => LuaValueDataV2 { vector: [v.x(), v.y(), v.z()] },
            mluau::Value::Nil => LuaValueDataV2 { integer: 0 }, // Placeholder, no data needed
            mluau::Value::LightUserData(ud) => LuaValueDataV2 { lightuserdata: ud.0 },
            _ => {
                match lua.app_data_mut::<Arena<mluau::Value>>() {
                    Some(mut arena) => {
                        let handle = arena.insert(value);
                        LuaValueDataV2 { handle }
                    }
                    None => {
                        let mut arena = Arena::new();
                        let handle = arena.insert(value);
                        lua.set_app_data(arena);
                        LuaValueDataV2 { handle }
                    }
                }
            },
        };
        GoLuaValueV2 { tag, data }
    }

    /// Note: This *always* clones the value for non-primitive types.
    pub fn to_value(&self, lua: &Lua) -> mluau::Value {
        unsafe {
            match self.tag {
                LuaValueTypeV2::Boolean => mluau::Value::Boolean(self.data.boolean),
                LuaValueTypeV2::Integer => mluau::Value::Integer(self.data.integer),
                LuaValueTypeV2::Number => mluau::Value::Number(self.data.number),
                LuaValueTypeV2::Vector => {
                    let vec = self.data.vector;
                    mluau::Value::Vector(mluau::Vector::new(vec[0], vec[1], vec[2]))
                }
                LuaValueTypeV2::Nil => mluau::Value::Nil,
                LuaValueTypeV2::LightUserData => {
                    let ptr = self.data.lightuserdata;
                    mluau::Value::LightUserData(mluau::LightUserData(ptr))
                }
                _ => {
                    let Some(arena) = lua.app_data_ref::<Arena<mluau::Value>>() else {
                        eprintln!("Warning: Lua has no Arena for GoLuaValueV2");
                        return mluau::Value::Nil
                    };
                    let other_ptr = self.data.handle;
                    match arena.get(other_ptr) {
                        Some(v) => v.clone(),
                        None => {
                            eprintln!("Warning: Handle {:?} not found in Arena", other_ptr);
                            mluau::Value::Nil
                        }
                    }
                }
            }
        }
    }

    /// Destroys the GoLuaValue,
    pub fn destroy(self, lua: &Lua) {
        unsafe {
            match self.tag {
                | LuaValueTypeV2::Boolean
                | LuaValueTypeV2::Integer
                | LuaValueTypeV2::Number
                | LuaValueTypeV2::Vector 
                | LuaValueTypeV2::Nil
                | LuaValueTypeV2::LightUserData => {
                    // No heap allocation, nothing to free
                }
                _ => {
                    let other_ptr = self.data.handle;
                    if let Some(mut arena) = lua.app_data_mut::<Arena<mluau::Value>>() {
                        arena.remove(other_ptr);
                    } else {
                        eprintln!("Warning: Lua has no Arena for GoLuaValueV2 during destroy");
                    }
                }
            }
        }
    }
}

// Note: this is safe to call multiple times thanks to the memory arena.
#[unsafe(no_mangle)]
pub extern "C" fn luago_valuev2_destroy(lua: *mut mluau::Lua, value: GoLuaValueV2) {
    wrap_failable(|| {
        let lua = unsafe { &*lua };
        value.destroy(lua);
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_valuev2_topointer(lua: *mut mluau::Lua, value: GoLuaValueV2) -> usize {
    wrap_failable(|| {
        let lua = unsafe { &*lua };
        let val = value.to_value(lua);
        val.to_pointer() as usize
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_valuev2_equals(lua: *mut mluau::Lua, a: GoLuaValueV2, b: GoLuaValueV2) -> GoBoolResult {
    wrap_failable(|| {
        let lua = unsafe { &*lua };
        let t1 = a.to_value(lua);
        let t2 = b.to_value(lua);

        match t1.equals(&t2) {
            Ok(eq) => GoBoolResult::ok(eq),
            Err(e) => GoBoolResult::err(e.to_string()),
        }
    })
}

#[repr(C)]
pub struct GoLuaValueV2Array {
    pub values: *mut GoLuaValueV2,
    pub size: usize,
}

impl GoLuaValueV2Array {
    pub fn to_values(&self, lua: &Lua) -> mluau::MultiValue {
        if self.size == 0 || self.values.is_null() {
            return mluau::MultiValue::with_capacity(0);
        }
        let slice = unsafe { std::slice::from_raw_parts(self.values, self.size) };
        let mut mv = mluau::MultiValue::with_capacity(self.size);
        for v in slice {
            mv.push_back(v.to_value(lua));
        }
        mv
    }

    /// Create a empty GoLuaValueV2Array with 0 size
    pub fn empty() -> Self {
        GoLuaValueV2Array {
            values: std::ptr::null_mut(),
            size: 0,
        }
    }

    /// Convert a rust mluau::MultiValue into a GoLuaValueV2Array
    pub fn from_values(lua: &Lua, values: mluau::MultiValue) -> Self {
        let size = values.len();
        if size == 0 {
            return GoLuaValueV2Array {
                values: std::ptr::null_mut(),
                size: 0,
            };
        }
        let mut vec = Vec::with_capacity(size);
        for v in values.into_iter() {
            vec.push(GoLuaValueV2::from_value(lua, v));
        }
        let boxed_slice = vec.into_boxed_slice();
        let ptr = Box::into_raw(boxed_slice) as *mut GoLuaValueV2;
        GoLuaValueV2Array { values: ptr, size }
    }

    /// Destroys the GoLuaValueV2Array, freeing any allocated memory.
    /// 
    /// Note: This does NOT destroy the inner GoLuaValueV2 values.
    pub fn destroy(self) {
        if self.size == 0 || self.values.is_null() {
            return;
        }
        unsafe {
            let _ = Box::from_raw(std::slice::from_raw_parts_mut(self.values, self.size));
        }
    }
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_valuev2array_alloc(size: usize) -> GoLuaValueV2Array {
    if size == 0 {
        return GoLuaValueV2Array {
            values: std::ptr::null_mut(),
            size: 0,
        };
    }
    let mut vec = Vec::with_capacity(size);
    for _ in 0..size {
        vec.push(GoLuaValueV2::empty());
    }
    let boxed_slice = vec.into_boxed_slice();
    let ptr = Box::into_raw(boxed_slice) as *mut GoLuaValueV2;
    GoLuaValueV2Array { values: ptr, size }
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_valuev2array_destroy(array: GoLuaValueV2Array) {
    println!("Destroying GoLuaValueV2Array of size {}", array.size);
    wrap_failable(|| {
        array.destroy();
    })
}