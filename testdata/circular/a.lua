local b = require("b")

local M = {}

function M.say_a()
    print("a")
    b.say_b()
end

return M