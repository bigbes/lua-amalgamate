-- Utility module
local M = {}

-- Add two numbers
-- @param a first number
-- @param b second number
-- @return sum of a and b
function M.add(a, b)
    -- Perform addition
    return a + b  -- return result
end

--[[
This is a block comment
that spans multiple lines
]]
function M.subtract(a, b)
    return a - b
end

--[==[
Another block comment with equals
]==]

return M  -- return module table