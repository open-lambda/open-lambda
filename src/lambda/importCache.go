package lambda

import (
	"log"

	"github.com/open-lambda/open-lambda/ol/sandbox"
)

type ImportCache struct {
	name     string
	pool     sandbox.SandboxPool
	requests chan *ParentReq
	events   chan sandbox.SandboxEvent
	killChan chan chan bool
}

type ParentReq struct {
	deps   *sandbox.Dependencies
	parent chan sandbox.Sandbox
}

func NewImportCache(name string, sizeMb int) (*ImportCache, error) {
	pool, err := sandbox.SandboxPoolFromConfig(name, sizeMb)
	if err != nil {
		return nil, err
	}

	cache := &ImportCache{
		name:     name,
		pool:     pool,
		requests: make(chan *ParentReq, 32),
		events:   make(chan sandbox.SandboxEvent, 32),
		killChan: make(chan chan bool),
	}

	go cache.run(pool)

	return cache, nil
}

func (cache *ImportCache) GetParent(deps *sandbox.Dependencies) sandbox.Sandbox {
	parent := make(chan sandbox.Sandbox)
	cache.requests <- &ParentReq{deps, parent}
	return <-parent
}

func (cache *ImportCache) Event(evType sandbox.SandboxEventType, sb sandbox.Sandbox) {
	if evType == sandbox.EvDestroy {
		cache.events <- sandbox.SandboxEvent{evType, sb}
	}
}

func (cache *ImportCache) Cleanup() {
	done := make(chan bool)
	cache.killChan <- done
	<-done
	cache.pool.Cleanup()
}

func (cache *ImportCache) create(parent sandbox.Sandbox, deps *sandbox.Dependencies) sandbox.Sandbox {
	sb, err := cache.pool.Create(parent, false, "", mkScratchDir("import-cache"), deps)
	if err != nil {
		log.Printf("import cache failed to create from '%v' with imports '%s'", parent, deps.Imports)
		return nil
	}
	return sb
}

func (cache *ImportCache) run(pool sandbox.SandboxPool) {
	forkServers := make(map[string]sandbox.Sandbox)
	root := cache.create(nil, nil)
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
			switch event.EvType {
			case sandbox.EvDestroy:
				delete(forkServers, event.SB.ID())
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
