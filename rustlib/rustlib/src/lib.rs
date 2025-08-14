pub mod vm;
pub mod string;
pub mod table;
pub mod result;
pub mod value;

use mluau::Lua;
use std::ffi::c_void;

// typedef void (*Callback)(void* val, void* handle);
// typedef void (*DropCallback)(void* handle);
type Callback = extern "C" fn(val: *mut c_void, handle: usize);
type DropCallback = extern "C" fn(handle: usize);

#[repr(C)]
pub struct IGoCallback {
    callback: Callback,
    drop: DropCallback,
    handle: usize,
}

pub struct IGoCallbackWrapper {
    callback: *mut IGoCallback,
}

impl IGoCallbackWrapper {
    pub fn new(callback: *mut IGoCallback) -> Self {
        IGoCallbackWrapper { callback }
    }

    pub fn callback(&self, val: *mut c_void) {
        if self.callback.is_null() {
            return;
        }
        let cb = unsafe { &*self.callback };
        // Ensure the callback function is valid before calling it.
        // This prevents dereferencing a null pointer or calling an invalid function.
        if cb.handle != 0 {
            (cb.callback)(val, cb.handle);
        }
    }
}

impl Drop for IGoCallbackWrapper {
    fn drop(&mut self) {
        // Safety: Call the drop function if it exists.
        if self.callback.is_null() {
            return;
        }
        let cb = unsafe { &*self.callback };
        // Ensure the drop function is called only if the handle is not null.
        // This prevents double freeing or calling drop on an invalid handle.
        if cb.handle != 0 {
            (cb.drop)(cb.handle);
        }
    }
} 

// Test callbacks
//void test_callback(struct IGoCallback* cb, void* val);

#[unsafe(no_mangle)]
pub extern "C-unwind" fn test_callback(cb: *mut IGoCallback, val: *mut c_void) {
    // Safety: Call the callback function with the provided value.
    let wrapper = IGoCallbackWrapper::new(cb);
    wrapper.callback(val);
}

pub struct LuaVmWrapper {
    pub lua: Lua,
}
