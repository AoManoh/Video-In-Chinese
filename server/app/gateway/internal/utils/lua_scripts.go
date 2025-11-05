package utils

// UpdateSettingsScript is a Lua script for atomically updating Redis settings with optimistic locking
const UpdateSettingsScript = `
-- KEYS[1]: Redis key (app:settings)
-- ARGV[1]: Expected version number
-- ARGV[2..N]: Field-value pairs (field1, value1, field2, value2, ...)

local key = KEYS[1]
local expectedVersion = tonumber(ARGV[1])

-- Read current version
local currentVersion = redis.call('HGET', key, 'version')
if currentVersion == false then
    currentVersion = 0
else
    currentVersion = tonumber(currentVersion)
end

-- Check version (optimistic lock)
if currentVersion ~= expectedVersion then
    return {-1, currentVersion}  -- Return error code and current version
end

-- Update all fields
local numFields = (#ARGV - 1) / 2
for i = 1, numFields do
    local field = ARGV[i * 2]
    local value = ARGV[i * 2 + 1]
    redis.call('HSET', key, field, value)
end

-- Increment version
local newVersion = currentVersion + 1
redis.call('HSET', key, 'version', newVersion)

return {0, newVersion}  -- Return success code and new version
`

