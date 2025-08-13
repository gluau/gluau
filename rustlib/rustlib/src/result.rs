use std::ffi::{c_char, c_void, CString};
use mluau::{Error, Result};

// Rust code
#[repr(C)]
pub struct GoResult {
    value: *mut c_void, // Pointer to the result value
    error: *mut c_char,
}

impl GoResult {
    pub fn new_ptr<T: Sized + 'static>(value: Result<T>) -> *mut GoResult {
        let result = GoResult::new(value);
        Box::into_raw(Box::new(result))
    }

    fn new<T: Sized + 'static>(value: Result<T>) -> Self {
        match value {
            Ok(val) => GoResult::from_value(val),
            Err(err) => GoResult::from_error(err),
        }
    }

    fn from_value<T: Sized + 'static>(value: T) -> Self {
        let boxed_value = Box::new(value);
        GoResult {
            value: Box::into_raw(boxed_value) as *mut c_void,
            error: std::ptr::null_mut(),
        }
    }

    fn from_error(error: Error) -> Self {
        let error_str = format!("{:?}", error);
        let error_cstr = CString::new(error_str).unwrap_or_else(|_| CString::new("Invalid error string").unwrap());
        let boxed_error = Box::new(error_cstr);
        GoResult {
            value: std::ptr::null_mut(),
            error: Box::into_raw(boxed_error) as *mut c_char,
        }
    }
}

#[unsafe(no_mangle)]
/// Free's the memory allocated for a GoResult.
/// 
/// Note: the underlying value pointer will not be free'd here (for obvious reasons)
pub extern "C" fn luago_result_free(result: *mut GoResult) {
    // Safety: Assume result is a valid, non-null pointer to a GoResult
    if result.is_null() {
        return;
    }

    // Re-box the GoResult pointer to manage its memory automatically.
    // When `res` goes out of scope, the GoResult struct itself will be freed.
    let res = unsafe { Box::from_raw(result) };
    
    // If there was an error string, reconstruct the CString and let it drop.
    if !res.error.is_null() {
        unsafe {
            drop(CString::from_raw(res.error));
        }
    }
}