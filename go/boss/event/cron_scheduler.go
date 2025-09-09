package event

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"sync"

	"github.com/open-lambda/open-lambda/go/boss/cloudvm"
	"github.com/open-lambda/open-lambda/go/common"
	"github.com/robfig/cron/v3"
)

type CronScheduler struct {
	cron       *cron.Cron
	mapLock    sync.Mutex                // protects jobs map
	jobs       map[string][]cron.EntryID // functionName -> list of job IDs
	workerPool *cloudvm.WorkerPool       // to forward the req to worker
}

func NewCronScheduler(pool *cloudvm.WorkerPool) *CronScheduler {
	c := &CronScheduler{
		cron:       cron.New(),
		jobs:       make(map[string][]cron.EntryID),
		workerPool: pool,
	}
	c.cron.Start()
	return c
}

func (c *CronScheduler) Register(functionName string, triggers []common.CronTrigger) error {
	if len(triggers) == 0 {
		return nil
	}

	c.mapLock.Lock()
	defer c.mapLock.Unlock()

	for _, trigger := range triggers {
		funcName := functionName
		schedule := trigger.Schedule

		entryID, err := c.cron.AddFunc(schedule, func() {
			c.Invoke(funcName)
		})
		if err != nil {
			return fmt.Errorf("[CronScheduler] Failed to add cron job for %s: %v", funcName, err)
		}
		c.jobs[funcName] = append(c.jobs[funcName], entryID)
	}

	return nil
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

func (c *CronScheduler) Invoke(functionName string) {
	// Simulate HTTP request to /run/<function>
	req := httptest.NewRequest(http.MethodPost, "/run/"+functionName, bytes.NewBuffer([]byte(`{}`)))
	w := httptest.NewRecorder()

	c.workerPool.RunLambda(w, req)

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		// TODO: Improve how this error is surfaced. The platform operator can see logs,
		// but function developers likely cannot — consider exposing errors through a user-facing mechanism.
		log.Printf("[CronScheduler] Lambda %s returned non-200 (%d): %s", functionName, resp.StatusCode, string(body))
	} else {
		log.Printf("[CronScheduler] Lambda %s invoked successfully. Response: %s", functionName, string(body))
	}
}
