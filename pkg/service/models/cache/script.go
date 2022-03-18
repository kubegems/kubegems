package cache

const FindParentScript = `
local kind = KEYS[1]
local id = KEYS[2]
local ret = {}
local function getparents(kind, id)
    local key = kind.."_"..id
    local current = redis.call("HGET", "t", key)
    if not current then
        return
    end
    table.insert(ret, current)
    local cdata = cjson.decode(current)
    if cdata["owner"] then
        for k, parent in ipairs(cdata["owner"]) do
            getparents(parent["kind"], parent["id"])
        end 
    end
end

local function reverse(arr)
    local nret = {}
    for k, it in ipairs(arr) do
        nret[#arr-k+1] = it
    end 
    return nret
end

getparents(kind, id)
return reverse(ret)
`
