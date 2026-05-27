package rate_limit

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type Limiter struct {
	redis *redis.Client
}

func New(redisClient *redis.Client) *Limiter {
	return &Limiter{redis: redisClient}
}

var allowScript = redis.NewScript(`
local current = redis.call("INCR", KEYS[1])
if current == 1 then
  redis.call("EXPIRE", KEYS[1], ARGV[1])
end
if current > tonumber(ARGV[2]) then
  return 0
end
return 1
`)

// Allow returns true if the key is allowed to make the request.
func (l *Limiter) Allow(ctx context.Context, key string, window time.Duration, limit int) (bool, error) {
	secs := int(window.Seconds())
	res, err := allowScript.Run(ctx, l.redis, []string{key}, secs, limit).Int()
	if err != nil {
		return false, err
	}
	return res == 1, nil
}
