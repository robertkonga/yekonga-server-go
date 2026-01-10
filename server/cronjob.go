package Yekonga

import (
	"time"
)

func NewCronjob(app *YekongaData) *Cronjob {
	c := Cronjob{
		app:  app,
		jobs: make([]Job, 0),
	}

	c.initialize()

	return &c
}

type Job struct {
	name     string
	delay    time.Duration
	time     time.Duration
	callback func(app *YekongaData, time time.Time)
	process  bool
	lastRun  time.Time
}

type Cronjob struct {
	app  *YekongaData
	jobs []Job
}

func (c *Cronjob) initialize() {
	go func() {
		ticker := time.NewTicker(time.Second * 1)
		defer ticker.Stop()

		for {
			t := <-ticker.C
			// console.Error("Count", t.String())
			for i, job := range c.jobs {
				c.jobs[i].time += time.Duration(1) * time.Second

				if job.time >= job.delay && !job.process {
					c.jobs[i].lastRun = t
					c.jobs[i].time = time.Duration(0)
					c.jobs[i].process = true

					go func() {
						job.callback(c.app, t)
						c.jobs[i].process = false
					}()

					// console.Info("Info", job.name, job.delay, job.time)
				}
			}
		}
	}()
}

func (c *Cronjob) registerJob(name string, delay time.Duration, callback func(app *YekongaData, time time.Time)) {
	c.jobs = append(c.jobs, Job{
		name:     name,
		delay:    delay,
		time:     0,
		callback: callback,
		process:  false,
	})
}
