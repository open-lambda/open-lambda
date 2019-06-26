// handler package implements a library for handling run lambda requests from
// the worker server.
package handler

import (
	"container/list"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path/filepath"
	"sync"
	"time"

	"github.com/open-lambda/open-lambda/ol/config"
	"github.com/open-lambda/open-lambda/ol/pip-manager"

	sb "github.com/open-lambda/open-lambda/ol/sandbox"
)

// provides thread-safe access to map of lambda functions, and stores
// references to various helpers:
// 1. codePuller, for pulling lambda code
// 2. pipMgr, for doing pip installs on the host
// 3. sbPool, for allocating Sandbox instances
type LambdaMgr struct {
	codePuller *CodePuller
	pipMgr     pip.InstallManager
	sbPool     sb.SandboxPool

	mapMutex sync.Mutex
	lfuncMap map[string]*LambdaFunc
}

// Represents a single lambda function (the code)
type LambdaFunc struct {
	lmgr *LambdaMgr
	name string

	// lambda code
	lastPull *time.Time
	codeDir  string
	imports  []string
	installs []string

	// lambda execution
	funcChan  chan *Invocation // server to func
	instChan  chan *Invocation // func to instances
	doneChan  chan *Invocation // instances to func
	instances *list.List
}

// This is essentially a virtual sandbox.  It is backed by a real
// Sandbox (when it is allowed to allocate one).  It pauses/unpauses
// based on usage, and starts fresh instances when they die.
type LambdaInstance struct {
	lfunc *LambdaFunc

	// copied from LambdaFunc, at the time of creation
	codeDir  string
	imports  []string
	installs []string
}

// represents an HTTP request to be handled by a lambda instance
type Invocation struct {
	w    http.ResponseWriter
	r    *http.Request
	done chan bool // func to server
}

func NewLambdaMgr() (mgr *LambdaMgr, err error) {
	var t time.Time

	// init code puller, pip manager, handler cache, and init cache
	log.Printf("Create CodePuller")
	t = time.Now()
	cp, err := NewCodePuller(filepath.Join(config.Conf.Worker_dir, "lambda_code"), config.Conf.Registry)
	if err != nil {
		return nil, err
	}
	log.Printf("Initialized CodePuller (took %v)", time.Since(t))

	log.Printf("Create InstallManager")
	t = time.Now()
	pm, err := pip.InitInstallManager()
	if err != nil {
		return nil, err
	}
	log.Printf("Create InstallManager (took %v)", time.Since(t))

	log.Printf("Create ContainerFactory")
	t = time.Now()
	sbp, err := sb.SandboxPoolFromConfig()
	if err != nil {
		return nil, err
	}
	log.Printf("Initialized handler container factory (took %v)", time.Since(t))

	t = time.Now()

	mgr = &LambdaMgr{
		lfuncMap:   make(map[string]*LambdaFunc),
		codePuller: cp,
		pipMgr:     pm,
		sbPool:     sbp,
	}

	return mgr, nil
}

// Returns an existing instance (if there is one), or creates a new one
func (mgr *LambdaMgr) Get(name string) (f *LambdaFunc) {
	mgr.mapMutex.Lock()
	defer mgr.mapMutex.Unlock()

	f = mgr.lfuncMap[name]

	if f == nil {
		mgr.lfuncMap[name] = &LambdaFunc{
			lmgr:      mgr,
			name:      name,
			imports:   []string{},
			installs:  []string{},
			funcChan:  make(chan *Invocation, 32),
			instChan:  make(chan *Invocation, 32),
			doneChan:  make(chan *Invocation, 32),
			instances: list.New(),
		}

		f = mgr.lfuncMap[name]
	}

	go f.Task()

	return f
}

func (mgr *LambdaMgr) Cleanup() {
	mgr.mapMutex.Lock() // we don't unlock, because nobody else should use this anyway
	mgr.sbPool.Cleanup()
}

func (f *LambdaFunc) Invoke(w http.ResponseWriter, r *http.Request) {
	done := make(chan bool)
	req := &Invocation{w: w, r: r, done: done}

	// send invocation to lambda func task, if room in queue
	select {
	case f.funcChan <- req:
		// block until it's done
		<-done
	default:
		// queue cannot accept more, so reply with backoff
		req.w.WriteHeader(http.StatusTooManyRequests)
		req.w.Write([]byte("lambda function queue is full"))
	}
}

func (f *LambdaFunc) checkCodeCache() (err error) {
	// check if there is newer code, download it if necessary
	now := time.Now()
	cache_ns := int64(config.Conf.Registry_cache_ms) * 1000000
	if f.lastPull == nil || int64(now.Sub(*f.lastPull)) > cache_ns {
		codeDir, err := f.lmgr.codePuller.Pull(f.name)
		if err != nil {
			return err
		}

		imports, installs, err := parsePkgFile(codeDir)
		if err != nil {
			return err
		}

		f.lastPull = &now
		f.codeDir = codeDir
		f.imports = imports
		f.installs = installs
	}

	// TODO: shouldn't we do this inside the Sandbox?
	if err := f.lmgr.pipMgr.Install(f.installs); err != nil {
		return err
	}

	return nil
}

// this Task receives lambda requests, fetches new lambda code as
// needed, and dispatches to a set of lambda instances.  Task also
// monitors outstanding requests, and scales the number of instances
// up or down as needed.
//
// communication for a given request is as follows:
//
// server -> Task -> instance -> Task -> server
//
// each of the 4 handoffs above is over a chan.  In order, those chans are:
// 1. LambdaFunc.funcChan
// 2. LambdaFunc.instChan
// 3. LambdaFunc.doneChan
// 4. Invocation.done
//
// If either LambdaFunc.funcChan or LambdaFunc.instChan is full, we
// respond to the client with a backoff message
// (http.StatusTooManyRequests)
func (f *LambdaFunc) Task() {
	outstandingReqs := 0

	for {
		select {
		case req := <-f.funcChan:
			// incoming request

			// TODO: if checkCodeCache pulls new code, restart all the instances
			if err := f.checkCodeCache(); err == nil {
				select {
				case f.instChan <- req:
					outstandingReqs += 1
				default:
					// queue cannot accept more, so reply with backoff
					req.w.WriteHeader(http.StatusTooManyRequests)
					req.w.Write([]byte("lambda instance queue is full"))
					req.done <- true
				}
			} else {
				log.Printf("Error checking for new lambda code: %v", err)
				req.w.WriteHeader(http.StatusInternalServerError)
				req.w.Write([]byte(err.Error() + "\n"))
				req.done <- true
			}
		case req := <-f.doneChan:
			// notification that request we sent out has completed

			outstandingReqs -= 1
			req.done <- true
		}

		// TODO: upside or downsize, based on request to
		// instance ratio
		if f.instances.Len() < 1 {
			f.newInstance()
		}
	}
}

func (f *LambdaFunc) newInstance() {
	if f.codeDir == "" {
		panic("cannot start instance until code has been fetched")
	}

	linst := &LambdaInstance{
		lfunc:    f,
		codeDir:  f.codeDir,
		imports:  f.imports,
		installs: f.installs,
	}

	f.instances.PushBack(linst)

	go linst.Task()
}

// this Task manages a single Sandbox (at any given time), and
// forwards requests from the function queue to that Sandbox.
// when there are no requests, the Sandbox is paused.
//
// These errors are handled as follows by Task:
//
// 1. Sandbox.Pause/Unpause: discard Sandbox, create new one to handle request
// 2. Sandbox.Create/Channel: discard Sandbox, propagate HTTP 500 to client
// 3. Error inside Sandbox: simply propagate whatever occured to client (TODO: restart Sandbox)
func (linst *LambdaInstance) Task() {
	f := linst.lfunc
	scratchPrefix := filepath.Join(config.Conf.Worker_dir, "handlers", f.name)

	var sb sb.Sandbox = nil
	//var client *http.Client = nil // whenever we create a Sandbox, we init this too
	var proxy *httputil.ReverseProxy = nil // whenever we create a Sandbox, we init this too
	var err error

outer:
	for {
		// wait for a request (blocking) before making the Sandbox ready
		req := <-f.instChan

		// if we have a sandbox, try unpausing it to see if it is still alive
		if sb != nil {
			// Unpause will often fail, because evictors
			// are likely to prefer to evict paused
			// sandboxes rather than inactive sandboxes.
			// Thus, if this fails, we'll try to handle it
			// by just creating a new sandbox.
			if err := sb.Unpause(); err != nil {
				log.Printf("discard sandbox %s due to Unpause error: %s", sb.ID())
				sb = nil
			}
		}

		// if we don't already have a Sandbox, create one, and HTTP proxy over the channel
		if sb == nil {
			sb, err = f.lmgr.sbPool.Create(nil, true, linst.codeDir, scratchPrefix, linst.imports)
			if err != nil {
				req.w.WriteHeader(http.StatusInternalServerError)
				req.w.Write([]byte("could not create Sandbox: " + err.Error() + "\n"))
				f.doneChan <- req
				continue // wait for another request before retrying
			}

			var tr *http.Transport
			tr, err = sb.Channel()
			if err != nil {
				req.w.WriteHeader(http.StatusInternalServerError)
				req.w.Write([]byte("could not connect to Sandbox: " + err.Error() + "\n"))
				f.doneChan <- req
				log.Printf("discard sandbox %s due to Channel error: %s", sb.ID(), err.Error())
				sb = nil
				continue // wait for another request before retrying
			}

			u, err := url.Parse("http://container")
			if err != nil {
				panic(err)
			}
			proxy = httputil.NewSingleHostReverseProxy(u)
			proxy.Transport = tr
		}

		// serve requests for as long as we can without blocking on the incoming queue
		for req != nil {
			//serve request

			// TODO: somehow check result, and kill error on 500 (others?)
			proxy.ServeHTTP(req.w, req.r)

			f.doneChan <- req
			if err != nil {
				log.Printf("discard sandbox %s due to HTTP error: %s", sb.ID(), err.Error())
				sb.Destroy()
				sb = nil
				continue outer
			}

			// grab another (non-blocking)
			select {
			case req = <-f.instChan:
			default:
				req = nil
			}
		}

		if err := sb.Pause(); err != nil {
			log.Printf("discard sandbox %s due to Pause error: %s", sb.ID())
			sb = nil
		}
	}
}
