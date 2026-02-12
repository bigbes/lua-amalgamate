-- Amalgamated by amalg
-- Entry: main

package.preload["main"] = function(...)
  local utils = require("utils")
  print("Hello from main!")
  print("Utils says: " .. utils.add(5, 3)) 
end

package.preload["utils"] = function(...)
  local M = {}
  function M.add(a, b)
   return a + b
  end
  function M.subtract(a, b)
   return a - b
  end
  return M 
end

require("main")
