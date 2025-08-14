use std::ffi::{c_char, c_void, CString};
use mluau::{Error, Result};

/// A GoResult struct that encapsulates a result value or an error.
/// 
/// GoResults are stack-allocated and can be used to return results from Rust functions to Go.
#[repr(C)]
pub struct GoResult {
    value: *mut c_void, // Pointer to the result value
    error: *mut c_char,
}

impl GoResult {
    pub fn new<T: Sized + 'static>(value: Result<T>) -> Self {
        match value {
            Ok(val) => GoResult::from_value(val),
            Err(err) => GoResult::from_error(err),
        }
    }

    pub fn from_value<T: Sized + 'static>(value: T) -> Self {
        let boxed_value = Box::new(value);
        GoResult {
            value: Box::into_raw(boxed_value) as *mut c_void,
            error: std::ptr::null_mut(),
        }
    }

    pub fn from_error(error: Error) -> Self {
        let error_str = format!("{error}");
        let error_cstr = CString::new(error_str).unwrap_or_else(|_| CString::new("Invalid error string").unwrap());
        GoResult {
            value: std::ptr::null_mut(),
            error: CString::into_raw(error_cstr) as *mut c_char,
        }
    }
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
