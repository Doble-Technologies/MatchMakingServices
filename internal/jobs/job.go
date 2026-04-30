package jobs

import (
	"context"

	"github.com/redis/go-redis/v9"
)

// Job interface defines the structure for any job that will be scheduled.
type Job interface {
	Name() string
	Schedule() string
	Run(ctx context.Context, rdb *redis.Client) error
}
