package errx

import (
	"errors"
	"net/http"

	"github.com/redis/go-redis/v9"
)

// WrapRedis maps Redis errors to the unified Error type with appropriate status codes.
func WrapRedis(err error) *Error {
	if err == nil {
		return nil
	}

	if errors.Is(err, redis.Nil) {
		return New(err, http.StatusNotFound, RedisNotFoundMessage)
	}

	return New(err, http.StatusBadGateway, RedisErrorMessage)
}
