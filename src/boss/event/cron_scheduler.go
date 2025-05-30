package event

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"sync"

	"github.com/open-lambda/open-lambda/ol/common"
	"github.com/robfig/cron/v3"
)

type CronScheduler struct {
	cron    *cron.Cron
	mapLock sync.Mutex                // protects jobs map
	jobs    map[string][]cron.EntryID // functionName -> list of job IDs
}

func NewCronScheduler() *CronScheduler {
	c := &CronScheduler{
		cron: cron.New(),
		jobs: make(map[string][]cron.EntryID),
	}
	c.cron.Start()
	return c
}

func (c *CronScheduler) Register(functionName string, triggers []common.CronTrigger) {
	c.mapLock.Lock()
	defer c.mapLock.Unlock()

	for _, trigger := range triggers {
		// save functionName for job context
		funcName := functionName
		schedule := trigger.Schedule

		entryID, err := c.cron.AddFunc(schedule, func() {
			c.Invoke(functionName)
		})
		if err == nil {
			c.jobs[funcName] = append(c.jobs[funcName], entryID)
		} else {
			println("[CronScheduler] Failed to add cron job for", funcName, ":", err.Error())
		}
	}
}

func (c *CronScheduler) Unregister(functionName string) {
	c.mapLock.Lock()
	defer c.mapLock.Unlock()

	ids, ok := c.jobs[functionName]
	if !ok {
		return
	}

	for _, id := range ids {
		c.cron.Remove(id)
	}

	delete(c.jobs, functionName)
}

func (c *CronScheduler) Stop() {
	c.cron.Stop()
}

func (c *CronScheduler) Invoke(functionName string) {
	url := "http://localhost:5000/run/" + functionName
	resp, err := http.Post(url, "application/json", bytes.NewBuffer([]byte(`{}`))) // empty JSON payload
	if err != nil {
		log.Printf("failed to invoke lambda %s: %v", functionName, err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[CronScheduler] Failed to read response body for lambda %s: %v", functionName, err)
		return
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("[CronScheduler] Lambda %s returned non-200 (%d): %s", functionName, resp.StatusCode, string(body))
	} else {
		log.Printf("[CronScheduler] Lambda %s invoked successfully. Response: %s", functionName, string(body))
	}
}
