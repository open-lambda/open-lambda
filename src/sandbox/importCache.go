package sandbox

import (
	"github.com/open-lambda/open-lambda/ol/config"
	"log"
	"path/filepath"
)

type ImportCache struct {
	name     string
	pool     SandboxPool
	requests chan *ParentReq
	events   chan SandboxEvent
	killChan chan chan bool
}

type ParentReq struct {
	imports []string
	parent  chan Sandbox
}

func NewImportCache(name string, sizeMb int) (*ImportCache, error) {
	pool, err := SandboxPoolFromConfig(name, sizeMb)
	if err != nil {
		return nil, err
	}

	cache := &ImportCache{
		name:     name,
		pool:     pool,
		requests: make(chan *ParentReq, 32),
		events:   make(chan SandboxEvent, 32),
		killChan: make(chan chan bool),
	}

	go cache.Run(pool)

	return cache, nil
}

func (cache *ImportCache) GetParent(imports []string) Sandbox {
	parent := make(chan Sandbox)
	cache.requests <- &ParentReq{imports, parent}
	return <-parent
}

func (cache *ImportCache) Cleanup() {
	done := make(chan bool)
	cache.killChan <- done
	<-done
	cache.pool.Cleanup()
}

func (cache *ImportCache) Event(evType SandboxEventType, sb Sandbox) {
	if evType == evDestroy {
		cache.events <- SandboxEvent{evType, sb}
	}
}

func (cache *ImportCache) create(parent Sandbox, imports []string) Sandbox {
	scratchPrefix := filepath.Join(config.Conf.Worker_dir, cache.name+"-scratch")
	sb, err := cache.pool.Create(parent, false, "", scratchPrefix, imports)
	if err != nil {
		log.Printf("import cache failed to create from '%v' with imports '%s'", parent, imports)
		return nil
	}
	return sb
}

func (cache *ImportCache) Run(pool SandboxPool) {
	forkServers := make(map[string]Sandbox)
	root := cache.create(nil, []string{})
	if root == nil {
		panic("could not even create a root Zygote")
	}
	forkServers[root.ID()] = root

	for {
		select {
		case req := <-cache.requests:
			// POLICY: which parent should we return?

			// TODO: create (and use) more Zygotes
			req.parent <- root
		case event := <-cache.events:
			switch event.evType {
			case evDestroy:
				delete(forkServers, event.sb.ID())
			}
		case done := <-cache.killChan:
			for _, sb := range forkServers {
				sb.Destroy()
			}
			done <- true
			return
		}
	}
}
