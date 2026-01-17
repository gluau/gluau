use crate::result::Errorable;

#[repr(C)]
/// Bytes which are owned by Rust and must be freed by Rust 
pub struct RustOwnedBytes {
    // Pointer to the borrowed bytes
    pub data: *const u8,
    // Length of the borrowed bytes
    pub size: usize,
}

impl RustOwnedBytes {
    pub fn new(data: Vec<u8>) -> Self {
        let boxed_slice = data.into_boxed_slice();
        let size = boxed_slice.len();
        let data_ptr = boxed_slice.as_ptr();
        // Prevent Rust from freeing the memory
        std::mem::forget(boxed_slice);
        
        Self {
            data: data_ptr,
            size,
        }
    }
}

impl Errorable for RustOwnedBytes {
    fn error_variant(_s: String) -> Self {
        Self {
            data: std::ptr::null(),
            size: 0,
        }
    }
}

#[repr(C)]
/// Bytes which are owned by Go and must be freed by Go (not Rust)
pub struct GoOwnedBytes {
    // Pointer to the borrowed bytes
    pub data: *mut u8,
    // Length of the borrowed bytes
    pub size: usize,
}

impl GoOwnedBytes {
    pub fn with<F, R>(&self, f: F) -> R
    where
        F: FnOnce(&[u8]) -> R,
    {
        let slice = unsafe { std::slice::from_raw_parts(self.data, self.size) };
        f(slice)
    }

    /// Helper method to copy bytes from a Rust-owned byte slice to a Go-owned byte slice
    /// 
    /// Safety: go_buf must be a valid pointer to a buffer of at least go_buf_len bytes
    pub fn copy_rust_bytes_to_go(&self, rust_bytes: &[u8]) -> usize {
        let bytes_to_copy = std::cmp::min(rust_bytes.len(), self.size);
        if bytes_to_copy == 0 {
            return 0;
        }

        unsafe {
            std::ptr::copy_nonoverlapping(
                rust_bytes.as_ptr(),
                self.data,
                bytes_to_copy,
            );
        }

        bytes_to_copy
    }
}

#[unsafe(no_mangle)]
/// Safety: ptr must be a valid pointer to an ExternalString allocated by either Rust or Luau
pub extern "C" fn luago_rustownedbytes_free(ptr: *mut RustOwnedBytes) {
    if ptr.is_null() {
        return;
    }

    unsafe { drop(Box::from_raw(ptr)); }
}