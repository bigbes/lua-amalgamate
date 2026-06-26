
-- Amalgamated by lua-amalgamate
-- Entry: main


do
local _ENV = _ENV
package.preload["main"] = function(...)
  local name = ...
  package.loaded[name] = true
  local arg = _G.arg
  local mod = require("module")
  
  print("Hello from main!")
  print("Module says: " .. mod.greet())
end
end
do
local _ENV = _ENV
package.preload["module"] = function(...)
  local name = ...
  package.loaded[name] = true
  local arg = _G.arg
  local M = {}
  
  function M.greet()
      return "Hello from module!"
  end
  
  return M
end
end
return require("main")
