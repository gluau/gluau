use std::ffi::{c_void, CString};

use crate::LuaVmWrapper;

#[repr(C)]
pub struct OpaqueCVec {
    ptr: *mut *mut c_void,
    len: usize,
    cap: usize,
}

#[repr(C)]
pub enum LuaValueType {
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
    Error = 12,
    Other = 13,
}

impl LuaValueType {
    pub fn from_value(value: &mluau::Value) -> Self {
        match value {
            mluau::Value::Nil => LuaValueType::Nil,
            mluau::Value::Boolean(_) => LuaValueType::Boolean,
            mluau::Value::LightUserData(_) => LuaValueType::LightUserData,
            mluau::Value::Integer(_) => LuaValueType::Integer,
            mluau::Value::Number(_) => LuaValueType::Number,
            mluau::Value::Vector(_) => LuaValueType::Vector,
            mluau::Value::String(_) => LuaValueType::String,
            mluau::Value::Table(_) => LuaValueType::Table,
            mluau::Value::Function(_) => LuaValueType::Function,
            mluau::Value::Thread(_) => LuaValueType::Thread,
            mluau::Value::UserData(_) => LuaValueType::UserData,
            mluau::Value::Buffer(_) => LuaValueType::Buffer,
            mluau::Value::Error(_) => LuaValueType::Error,
            _ => LuaValueType::Other, // Other types
        }
    }
}

#[repr(C)]
pub union LuaValueData {
    boolean: bool,
    light_userdata: *mut c_void,
    integer: i64,
    number: f64,
    vector: [f32; 3], 
    string: *mut mluau::String,
    table: *mut mluau::Table,
    function: *mut mluau::Function,
    thread: *mut mluau::Thread,
    userdata: *mut mluau::AnyUserData,
    buffer: *mut mluau::Buffer,
    error: *mut i8,
    other: *mut c_void, // Placeholder for other types
}

#[repr(C)]
pub struct GoLuaValue {
    tag: LuaValueType,
    data: LuaValueData,
}

impl GoLuaValue {
    pub fn from_ref(value: &mluau::Value) -> Self {
        let tag = LuaValueType::from_value(value);
        let data = match value {
            mluau::Value::Nil => LuaValueData { boolean: false },
            mluau::Value::Boolean(b) => LuaValueData { boolean: *b },
            mluau::Value::LightUserData(ptr) => LuaValueData { light_userdata: ptr.0 },
            mluau::Value::Integer(i) => LuaValueData { integer: *i },
            mluau::Value::Number(n) => LuaValueData { number: *n },
            mluau::Value::Vector(v) => LuaValueData { vector: [v.x(), v.y(), v.z()] },
            mluau::Value::String(s) => LuaValueData { string: Box::into_raw(Box::new(s.clone())) },
            mluau::Value::Table(t) => LuaValueData { table: Box::into_raw(Box::new(t.clone())) },
            mluau::Value::Function(f) => LuaValueData { function: Box::into_raw(Box::new(f.clone())) },
            mluau::Value::Thread(t) => LuaValueData { thread: Box::into_raw(Box::new(t.clone())) },
            mluau::Value::UserData(ud) => LuaValueData { userdata: Box::into_raw(Box::new(ud.clone())) },
            mluau::Value::Buffer(buf) => LuaValueData { buffer: Box::into_raw(Box::new(buf.clone())) },
            mluau::Value::Error(err) => {
                let err_str = format!("{err}");
                let err_cstr = CString::new(err_str).unwrap_or_else(|_| CString::new("Invalid error string").unwrap());
                // Store the error as a CString to ensure proper memory management
                let err_ptr = CString::into_raw(err_cstr);
                LuaValueData { error: err_ptr }
            },
            _ => LuaValueData { other: std::ptr::null_mut() }, // Handle other types
        };
        GoLuaValue { tag, data }
    }

    pub fn from_owned(value: mluau::Value) -> Self {
        let tag = LuaValueType::from_value(&value);
        let data = match value {
            mluau::Value::Nil => LuaValueData { boolean: false },
            mluau::Value::Boolean(b) => LuaValueData { boolean: b },
            mluau::Value::LightUserData(ptr) => LuaValueData { light_userdata: ptr.0 },
            mluau::Value::Integer(i) => LuaValueData { integer: i },
            mluau::Value::Number(n) => LuaValueData { number: n },
            mluau::Value::Vector(v) => LuaValueData { vector: [v.x(), v.y(), v.z()] },
            mluau::Value::String(s) => LuaValueData { string: Box::into_raw(Box::new(s)) },
            mluau::Value::Table(t) => LuaValueData { table: Box::into_raw(Box::new(t)) },
            mluau::Value::Function(f) => LuaValueData { function: Box::into_raw(Box::new(f)) },
            mluau::Value::Thread(t) => LuaValueData { thread: Box::into_raw(Box::new(t)) },
            mluau::Value::UserData(ud) => LuaValueData { userdata: Box::into_raw(Box::new(ud)) },
            mluau::Value::Buffer(buf) => LuaValueData { buffer: Box::into_raw(Box::new(buf)) },
            mluau::Value::Error(err) => {
                let err_str = format!("{err}");
                let err_cstr = CString::new(err_str).unwrap_or_else(|_| CString::new("Invalid error string").unwrap());
                // Store the error as a CString to ensure proper memory management
                let err_ptr = CString::into_raw(err_cstr);
                LuaValueData { error: err_ptr }
            },
            mluau::Value::Other(_) => LuaValueData { other: std::ptr::null_mut() }, // TODO: Handle other types
        };
        GoLuaValue { tag, data }
    }
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_error_free(error: *mut i8) {
    // Safety: Assume error is a valid, non-null pointer to a C string
    if error.is_null() {
        return;
    }

    // Reconstruct the CString and let it drop, freeing the memory.
    unsafe {
        drop(CString::from_raw(error));
    }
}

#[repr(C)]
pub struct DebugValue {
    values: [GoLuaValue; 2],
}

// test api
#[unsafe(no_mangle)]
pub extern "C" fn luago_dbg_value(ptr: *mut LuaVmWrapper) -> DebugValue {
    // Create a dummy Lua value for testing purposes
        // Safety: Assume ptr is a valid, non-null pointer to a LuaVmWrapper
    // and that s points to a valid C string of length len.
    let res = unsafe {
        let lua = &mut (*ptr).lua;
        lua.create_string("Testing testing 123").unwrap()
    };

    DebugValue {
        values: [
        GoLuaValue::from_owned(mluau::Value::String(res)),
        GoLuaValue::from_owned(mluau::Value::Error(mluau::Error::external("This is a test error".to_string()).into())),
        ]
    }
}