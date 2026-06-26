
-- Amalgamated by lua-amalgamate
-- Entry: main


do
local _ENV = _ENV
package.preload["./foo"] = function(...)
  local name = ...
  package.loaded[name] = true
  local arg = _G.arg
  local M = {}
  function M.say()
      return "foo"
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
  local foo = require("./foo")
  local bar = require("sub/bar")
  print("relative test")
  print(foo.say())
  print(bar.say())
end
end
do
local _ENV = _ENV
package.preload["sub/bar"] = function(...)
  local name = ...
  package.loaded[name] = true
  local arg = _G.arg
  local M = {}
  function M.say()
      return "bar"
  end
  return M
end
end
return require("main")
