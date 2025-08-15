use crate::{compiler::CompilerOpts, result::GoFunctionResult, LuaVmWrapper};

// A ChunkString will be deallocated by Rust directly.
pub struct ChunkString {
    // Vec<u8> is used to store the data of the chunk string.
    pub data: Vec<u8>,
}

#[unsafe(no_mangle)]
pub extern "C-unwind" fn luago_chunk_string_new(data: *const u8, len: usize) -> *mut ChunkString {
    if data.is_null() || len == 0 {
        return std::ptr::null_mut();
    }

    // Safety: Convert the raw pointer to a Rust string slice
    let slice = unsafe { std::slice::from_raw_parts(data, len) };
    let chunk_string = ChunkString {
        data: slice.to_vec()
    };
    Box::into_raw(Box::new(chunk_string))
}

#[repr(C)]
pub struct ChunkOpts {
    // The name of the chunk, used for debugging and error messages.
    pub name: *mut ChunkString,
    // The environment table for the chunk.
    pub env: *mut mluau::Table,
    // The chunks mode (either text or binary).
    pub mode: u8,
    // The compiler options for the chunk.
    pub compiler_opts: *mut CompilerOpts,
    // The actual code of the chunk
    pub code: *mut ChunkString,
}

#[unsafe(no_mangle)]
pub extern "C-unwind" fn luago_load_chunk(ptr: *mut LuaVmWrapper, opts: ChunkOpts) -> GoFunctionResult {
    if ptr.is_null() || opts.code.is_null() {
        return GoFunctionResult::err("LuaVmWrapper pointer or ChunkOpts code is null".to_string());
    }

    let lua = unsafe { &(*ptr).lua };
    let code = unsafe { Box::from_raw(opts.code) };
    
    // Load the chunk with the provided options
    let mut chunk = lua.load(&code.data);
    if !opts.name.is_null() {
        let name = unsafe { Box::from_raw(opts.name) };
        chunk = chunk.set_name(String::from_utf8_lossy(&name.data));
    }

    if !opts.env.is_null() {
        let tab = unsafe { &*(opts.env) };
        chunk = chunk.set_environment(tab.clone());
    }

    chunk = match opts.mode {
        0 => chunk.set_mode(mluau::ChunkMode::Text),
        1 => chunk.set_mode(mluau::ChunkMode::Binary),
        _ => chunk.set_mode(mluau::ChunkMode::Text), // Default to text
    };

    if !opts.compiler_opts.is_null() {
        let compiler_opts = unsafe { &*(opts.compiler_opts) };
        chunk = chunk.set_compiler(compiler_opts.clone().to_compiler());
    }

    match chunk.into_function() {
        Ok(f) => GoFunctionResult::ok(Box::into_raw(Box::new(f))),
        Err(err) => GoFunctionResult::err(format!("{err}"))
    }
}