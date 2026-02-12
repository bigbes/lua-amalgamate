-- Amalgamated by amalg
-- Entry: main

package.preload["main"] = function(...)
  local modname = "module"
  local mod = require(modname)
  print("dynamic test")
end

require("main")
