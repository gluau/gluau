# gluau

gluau provides Go bindings for the Luau programming language

**Heavy WIP**: This library is currently in development and not yet ready for production use. The API may change frequently, and there are many features that are not yet implemented.

## Implemented APIs so far

- VM initialization and shutdown
- Basic Lua value API to abstract over Lua values via Go interfaces
- Lua Strings (along with API's)
- Lua Tables (API's are WIP)

## Benefits over other libraries

### Exception Handling Support

Unlike prior attempts at this such as [golua](https://github.com/aarzilli/golua), gluau has full support for Luau exception handling. This means that you can use Luau's `pcall` and `xpcall` functions to handle errors in your Lua code, and they will work seamlessly with Go's error handling.

gluau achieves this feat by using a Rust proxy layer to actually manage the Lua VM, which allows it to handle exceptions in a way that is compatible with Go's error handling.