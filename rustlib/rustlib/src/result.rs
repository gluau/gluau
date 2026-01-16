use std::{ffi::{c_char, CString}, panic::AssertUnwindSafe};

use crate::{VmHandle, value_v2::{GoLuaValueV2, GoLuaValueV2Array}, valuearena::Handle};

pub trait Errorable {
    fn error_variant(s: String) -> Self;
}

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
            error: to_c_string(error),
        }
    }
}

impl Errorable for GoNoneResult {
    fn error_variant(s: String) -> Self {
        Self::err(s)
    }
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_none_result_free(ptr: *mut GoNoneResult) {
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
            error: to_c_string(error),
        }
    }
}

impl Errorable for GoBoolResult {
    fn error_variant(s: String) -> Self {
        Self::err(s)
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
            error: to_c_string(error),
        }
    }
}

impl Errorable for GoI64Result {
    fn error_variant(s: String) -> Self {
        Self::err(s)
    }
}

#[repr(C)]
pub struct GoUsizePtrResult {
    value: usize,
    error: *mut c_char
}

impl GoUsizePtrResult {
    pub fn ok(v: usize) -> Self {
        Self {
            value: v,
            error: std::ptr::null_mut(),
        }
    }

    pub fn err(error: String) -> Self {
        Self {
            value: 0,
            error: to_c_string(error),
        }
    }
}

impl Errorable for GoUsizePtrResult {
    fn error_variant(s: String) -> Self {
        Self::err(s)
    }
}

#[repr(C)]
pub struct GoValueV2Result {
    value: GoLuaValueV2,
    error: *mut c_char
}

impl GoValueV2Result {
    pub fn ok(v: GoLuaValueV2) -> Self {
        Self {
            value: v,
            error: std::ptr::null_mut(),
        }
    }

    pub fn err(error: String) -> Self {
        Self {
            value: GoLuaValueV2::empty(),
            error: to_c_string(error),
        }
    }
}

impl Errorable for GoValueV2Result {
    fn error_variant(s: String) -> Self {
        Self::err(s)
    }
}

#[repr(C)]
pub struct GoVmHandleResult {
    value: VmHandle,
    error: *mut c_char
}

impl GoVmHandleResult {
    pub fn ok(v: VmHandle) -> Self {
        Self {
            value: v,
            error: std::ptr::null_mut(),
        }
    }

    pub fn err(error: String) -> Self {
        Self {
            value: VmHandle { handle: Handle { index: 0, generation: 0 } },
            error: to_c_string(error),
        }
    }
}

impl Errorable for GoVmHandleResult {
    fn error_variant(s: String) -> Self {
        Self::err(s)
    }
}

#[repr(C)]
pub struct GoLuaValueV2ArrayResult {
    value: GoLuaValueV2Array,
    error: *mut c_char
}

impl GoLuaValueV2ArrayResult {
    pub fn ok(v: GoLuaValueV2Array) -> Self {
        Self {
            value: v,
            error: std::ptr::null_mut(),
        }
    }

    pub fn err(error: String) -> Self {
        Self {
            value: GoLuaValueV2Array {
                values: std::ptr::null_mut(),
                size: 0,
            },
            error: to_c_string(error),
        }
    }
}

impl Errorable for GoLuaValueV2ArrayResult {
    fn error_variant(s: String) -> Self {
        Self::err(s)
    }
}

/// Given a error string, return a heap allocated error
/// 
/// Useful for API's which have no return
pub fn to_c_string(error: String) -> *mut c_char {
    let error_str = error.replace('\0', ""); // Ensure no null characters in the string
    let error_cstr = CString::new(error_str).unwrap_or_else(|_| CString::new("Invalid error string").unwrap());
    CString::into_raw(error_cstr)
}

/// Given a error string, return a heap allocated error
/// 
/// Useful for API's which have no return
pub fn to_c_string_from_ref(s: &str) -> *mut c_char {
    let error_str = s.replace('\0', ""); // Ensure no null characters in the string
    let error_cstr = CString::new(error_str).unwrap_or_else(|_| CString::new("Invalid error string").unwrap());
    CString::into_raw(error_cstr)
}

// Creates a new CString given string and length
#[unsafe(no_mangle)]
pub extern "C" fn luago_string_new(s: *const c_char, len: usize) -> *mut c_char {
    if s.is_null() || len == 0 {
        let c_string = CString::new("").unwrap_or_else(|_| CString::new("Invalid string").unwrap());
        return CString::into_raw(c_string);
    }
    // Safety: Assume s points to a valid C string of length len.
    let slice = unsafe { std::slice::from_raw_parts(s as *const u8, len) };
    let c_string = CString::new(slice).unwrap_or_else(|_| CString::new("Invalid string").unwrap());
    // Convert CString to raw pointer
    CString::into_raw(c_string)
}

/// Frees the memory for an error string created by Rust.
#[unsafe(no_mangle)]
pub extern "C" fn luago_string_free(error_ptr: *mut c_char) {
    if !error_ptr.is_null() {
        // Reconstruct the CString from the raw pointer and let it drop,
        // which deallocates the memory.
        unsafe { drop(CString::from_raw(error_ptr)); }
    }
}

/// Helper to wrap a Errorable in a catch_unwind
pub fn wrap_failable<T: Errorable>(f: impl FnOnce() -> T) -> T {
    match std::panic::catch_unwind(AssertUnwindSafe(|| f())) {
        Ok(t) => t,
        Err(e) => {
            if let Some(s) = e.downcast_ref::<&str>() {
                T::error_variant(s.to_string())
            } else if let Some(s) = e.downcast_ref::<String>() {
                T::error_variant(s.to_string())
            } else {
                T::error_variant("unknown panic reason".to_string())
            }
        }
    }
}

impl Errorable for () {
    fn error_variant(_s: String) -> Self {
        ()
    }
}

impl Errorable for u8 {
    fn error_variant(_s: String) -> Self {
        0
    }
}

impl Errorable for usize {
    fn error_variant(_s: String) -> Self {
        0
    }
}

impl Errorable for bool {
    fn error_variant(_s: String) -> Self {
        false
    }
}


impl<T> Errorable for *mut T {
    fn error_variant(_s: String) -> Self {
        std::ptr::null_mut()
    }
}

impl Errorable for GoLuaValueV2 {
    fn error_variant(_s: String) -> Self {
        GoLuaValueV2::empty()
    }
}