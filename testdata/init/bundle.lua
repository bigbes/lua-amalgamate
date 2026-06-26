-- Amalgamated by lua-amalgamate
-- Entry: main

do
local _ENV = _ENV
package.preload["foo"] = function(...)
  local name = ...
  package.loaded[name] = true
  local arg = _G.arg
  local M = {}
  function M.greet()
      return "Hello from foo"
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
  local foo = require("foo")
  print("init test")
  print(foo.greet())
end
end
return require("main")
