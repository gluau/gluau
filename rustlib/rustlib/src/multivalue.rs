use mluau::MultiValue;

use crate::value::GoLuaValue;
use std::sync::RwLock;

// A list of mlua Value's, used for passing multiple values from Rust to Go.
//
// This is a opaque structure
pub struct GoMultiValue {
    pub values: RwLock<mluau::MultiValue>,
}

impl GoMultiValue {
    // Instantiates a new GoMultiValue allocated on the heap.
    pub fn new() -> *mut Self {
        Box::into_raw(Box::new(GoMultiValue { values: RwLock::new(MultiValue::new()) }))
    }

    // Instantiates a new GoMultiValue allocated on the heap to be sent to the Go side.
    //
    // It is the responsibility of the Go side to free this memory.
    pub fn inst(values: MultiValue) -> *mut Self {
        Box::into_raw(Box::new(GoMultiValue { values: RwLock::new(values) }))
    }
}

// Creates an empty LuaValueVec.
#[unsafe(no_mangle)]
pub extern "C-unwind" fn luago_create_multivalue_with_capacity(cap: usize) -> *mut GoMultiValue {
    let vec = GoMultiValue { values: RwLock::new(MultiValue::with_capacity(cap)) };
    Box::into_raw(Box::new(vec))
}

// Adds a GoLuaValue to the LuaValueVec, taking ownership of the passed value.
//
// Useful for passing multiple values from Go to Rust.
#[unsafe(no_mangle)]
pub extern "C-unwind" fn luago_multivalue_push(vec_ptr: *mut GoMultiValue, value: GoLuaValue) {
    // Safety: Assume vec_ptr is a valid, non-null pointer to a LuaValueVec
    if vec_ptr.is_null() {
        panic!("LuaValueVec pointer is null");
    }
    let vec = unsafe { &*vec_ptr };
    // SAFETY: the Go side may not limit concurrent access to this vector,
    // so we use a RwLock to ensure safe concurrent access.
    let mut values = vec.values.write().unwrap();
    values.push_back(value.to_value_from_owned());
}

// Returns the number of values in the LuaValueVec.
#[unsafe(no_mangle)] 
pub extern "C-unwind" fn luago_multivalue_len(vec_ptr: *mut GoMultiValue) -> usize {
    // Safety: Assume vec_ptr is a valid, non-null pointer to a LuaValueVec
    if vec_ptr.is_null() {
        panic!("LuaValueVec pointer is null");
    }
    let vec = unsafe { &*vec_ptr };
    // SAFETY: the Go side may not limit concurrent access to this vector,
    // so we use a RwLock to ensure safe concurrent access.
    let values = vec.values.read().unwrap();
    values.len()
}

// Pops the front value from the GoMultiValue and returns it.
#[unsafe(no_mangle)]
pub extern "C-unwind" fn luago_multivalue_pop(vec_ptr: *mut GoMultiValue) -> GoLuaValue {
    // Safety: Assume vec_ptr is a valid, non-null pointer to a GoMultiValue
    if vec_ptr.is_null() {
        panic!("GoMultiValue pointer is null");
    }
    let vec = unsafe { &*vec_ptr };
    // SAFETY: the Go side may not limit concurrent access to this vector,
    // so we use a RwLock to ensure safe concurrent access.
    let mut values = vec.values.write().unwrap();
    if let Some(value) = values.pop_front() {
        GoLuaValue::from_owned(value)
    } else {
        GoLuaValue::from_owned(mluau::Value::Nil) // Return Nil if the vector is empty.
    }
}

// Frees the memory for a GoMultiValue.
#[unsafe(no_mangle)]
pub extern "C-unwind" fn luago_free_multivalue(vec_ptr: *mut GoMultiValue) {
    // Safety: Assume vec_ptr is a valid, non-null pointer to a GoMultiValue
    if vec_ptr.is_null() {
        return;
    }
    // Re-box the GoMultiValue pointer to manage its memory automatically.
    unsafe { drop(Box::from_raw(vec_ptr)) };
}