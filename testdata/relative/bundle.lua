-- Amalgamated by amalg
-- Entry: main

package.preload["./foo"] = function(...)
  local M = {}
  function M.say()
      return "foo"
  end
  return M
end

package.preload["main"] = function(...)
  local foo = require("./foo")
  local bar = require("sub/bar")
  print("relative test")
  print(foo.say())
  print(bar.say())
end

package.preload["sub/bar"] = function(...)
  local M = {}
  function M.say()
      return "bar"
  end
  return M
end

require("main")
