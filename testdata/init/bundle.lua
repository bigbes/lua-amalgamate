-- Amalgamated by amalg
-- Entry: main

package.preload["foo"] = function(...)
  local M = {}
  function M.greet()
      return "Hello from foo"
  end
  return M
end

package.preload["main"] = function(...)
  local foo = require("foo")
  print("init test")
  print(foo.greet())
end

require("main")
