-- Amalgamated by amalg
-- Entry: main

package.preload["a"] = function(...)
  local b = require("b")
  
  local M = {}
  
  function M.say_a()
      print("a")
      b.say_b()
  end
  
  return M
end

package.preload["b"] = function(...)
  local a = require("a")
  
  local M = {}
  
  function M.say_b()
      print("b")
  end
  
  return M
end

package.preload["main"] = function(...)
  local a = require("a")
  
  print("circular test")
  a.say_a()
end

require("main")
