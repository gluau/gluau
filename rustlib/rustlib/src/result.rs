use std::ffi::{c_char, CString};

use crate::{multivalue::GoMultiValue, value::GoLuaValue};

#[repr(C)]
pub struct GoNoneResult {
    error: *mut c_char
}

impl GoNoneResult {
    pub fn ok() -> Self {
        Self {
            error: std::ptr::null_mut(),
        }
    }

    pub fn err(error: String) -> Self {
        Self {
            error: to_error(error),
        }
    }
}

#[unsafe(no_mangle)]
pub extern "C-unwind" fn luago_none_result_free(ptr: *mut GoNoneResult) {
    if ptr.is_null() {
        return;
    }

    unsafe { drop(Box::from_raw(ptr)); }
}

#[repr(C)]
pub struct GoBoolResult {
    value: bool,
    error: *mut c_char
}

impl GoBoolResult {
    pub fn ok(b: bool) -> Self {
        Self {
            value: b,
            error: std::ptr::null_mut(),
        }
    }

    pub fn err(error: String) -> Self {
        Self {
            value: false,
            error: to_error(error),
        }
    }
}

#[repr(C)]
pub struct GoI64Result {
    value: i64,
    error: *mut c_char
}

impl GoI64Result {
    pub fn ok(v: i64) -> Self {
        Self {
            value: v,
            error: std::ptr::null_mut(),
        }
    }

    pub fn err(error: String) -> Self {
        Self {
            value: 0,
            error: to_error(error),
        }
    }
}

#[repr(C)]
pub struct GoStringResult {
    value: *mut mluau::String,
    error: *mut c_char
}

impl GoStringResult {
    pub fn ok(s: *mut mluau::String) -> Self {
        Self {
            value: s,
            error: std::ptr::null_mut(),
        }
    }

    pub fn err(error: String) -> Self {
        Self {
            value: std::ptr::null_mut(),
            error: to_error(error),
        }
    }
}

#[repr(C)]
pub struct GoTableResult {
    value: *mut mluau::Table,
    error: *mut c_char
}

impl GoTableResult {
    pub fn ok(t: *mut mluau::Table) -> Self {
        Self {
            value: t,
            error: std::ptr::null_mut(),
        }
    }

    pub fn err(error: String) -> Self {
        Self {
            value: std::ptr::null_mut(),
            error: to_error(error),
        }
    }
}

#[unsafe(no_mangle)]
pub extern "C-unwind" fn luago_tableresult_free(ptr: *mut GoTableResult) {
    if ptr.is_null() {
        return;
    }

    unsafe { drop(Box::from_raw(ptr)); }    
}

#[repr(C)]
pub struct GoFunctionResult {
    value: *mut mluau::Function,
    error: *mut c_char
}

impl GoFunctionResult {
    pub fn ok(f: *mut mluau::Function) -> Self {
        Self {
            value: f,
            error: std::ptr::null_mut(),
        }
    }

    pub fn err(error: String) -> Self {
        Self {
            value: std::ptr::null_mut(),
            error: to_error(error),
        }
    }
}

#[repr(C)]
pub struct GoMultiValueResult {
    value: *mut GoMultiValue,
    error: *mut c_char
}

impl GoMultiValueResult {
    pub fn ok(f: *mut GoMultiValue) -> Self {
        Self {
            value: f,
            error: std::ptr::null_mut(),
        }
    }

    pub fn err(error: String) -> Self {
        Self {
            value: std::ptr::null_mut(),
            error: to_error(error),
        }
    }
}

#[repr(C)]
pub struct GoValueResult {
    value: GoLuaValue,
    error: *mut c_char
}

impl GoValueResult {
    pub fn ok(v: GoLuaValue) -> Self {
        Self {
            value: v,
            error: std::ptr::null_mut(),
        }
    }

    pub fn err(error: String) -> Self {
        Self {
            value: GoLuaValue::from_owned(mluau::Value::Nil),
            error: to_error(error),
        }
    }
}

/// Given a error string, return a heap allocated error
/// 
/// Useful for API's which have no return
pub fn to_error(error: String) -> *mut c_char {
    let error_str = error.replace('\0', ""); // Ensure no null characters in the string
    let error_cstr = CString::new(error_str).unwrap_or_else(|_| CString::new("Invalid error string").unwrap());
    CString::into_raw(error_cstr)
}

/// Frees the memory for an error string created by Rust.
#[unsafe(no_mangle)]
pub extern "C-unwind" fn luago_result_error_free(error_ptr: *mut c_char) {
    if !error_ptr.is_null() {
        // Reconstruct the CString from the raw pointer and let it drop,
        // which deallocates the memory.
        unsafe { drop(CString::from_raw(error_ptr)); }
    }
}
