use mluau::{NavigateError, Require};

use crate::{IGoCallback, IGoCallbackWrapper, result::{GoValueV2Result, to_c_string_from_ref, wrap_failable}, value_v2::GoLuaValueV2};
use std::ffi::{c_char, c_void, CString};

#[repr(C)]
pub struct GoNavigationResult {
    not_found: bool,
    ambiguous: bool,
    other: *mut c_char // Rust will deallocate this automatically. Should be allocated with moveStringToRust
}

impl GoNavigationResult {
    fn new() -> Self {
        Self {
            not_found: false,
            ambiguous: false,
            other: std::ptr::null_mut()
        }
    }

    // Converts a GoNavigationResult into a NavigateError
    fn to_result(self) -> Result<(), NavigateError> {
        if self.not_found {
            Err(NavigateError::NotFound)
        } else if self.ambiguous {
            Err(NavigateError::Ambiguous)
        } else if !self.other.is_null() {
            let error = unsafe { CString::from_raw(self.other) };
            Err(NavigateError::Other(mluau::Error::external(error.to_string_lossy())))
        } else {
            Ok(())
        }
    }
}

#[repr(C)]
pub struct GoRequire {
    pub is_require_allowed: IGoCallback,
    pub reset: IGoCallback,
    pub jump_to_alias: IGoCallback,
    pub to_parent: IGoCallback,
    pub to_child: IGoCallback,
    pub has_module: IGoCallback,
    pub cache_key: IGoCallback,
    pub has_config: IGoCallback,
    pub config: IGoCallback,
    pub loader: IGoCallback,
}

impl GoRequire {
    fn to_impl(self) -> GoRequireImpl {
        GoRequireImpl {
            is_require_allowed: IGoCallbackWrapper::new(self.is_require_allowed),
            reset: IGoCallbackWrapper::new(self.reset),
            jump_to_alias: IGoCallbackWrapper::new(self.jump_to_alias),
            to_parent: IGoCallbackWrapper::new(self.to_parent),
            to_child: IGoCallbackWrapper::new(self.to_child),
            has_module: IGoCallbackWrapper::new(self.has_module),
            cache_key: IGoCallbackWrapper::new(self.cache_key),
            has_config: IGoCallbackWrapper::new(self.has_config),
            config: IGoCallbackWrapper::new(self.config),
            loader: IGoCallbackWrapper::new(self.loader),
        }
    }
}

pub struct GoRequireImpl {
    pub is_require_allowed: IGoCallbackWrapper,
    pub reset: IGoCallbackWrapper,
    pub jump_to_alias: IGoCallbackWrapper,
    pub to_parent: IGoCallbackWrapper,
    pub to_child: IGoCallbackWrapper,
    pub has_module: IGoCallbackWrapper,
    pub cache_key: IGoCallbackWrapper,
    pub has_config: IGoCallbackWrapper,
    pub config: IGoCallbackWrapper,
    pub loader: IGoCallbackWrapper,
}

impl GoRequireImpl {
    /// Calls the callback provided by the Go side with data and returns the filled in data
    fn fill<R>(&self, cb_wrapper: &IGoCallbackWrapper, data: R) -> R {
        let ptr = Box::into_raw(Box::new(data));
        cb_wrapper.callback(ptr as *mut c_void);
        let data = unsafe { Box::from_raw(ptr) };
        *data
    }
}

impl Require for GoRequireImpl {
    fn is_require_allowed(&self, chunk_name: &str) -> bool {
        let data = self.fill(&self.is_require_allowed, IsRequireAllowed {
            chunk_name: to_c_string_from_ref(chunk_name),
            data: false,
        });

        data.data
    }

    fn reset(&mut self, chunk_name: &str) -> Result<(), NavigateError> {
        let data = self.fill(&self.reset, ResetOrJumpToAliasOrToChild {
            str: to_c_string_from_ref(chunk_name),
            data: GoNavigationResult::new(),
        });

        data.data.to_result()
    }

    fn jump_to_alias(&mut self, path: &str) -> Result<(), NavigateError> {
        let data = self.fill(&self.jump_to_alias, ResetOrJumpToAliasOrToChild {
            str: to_c_string_from_ref(path),
            data: GoNavigationResult::new(),
        });

        data.data.to_result()
    }

    fn to_parent(&mut self) -> Result<(), NavigateError> {
        let data = self.fill(&self.to_parent, ToParent {
            data: GoNavigationResult::new(),
        });

        data.data.to_result()
    }

    fn to_child(&mut self, name: &str) -> Result<(), NavigateError> {
        let data = self.fill(&self.to_child, ResetOrJumpToAliasOrToChild {
            str: to_c_string_from_ref(name),
            data: GoNavigationResult::new(),
        });

        data.data.to_result()
    }

    fn has_module(&self) -> bool {
        let data = self.fill(&self.has_module, HasModuleOrHasConfig {
            data: false
        });

        data.data
    }

    fn cache_key(&self) -> String {
        let data = self.fill(&self.cache_key, CacheKey {
            data: std::ptr::null_mut()
        });

        assert!(!data.data.is_null());

        let key = unsafe { CString::from_raw(data.data) };
        key.to_string_lossy().to_string()
    }

    fn has_config(&self) -> bool {
        let data = self.fill(&self.has_config, HasModuleOrHasConfig {
            data: false
        });

        data.data
    }

    fn config(&self) -> std::io::Result<Vec<u8>> {
        let data = self.fill(&self.config, Config {
            data: std::ptr::null_mut(),
            error: std::ptr::null_mut()
        });

        if !data.error.is_null() {
            if !data.data.is_null() {
                // Avoid a memory leak by deallocating it
                unsafe { drop(CString::from_raw(data.data)) };
            }
            let error = unsafe { CString::from_raw(data.error) };
            return Err(std::io::Error::other(error.to_string_lossy()));
        }

        assert!(!data.data.is_null());

        let key = unsafe { CString::from_raw(data.data) };
        Ok(key.as_bytes().to_vec())
    }

    fn loader(&self, lua: &mluau::Lua) -> mluau::Result<mluau::Function> {
        let wrapper = Box::new(lua.clone());
        let lua_ptr = Box::into_raw(wrapper);

        let data = self.fill(&self.loader, Loader {
            lua: lua_ptr,
            error: std::ptr::null_mut(),
            function: std::ptr::null_mut(),
        });

        if !data.error.is_null() {
            assert!(data.function.is_null());
            let error = unsafe { CString::from_raw(data.error) };
            return Err(mluau::Error::external(error.to_string_lossy()))
        }

        assert!(!data.function.is_null());
        // Safety: Go side must ensure values cannot be used after it is set
        // here as a return value
        let func = unsafe { Box::from_raw(data.function) };
        Ok(*func)
    }
}

#[repr(C)]
pub struct IsRequireAllowed {
    pub chunk_name: *mut c_char, // Go will free this automatically with the moveStringToGo function

    // Go may set this to true if the require is allowed
    pub data: bool,
}

#[repr(C)]
pub struct ResetOrJumpToAliasOrToChild {
    pub str: *mut c_char, // Go will free this automatically with the moveStringToGo function

    // Go may set this to true if the require is allowed
    pub data: GoNavigationResult,
}

#[repr(C)]
pub struct ToParent {
    // Go may set this to true if the require is allowed
    pub data: GoNavigationResult,
}

#[repr(C)]
pub struct HasModuleOrHasConfig {
    // Go may set this to true if the require is allowed
    pub data: bool,
}

#[repr(C)]
pub struct CacheKey {
    pub data: *mut c_char // Rust will deallocate this automatically. Should be allocated with moveStringToRust
}

#[repr(C)]
pub struct Config {
    pub data: *mut c_char, // Rust will deallocate this automatically. Should be allocated with moveStringToRust
    pub error: *mut c_char,
}

#[repr(C)]
pub struct Loader {
    pub lua: *mut mluau::Lua,
    
    // Go side may set this in response
    pub function: *mut mluau::Function,
    pub error: *mut c_char,
}

#[unsafe(no_mangle)]
pub extern "C" fn luago_create_require_function(ptr: *mut mluau::Lua, gr: GoRequire) -> GoValueV2Result  {
    wrap_failable(|| {
        // Safety: Assume ptr is a valid, non-null pointer to a Lua
        if ptr.is_null() {
            return GoValueV2Result::err("Lua pointer is null".to_string());
        }

        let lua = unsafe { &*ptr };
        let func = lua.create_require_function(gr.to_impl());

        match func {
            Ok(f) => GoValueV2Result::ok(GoLuaValueV2::from_value(lua, mluau::Value::Function(f))),
            Err(err) => GoValueV2Result::err(format!("{err}")),
        }
    })
}