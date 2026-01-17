pub mod vm;
pub mod string;
pub mod table;
pub mod function;
pub mod result;
pub mod value_v2;
pub mod valuearena;
pub mod compiler;
pub mod chunk;
pub mod userdata;
pub mod thread;
pub mod buffer;
pub mod require;
pub mod externalstring;

use std::ffi::c_void;

use crate::{valuearena::{Arena, Handle}};

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
    callback: IGoCallback,
}

impl IGoCallbackWrapper {
    pub fn new(callback: IGoCallback) -> Self {
        IGoCallbackWrapper { callback }
    }

    pub fn callback(&self, val: *mut c_void) {
        // Ensure the callback function is valid before calling it.
        // This prevents dereferencing a null pointer or calling an invalid function.
        if self.callback.handle != 0 {
            (self.callback.callback)(val, self.callback.handle);
        }
    }
}

impl Drop for IGoCallbackWrapper {
    fn drop(&mut self) {
        // Ensure the drop function is called only if the handle is not null.
        // This prevents double freeing or calling drop on an invalid handle.
        if self.callback.handle != 0 {
            (self.callback.drop)(self.callback.handle);
        }
    }
} 

// Test callbacks
//void test_callback(struct IGoCallback* cb, void* val);

#[unsafe(no_mangle)]
pub extern "C" fn test_callback(cb: IGoCallback, val: *mut c_void) {
    // Safety: Call the callback function with the provided value.
    let wrapper = IGoCallbackWrapper::new(cb);
    wrapper.callback(val);
}

pub(crate) static VM_ARENA: parking_lot::RwLock<Arena<mluau::Lua>> = parking_lot::RwLock::new(Arena::const_new());

#[derive(Clone, Copy)]
#[repr(transparent)]
pub struct VmHandle {
    pub handle: Handle,
}

impl VmHandle {
    pub fn new(lua: mluau::Lua) -> Self {
        let mut arena_lock = VM_ARENA.write();
        let handle = arena_lock.insert(lua);
        VmHandle { handle }
    }

    pub fn get(&self) -> mluau::Lua {
        let arena_lock = VM_ARENA.read();
        arena_lock.get(self.handle).cloned().expect("Invalid VM handle")
    }

    pub fn remove(&self) {
        let mut arena_lock = VM_ARENA.write();
        arena_lock.remove(self.handle);
    }
}