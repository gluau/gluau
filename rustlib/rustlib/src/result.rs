use std::ffi::{c_char, CString};
use mluau::Error;

use crate::value::GoLuaValue;

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

    pub fn err(error: Error) -> Self {
        Self {
            error: to_error(error),
        }
    }
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

    pub fn err(error: Error) -> Self {
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

    pub fn err(error: Error) -> Self {
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

    pub fn err(error: Error) -> Self {
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

    pub fn err(error: Error) -> Self {
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

    pub fn err(error: Error) -> Self {
        Self {
            value: GoLuaValue::from_owned(mluau::Value::Nil),
            error: to_error(error),
        }
    }
}

/// Given a error string, return a heap allocated error
/// 
/// Useful for API's which have no return
pub fn to_error(error: Error) -> *mut c_char {
    let error_str = format!("{error}");
    let error_cstr = CString::new(error_str).unwrap_or_else(|_| CString::new("Invalid error string").unwrap());
    CString::into_raw(error_cstr) as *mut c_char
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
