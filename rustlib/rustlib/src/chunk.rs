use crate::{compiler::CompilerOpts, result::{GoValueV2Result, wrap_failable}, table::get_table, value_v2::GoLuaValueV2};

// A ChunkString will be deallocated by Rust directly.
pub struct ChunkString {
    // Vec<u8> is used to store the data of the chunk string.
    pub data: Vec<u8>,
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_chunk_string_new(data: *const u8, len: usize) -> *mut ChunkString {
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
    pub env: GoLuaValueV2,
    // The chunks mode (either text or binary).
    pub mode: u8,
    // The compiler options for the chunk.
    pub compiler_opts: *mut CompilerOpts,
    // The actual code of the chunk
    pub code: *mut ChunkString,
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_load_chunk(ptr: *mut mluau::Lua, opts: ChunkOpts) -> GoValueV2Result {
    wrap_failable(|| {
        if ptr.is_null() || opts.code.is_null() {
            return GoValueV2Result::err("Lua pointer or ChunkOpts code is null".to_string());
        }

        let lua = unsafe { &*ptr };
        let code = unsafe { Box::from_raw(opts.code) };
        
        // Load the chunk with the provided options
        let mut chunk = lua.load(&code.data);
        if !opts.name.is_null() {
            let name = unsafe { Box::from_raw(opts.name) };
            chunk = chunk.set_name(String::from_utf8_lossy(&name.data));
        }

        if let Some(tab) = get_table(lua, opts.env) {
            chunk = chunk.set_environment(tab);
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
            Ok(f) => GoValueV2Result::ok(GoLuaValueV2::from_value(lua, mluau::Value::Function(f))),
            Err(err) => GoValueV2Result::err(format!("{err}"))
        }
    })
}