-- Amalgamated by lua-amalgamate
-- Entry: main

do
local _ENV = _ENV
package.preload["a"] = function(...)
  local name = ...
  package.loaded[name] = true
  local arg = _G.arg
local b = require("b")

local M = {}

function M.say_a()
    print("a")
    b.say_b()
end

return M
end
end
do
local _ENV = _ENV
package.preload["b"] = function(...)
  local name = ...
  package.loaded[name] = true
  local arg = _G.arg
local a = require("a")

local M = {}

function M.say_b()
    print("b")
end

return M
end
end
do
local _ENV = _ENV
package.preload["main"] = function(...)
  local name = ...
  package.loaded[name] = true
  local arg = _G.arg
local a = require("a")

print("circular test")
a.say_a()
end
end
return require("main")
