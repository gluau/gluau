#[repr(C)]
#[derive(Clone)]
pub struct CompilerOpts {
    // The optimization level for the Lua chunk.
    pub optimization_level: u8,
    // The debug level for the Lua chunk.
    pub debug_level: u8,
    // The Luau type information level
    pub type_info_level: u8,
    // The coverage level to use
    pub coverage_level: u8,
}

impl CompilerOpts {
    pub fn to_compiler(self) -> mluau::Compiler {
        let mut compiler = mluau::Compiler::new();
        compiler = compiler.set_optimization_level(self.optimization_level);
        compiler = compiler.set_debug_level(self.debug_level);
        compiler = compiler.set_type_info_level(self.type_info_level);
        compiler = compiler.set_coverage_level(self.coverage_level);
        compiler
    }
}