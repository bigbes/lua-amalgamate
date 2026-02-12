-- Amalgamated by amalg
-- Entry: main

package.preload["a"] = function(...)
  local c = require("c")
  
  return {
      value = c.value + 1
  }
end

package.preload["b"] = function(...)
  local c = require("c")
  
  return {
      value = c.value + 2
  }
end

do
  local __loader = function(...)
  return {
      value = 100
  }
  end
  package.preload["c"] = __loader
  package.preload["c"] = __loader
end

package.preload["main"] = function(...)
  local mod_a = require("a")
  local mod_b = require("b")
  
  print("Main: a.value = " .. mod_a.value)
  print("Main: b.value = " .. mod_b.value)
end

require("main")
