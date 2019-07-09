package lambda

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	"github.com/open-lambda/open-lambda/ol/config"
	"github.com/open-lambda/open-lambda/ol/sandbox"
)

type ImportCache struct {
	name     string
	pool     sandbox.SandboxPool
	root     *ImportCacheNode
	requests chan *ParentReq
	events   chan sandbox.SandboxEvent
	killChan chan chan bool
}

type ParentReq struct {
	meta   *sandbox.SandboxMeta
	parent chan sandbox.Sandbox
}

type ImportCacheNode struct {
	Packages         []string           `json:"packages""`
	Children         []*ImportCacheNode `json:"children"`
	parent           *ImportCacheNode
	indirectPackages []string
	sb               sandbox.Sandbox
}

func NewImportCache(name string, sizeMb int) (*ImportCache, error) {
	cache := &ImportCache{
		name:     name,
		requests: make(chan *ParentReq, 32),
		events:   make(chan sandbox.SandboxEvent, 32),
		killChan: make(chan chan bool),
	}

	// a static tree of Zygotes may be specified by a file (if so, parse and init it)
	path := config.Conf.Import_cache_tree_path
	if path != "" {
		b, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("could not open import tree file (%v): %v\n", path, err.Error())
		}

		cache.root = &ImportCacheNode{}
		if err := json.Unmarshal(b, cache.root); err != nil {
			return nil, fmt.Errorf("could parse import tree file (%v): %v\n", path, err.Error())
		}
		if len(cache.root.Packages) > 0 {
			return nil, fmt.Errorf("root node in import cache may not import packages\n")
		}
		cache.root.recursiveInit([]string{})
		log.Printf("Import Cache Tree:")
		cache.root.Dump(0)
	}

	// import cache gets its own sandbox pool
	pool, err := sandbox.SandboxPoolFromConfig(name, sizeMb)
	if err != nil {
		return nil, err
	}
	pool.AddListener(cache.Event)
	cache.pool = pool

	// start background task to serve requests for Zygotes
	go cache.run(pool)
	return cache, nil
}

func (cache *ImportCache) GetParent(meta *sandbox.SandboxMeta) sandbox.Sandbox {
	parent := make(chan sandbox.Sandbox)
	cache.requests <- &ParentReq{meta, parent}
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

func (cache *ImportCache) create(parent sandbox.Sandbox, meta *sandbox.SandboxMeta) sandbox.Sandbox {
	sb, err := cache.pool.Create(parent, false, "", mkScratchDir("import-cache"), meta)
	if err != nil {
		log.Printf("import cache failed to create from '%v' with imports '%s'", parent, meta.Imports)
		return nil
	}
	return sb
}

func (cache *ImportCache) run(pool sandbox.SandboxPool) {
	forkServers := make(map[string]sandbox.Sandbox)
	var root sandbox.Sandbox = nil

	for {
		select {
		case req := <-cache.requests:
			// POLICY: which parent should we return?

			// TODO: create (and use) more Zygotes
			if root == nil {
				root = cache.create(nil, nil)
				if root != nil {
					forkServers[root.ID()] = root
				}
			}
			req.parent <- root
		case event := <-cache.events:
			switch event.EvType {
			case sandbox.EvDestroy:
				log.Printf("Sandbox %v in import cache has been destroyed", event.SB.ID())
				if event.SB.ID() == root.ID() {
					root = nil
				}
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

// 1. populate parent field of every struct
// 2. populate indirectPackages to contain the packages of every ancestor
func (node *ImportCacheNode) recursiveInit(indirectPackages []string) {
	node.indirectPackages = indirectPackages
	for _, child := range node.Children {
		child.parent = node
		child.recursiveInit(append(indirectPackages, node.Packages...))
	}
}

func (node *ImportCacheNode) String() string {
	s := strings.Join(node.Packages, ",")
	if s == "" {
		s = "ROOT"
	}
	if len(node.indirectPackages) > 0 {
		s += " [indirect: " + strings.Join(node.indirectPackages, ",") + "]"
	}
	return s
}

func (node *ImportCacheNode) Dump(indent int) {
	spaces := strings.Repeat("  ", indent)

	log.Printf("%s - %s", spaces, node.String())
	for _, child := range node.Children {
		child.Dump(indent + 1)
	}
}

func (node *ImportCacheNode) Lookup(packages []string) *ImportCacheNode {
	// if this node imports a package that's not wanted by the
	// lambda, neither this Zygote nor its children will work
	for _, nodePkg := range node.Packages {
		found := false
		for _, p := range packages {
			if p == nodePkg {
				found = true
				break
			}
		}
		if !found {
			return nil
		}
	}

	// check our descendents; is one of them a Zygote that works?
	// we prefer a child Zygote over the one for this node,
	// because they have more packages pre-imported
	for _, child := range node.Children {
		result := child.Lookup(packages)
		if result != nil {
			return result
		}
	}

	return node
}
