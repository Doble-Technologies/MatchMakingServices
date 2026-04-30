package app

import "github.com/redis/go-redis/v9"

type App struct {
	Redis *redis.Client
}
