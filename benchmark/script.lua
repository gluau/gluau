-- Benchmark script for comparing Lua implementations
-- Run with: luau script.lua / luajit script.lua / lua script.lua
--
-- Results (Alpine Linux, x86_64):
--
--   | Benchmark                    | Luau   | LuaJIT | Lua 5.x |
--   |------------------------------|--------|--------|---------|
--   | Fibonacci(30)                | 95ms   | 15ms   | 146ms   |
--   | Loop (10M iter)              | 110ms  | 33ms   | 198ms   |
--   | String naive (s=s.."x") 20K  | 219ms  | 29ms   | 125ms   |
--   | String table.concat 20K      | 1ms    | <1ms   | 2ms     |
--   | Table ops (100K)             | 4ms    | 1ms    | 8ms     |
--
-- LuaJIT's JIT compiler makes it 3-7x faster than Luau on most benchmarks.
-- Luau sits between LuaJIT and Lua 5.x for compute tasks.

local clock = os.clock

local function measure(name, fn)
    local start = clock()
    local result = fn()
    local elapsed = clock() - start
    print(string.format("%-35s %8.3fs  (result: %s)", name, elapsed, tostring(result)))
end

print("=== Lua Benchmark ===")
print()

-- Fibonacci (recursive)
measure("Fibonacci(30) recursive", function()
    local function fib(n)
        if n < 2 then return n end
        return fib(n-1) + fib(n-2)
    end
    return fib(30)
end)

-- Loop with arithmetic
measure("Loop with arithmetic (10M iter)", function()
    local sum = 0
    for i = 1, 10000000 do
        sum = sum + i * 2 - 1
    end
    return sum
end)

-- String naive concatenation
measure("String naive s=s..\"x\" (20K iter)", function()
    local s = ""
    for i = 1, 20000 do
        s = s .. "x"
    end
    return #s
end)

-- String table.concat
measure("String table.concat (20K iter)", function()
    local t = {}
    for i = 1, 20000 do
        t[i] = "x"
    end
    return #table.concat(t)
end)

-- Table insert/access
measure("Table insert/access (100K ops)", function()
    local t = {}
    for i = 1, 100000 do
        t[i] = i * 2
    end
    local sum = 0
    for i = 1, 100000 do
        sum = sum + t[i]
    end
    return sum
end)

print()
print("Done.")
