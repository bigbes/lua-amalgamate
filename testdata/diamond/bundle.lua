-- Amalgamated by lua-amalgamate
-- Entry: main

do
local _ENV = _ENV
package.preload["a"] = function(...)
  local name = ...
  package.loaded[name] = true
  local arg = _G.arg
local c = require("c")

return {
    value = c.value + 1
}
end
end
do
local _ENV = _ENV
package.preload["b"] = function(...)
  local name = ...
  package.loaded[name] = true
  local arg = _G.arg
local c = require("c")

return {
    value = c.value + 2
}
end
end
do
local _ENV = _ENV
package.preload["c"] = function(...)
  local name = ...
  package.loaded[name] = true
  local arg = _G.arg
return {
    value = 100
}
end
end
do
local _ENV = _ENV
package.preload["main"] = function(...)
  local name = ...
  package.loaded[name] = true
  local arg = _G.arg
local mod_a = require("a")
local mod_b = require("b")

print("Main: a.value = " .. mod_a.value)
print("Main: b.value = " .. mod_b.value)
end
end
return require("main")
