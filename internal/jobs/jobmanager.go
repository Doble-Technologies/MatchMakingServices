package jobs

import (
	"context"
	"log"

	"github.com/redis/go-redis/v9"
	"github.com/robfig/cron/v3"
)

// JobManager manages all scheduled jobs.
type JobManager struct {
	cron    *cron.Cron
	jobs    []Job
	context context.Context
	rdb     *redis.Client
}

// NewJobManager creates a new JobManager instance.
func NewJobManager(ctx context.Context, rdb *redis.Client) *JobManager {
	return &JobManager{
		cron:    cron.New(cron.WithSeconds()), // Enable seconds for fine-grained scheduling
		jobs:    []Job{},
		context: ctx,
		rdb:     rdb,
	}
}

// RegisterJob adds a job to the job manager.
func (jm *JobManager) RegisterJob(job Job) {
	jm.jobs = append(jm.jobs, job)
}

// StartScheduler starts the cron scheduler to execute jobs at their scheduled times.
func (jm *JobManager) StartScheduler() {
	for _, job := range jm.jobs {
		schedule := job.Schedule()
		if _, err := jm.cron.AddFunc(schedule, func() {
			if err := job.Run(jm.context, jm.rdb); err != nil {
				log.Printf("Error in job %s: %v", job.Name(), err)
			} else {
				log.Printf("Job %s executed successfully", job.Name())
			}
		}); err != nil {
			log.Printf("Failed to schedule job %s: %v", job.Name(), err)
		}
	}
	jm.cron.Start()
}
