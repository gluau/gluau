use std::{ffi::{c_void, CString}, sync::Arc};

use crate::{string::LuaStringBytes, LuaVmWrapper};

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

pub struct ErrorVariant {
    pub error: Arc<CString>,
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
    error: *mut ErrorVariant,
    other: *mut c_void, // Placeholder for other types
}

#[repr(C)]
pub struct GoLuaValue {
    tag: LuaValueType,
    data: LuaValueData,
}

impl GoLuaValue {
    // # Safety
    //
    // This function guarantees to not free any memory that is managed by Go.
    // It only converts the LuaValueType to a mluau::Value.
    //
    // In addition to this, it is safe to call this function multiple times
    // on the same GoLuaValue, as it does not mutate the internal state.
    pub fn to_value(&self) -> mluau::Value {
        match self.tag {
            LuaValueType::Nil => mluau::Value::Nil,
            LuaValueType::Boolean => {
                let boolean = unsafe { self.data.boolean };
                mluau::Value::Boolean(boolean)
            },
            LuaValueType::LightUserData => {
                let light_userdata = unsafe { self.data.light_userdata };
                mluau::Value::LightUserData(mluau::LightUserData(light_userdata))
            },
            LuaValueType::Integer => {
                let integer = unsafe { self.data.integer };
                mluau::Value::Integer(integer)
            },
            LuaValueType::Number => {
                let number = unsafe { self.data.number };
                mluau::Value::Number(number)
            },
            LuaValueType::Vector => {
                let vector = unsafe { self.data.vector };
                mluau::Value::Vector(mluau::Vector::new(vector[0], vector[1], vector[2]))
            },
            LuaValueType::String => {
                let string_ptr = unsafe { self.data.string };
                if string_ptr.is_null() {
                    mluau::Value::Nil
                } else {
                    // Safety: Avoid free'ing the string pointer here, as it is managed by Go
                    let string_ptr = unsafe { &*string_ptr };
                    mluau::Value::String(string_ptr.clone())
                }
            },
            LuaValueType::Table => {
                let table_ptr = unsafe { self.data.table };
                if table_ptr.is_null() {
                    mluau::Value::Nil
                } else {
                    // Safety: Avoid free'ing the table pointer here, as it is managed by Go
                    let table_ptr = unsafe { &*table_ptr };
                    mluau::Value::Table(table_ptr.clone())
                }
            },
            LuaValueType::Function => {
                let function_ptr = unsafe { self.data.function };
                if function_ptr.is_null() {
                    mluau::Value::Nil
                } else {
                    let function_ptr = unsafe { &*function_ptr };
                    // Safety: Avoid free'ing the function pointer here, as it is managed by Go
                    mluau::Value::Function(function_ptr.clone())
                }
            },
            LuaValueType::Thread => {
                let thread_ptr = unsafe { self.data.thread };
                if thread_ptr.is_null() {
                    mluau::Value::Nil
                } else {
                    // Safety: Avoid free'ing the thread pointer here, as it is managed by Go
                    let thread_ptr = unsafe { &*thread_ptr };
                    mluau::Value::Thread(thread_ptr.clone())
                }
            },
            LuaValueType::UserData => {
                let userdata_ptr = unsafe { self.data.userdata };
                if userdata_ptr.is_null() {
                    mluau::Value::Nil
                } else {
                    // Safety: Avoid free'ing the userdata pointer here, as it is managed by Go
                    let userdata_ptr = unsafe { &*userdata_ptr };
                    mluau::Value::UserData(userdata_ptr.clone())
                }
            },
            LuaValueType::Buffer => {
                let buffer_ptr = unsafe { self.data.buffer };
                if buffer_ptr.is_null() {
                    mluau::Value::Nil
                } else {
                    let buffer_ptr = unsafe { &*buffer_ptr };
                    mluau::Value::Buffer(buffer_ptr.clone())   
                }
            },
            LuaValueType::Error => {
                let error_ptr = unsafe { self.data.error };
                if error_ptr.is_null() {
                    mluau::Value::Nil
                } else {
                    // Safety: Avoid free'ing the error pointer here, as it is managed by Go
                    let error_variant = unsafe { &*error_ptr };
                    let error_string = error_variant.error.to_string_lossy().into_owned();
                    mluau::Value::Error(mluau::Error::external(error_string).into())
                }
            },
            LuaValueType::Other => {
                // Handle other types, currently returning Nil
                mluau::Value::Nil
            },
        }
    }

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
                let err_ptr = Arc::new(err_cstr);
                let ptr = Box::into_raw(Box::new(ErrorVariant {
                    error: err_ptr
                }));
                LuaValueData { error: ptr }
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
                let err_ptr = Arc::new(err_cstr);
                let ptr = Box::into_raw(Box::new(ErrorVariant {
                    error: err_ptr
                }));
                LuaValueData { error: ptr }
            },
            mluau::Value::Other(_) => LuaValueData { other: std::ptr::null_mut() }, // TODO: Handle other types
        };
        GoLuaValue { tag, data }
    }
}

// Creates a new error variant given char array and length
#[unsafe(no_mangle)]
pub extern "C" fn luago_error_new(error: *const i8, len: usize) -> *mut ErrorVariant {
    // Safety: Assume error is a valid, non-null pointer to a C string of length len.
    if error.is_null() || len == 0 {
        return std::ptr::null_mut();
    }
    // Convert the C string to a Rust CString
    let c_str = unsafe { std::slice::from_raw_parts(error as *const u8, len) };
    let c_string = CString::new(c_str).unwrap_or_else(|_| {
        // If the CString creation fails, return a null pointer
        CString::new("Invalid error string").unwrap()
    });
    let arc_str = Arc::new(c_string);

    let ptr = Box::into_raw(Box::new(ErrorVariant {
        error: arc_str,
    }));

    ptr
}

// Returns the inner error string from the ErrorVariant
#[unsafe(no_mangle)]
pub extern "C" fn luago_error_get_string(error: *mut ErrorVariant) -> super::string::LuaStringBytes {
    // Safety: Assume error is a valid, non-null pointer to an ErrorVariant
    if error.is_null() {
        return LuaStringBytes {
            data: std::ptr::null(),
            size: 0,
        };
    }

    // Reconstruct the ErrorVariant and get the error string
    let error_variant = unsafe { &*error };
    let error_string = error_variant.error.to_str().unwrap_or("Invalid error string");

    LuaStringBytes {
        data: error_string.as_ptr(),
        size: error_string.len(),
    }
}

// Needed to free a error string
#[unsafe(no_mangle)]
pub extern "C" fn luago_error_free(error: *mut ErrorVariant) {
    // Safety: Assume error is a valid, non-null pointer to a C string
    if error.is_null() {
        return;
    }

    // Reconstruct the ErrorVariant and drop it
    unsafe {
        drop(Box::from_raw(error));
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