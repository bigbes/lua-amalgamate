-- Amalgamated by lua-amalgamate
-- Entry: main

do
local _ENV = _ENV
package.preload["data"] = function(...)
  local name = ...
  package.loaded[name] = true
  local arg = _G.arg
return [[
alpha
  beta is indented two spaces
gamma
]]
end
end
do
local _ENV = _ENV
package.preload["main"] = function(...)
  local name = ...
  package.loaded[name] = true
  local arg = _G.arg
local s = require("data")
io.write(s)
end
end
return require("main")
