package lambda

import (
	"bytes"
	"container/list"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/open-lambda/open-lambda/ol/config"
	"github.com/open-lambda/open-lambda/ol/sandbox"
	"github.com/open-lambda/open-lambda/ol/stats"
)

// provides thread-safe getting of lambda functions and collects all
// lambda subsystems (resource pullers and sandbox pools) in one place
type LambdaMgr struct {
	// subsystems (these are thread safe)
	*HandlerPuller
	*ModulePuller
	sbPool      sandbox.SandboxPool
	importCache *sandbox.ImportCache

	// thread-safe map from a lambda's name to its LambdaFunc
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

	// send chan to the kill chan to destroy the instance, then
	// wait for msg on sent chan to block until it is done
	killChan chan chan bool
}

// This is essentially a virtual sandbox.  It is backed by a real
// Sandbox (when it is allowed to allocate one).  It pauses/unpauses
// based on usage, and starts fresh instances when they die.
type LambdaInstance struct {
	lfunc *LambdaFunc

	// snapshot of LambdaFunc, at the time the LambdaInstance is created
	codeDir  string
	imports  []string
	installs []string

	// send chan to the kill chan to destroy the instance, then
	// wait for msg on sent chan to block until it is done
	killChan chan chan bool
}

// represents an HTTP request to be handled by a lambda instance
type Invocation struct {
	w http.ResponseWriter
	r *http.Request

	// signal to client that response has been written to w
	done chan bool

	// how many milliseconds did ServeHTTP take?  (doesn't count
	// queue time or Sandbox init)
	execMs int
}

func NewLambdaMgr() (mgr *LambdaMgr, err error) {
	log.Printf("Create HandlerPuller")
	hp, err := NewHandlerPuller(filepath.Join(config.Conf.Worker_dir, "lambda_code"), config.Conf.Registry)
	if err != nil {
		return nil, err
	}

	log.Printf("Create ModulePuller")
	mp, err := NewModulePuller()
	if err != nil {
		return nil, err
	}

	log.Printf("Create SandboxPool")
	sbp, err := sandbox.SandboxPoolFromConfig("sock-handlers", config.Conf.Handler_cache_mb)
	if err != nil {
		return nil, err
	}

	importCacheMb := config.Conf.Import_cache_mb
	var importCache *sandbox.ImportCache = nil
	if importCacheMb > 0 {
		log.Printf("Create ImportCache")
		importCache, err = sandbox.NewImportCache("sock-cache", importCacheMb)
		if err != nil {
			return nil, err
		}
	}

	return &LambdaMgr{
		lfuncMap:      make(map[string]*LambdaFunc),
		HandlerPuller: hp,
		ModulePuller:  mp,
		sbPool:        sbp,
		importCache:   importCache,
	}, nil
}

// Returns an existing instance (if there is one), or creates a new one
func (mgr *LambdaMgr) Get(name string) (f *LambdaFunc) {
	mgr.mapMutex.Lock()
	defer mgr.mapMutex.Unlock()

	f = mgr.lfuncMap[name]

	if f == nil {
		f = &LambdaFunc{
			lmgr:      mgr,
			name:      name,
			imports:   []string{},
			installs:  []string{},
			funcChan:  make(chan *Invocation, 32),
			instChan:  make(chan *Invocation, 32),
			doneChan:  make(chan *Invocation, 32),
			instances: list.New(),
			killChan:  make(chan chan bool, 1),
		}

		go f.Task()
		mgr.lfuncMap[name] = f
	}

	return f
}

func (mgr *LambdaMgr) Cleanup() {
	// HandlerPuller requires no cleanup

	// ModulePuller requires no cleanup

	// cleanup SandboxPool
	mgr.mapMutex.Lock() // don't unlock, because this shouldn't be used anymore
	for _, f := range mgr.lfuncMap {
		fmt.Printf("Kill function: %s", f.name)
		f.Kill()
	}
	mgr.sbPool.Cleanup() // assumes all Sandboxes are gone

	// cleanup ImportCache
	if mgr.importCache != nil {
		mgr.importCache.Cleanup()
	}
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

// add function name to each log message so we know which logs
// correspond to which LambdaFuncs
func (f *LambdaFunc) printf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	log.Printf("%s [FUNC %s]", strings.TrimRight(msg, "\n"), f.name)
}

func (f *LambdaFunc) checkCodeCache() (err error) {
	// check if there is newer code, download it if necessary
	now := time.Now()
	cache_ns := int64(config.Conf.Registry_cache_ms) * 1000000
	if f.lastPull == nil || int64(now.Sub(*f.lastPull)) > cache_ns {
		codeDir, err := f.lmgr.HandlerPuller.Pull(f.name)
		if err != nil {
			return err
		}

		if codeDir == f.codeDir {
			// no changes since last time we pulled the code
			return nil
		}

		imports, installs, err := parsePkgFile(codeDir)
		if err != nil {
			return err
		}

		f.lastPull = &now
		f.codeDir = codeDir
		f.imports = imports
		f.installs = installs

		// TODO: shouldn't we do this inside the Sandbox?
		if err := f.lmgr.ModulePuller.Install(f.installs); err != nil {
			return err
		}
	}

	return nil
}

// this Task receives lambda requests, fetches new lambda code as
// needed, and dispatches to a set of lambda instances.  Task also
// monitors outstanding requests, and scales the number of instances
// up or down as needed.
//
// communication for a given request is as follows (each of the four
// transfers are commented within the function):
//
// client -> function -> instance -> function -> client
//
// each of the 4 handoffs above is over a chan.  In order, those chans are:
// 1. LambdaFunc.funcChan
// 2. LambdaFunc.instChan
// 3. LambdaFunc.doneChan
// 4. Invocation.done
//
// If either LambdaFunc.funcChan or LambdaFunc.instChan is full, we
// respond to the client with a backoff message: StatusTooManyRequests
func (f *LambdaFunc) Task() {
	f.printf("debug: LambdaFunc.Task() runs on goroutine %d", getGID())

	// we want to perform various cleanup actions, such as killing
	// instances and deleting old code.  We want to do these
	// asyncronously, but in order.  Thus, we use a chan to get
	// FIFO behavior and a single cleanup task to get async.
	//
	// two types can be sent to this chan:
	//
	// 1. string: this is a path to be deleted
	//
	// 2. chan: this is a signal chan that corresponds to
	// previously initiated cleanup work.  We block until we
	// receive the complete signal, before proceeding to
	// subsequent cleanup tasks in the FIFO.
	cleanupChan := make(chan interface{}, 32)
	cleanupTaskDone := make(chan bool)
	go func() {
		for {
			msg, ok := <-cleanupChan
			if !ok {
				cleanupTaskDone <- true
				return
			}

			switch op := msg.(type) {
			case string:
				if err := os.RemoveAll(op); err != nil {
					f.printf("Async code cleanup could not delete %s, even after all instances using it killed: %v", op, err)
				}
			case chan bool:
				<-op
			}
		}
	}()

	// stats for autoscaling
	outstandingReqs := 0
	execMs := stats.NewRollingAvg(10)
	var lastScaling *time.Time = nil
	timeout := time.NewTimer(0)

	for {
		select {
		case <-timeout.C:
			if f.codeDir == "" {
				continue
			}
		case req := <-f.funcChan:
			// msg: client -> function

			// check for new code, and cleanup old code
			// (and instances that use it) if necessary
			oldCodeDir := f.codeDir
			if err := f.checkCodeCache(); err != nil {
				f.printf("Error checking for new lambda code: %v", err)
				if f.codeDir == "" {
					// we don't have older code we can run instead
					req.w.WriteHeader(http.StatusInternalServerError)
					req.w.Write([]byte(err.Error() + "\n"))
					req.done <- true
					continue
				}
			}

			if oldCodeDir != "" && oldCodeDir != f.codeDir {
				el := f.instances.Front()
				for el != nil {
					waitChan := el.Value.(*LambdaInstance).AsyncKill()
					cleanupChan <- waitChan
					el = el.Next()
				}
				f.instances = list.New()

				// cleanupChan is a FIFO, so this will
				// happen after the cleanup task waits
				// for all instance kills to finish
				cleanupChan <- oldCodeDir
			}

			select {
			case f.instChan <- req:
				// msg: function -> instance
				outstandingReqs += 1
			default:
				// queue cannot accept more, so reply with backoff
				req.w.WriteHeader(http.StatusTooManyRequests)
				req.w.Write([]byte("lambda instance queue is full"))
				req.done <- true
			}
		case req := <-f.doneChan:
			// msg: instance -> function

			execMs.Add(req.execMs)
			outstandingReqs -= 1

			// msg: function -> client
			req.done <- true

		case done := <-f.killChan:
			// signal all instances to die, then wait for
			// cleanup task to finish and exit
			el := f.instances.Front()
			for el != nil {
				waitChan := el.Value.(*LambdaInstance).AsyncKill()
				cleanupChan <- waitChan
				el = el.Next()
			}
			if f.codeDir != "" {
				cleanupChan <- f.codeDir
			}
			close(cleanupChan)
			<-cleanupTaskDone
			done <- true
			return
		}

		// POLICY: how many instances (i.e., virtual sandboxes) should we allocate?

		// AUTOSCALING STEP 1: decide how many instances we want

		// let's aim to have 1 sandbox per second of outstanding work
		inProgressWorkMs := outstandingReqs * execMs.Avg
		desiredInstances := inProgressWorkMs / 1000

		// if we have, say, one job that will take 100
		// seconds, spinning up 100 instances won't do any
		// good, so cap by number of outstanding reqs
		if outstandingReqs < desiredInstances {
			desiredInstances = outstandingReqs
		}

		// always try to have one instance
		if desiredInstances < 1 {
			desiredInstances = 1
		}

		// AUTOSCALING STEP 2: tweak how many instances we have, to get closer to our goal

		// make at most one scaling adjustment per second
		adjustFreq := time.Second
		now := time.Now()
		if lastScaling != nil {
			elapsed := now.Sub(*lastScaling)
			if elapsed < adjustFreq {
				if desiredInstances != f.instances.Len() {
					timeout = time.NewTimer(adjustFreq - elapsed)
				}
				continue
			}
		}

		// kill or start at most one instance to get closer to
		// desired number
		if f.instances.Len() < desiredInstances {
			f.printf("increase instances to %d", f.instances.Len()+1)
			f.newInstance()
			lastScaling = &now
		} else if f.instances.Len() > desiredInstances {
			f.printf("reduce instances to %d", f.instances.Len()-1)
			waitChan := f.instances.Back().Value.(*LambdaInstance).AsyncKill()
			f.instances.Remove(f.instances.Back())
			cleanupChan <- waitChan
			lastScaling = &now
		}

		if f.instances.Len() != desiredInstances {
			// we can only adjust quickly, so we want to
			// run through this loop again as soon as
			// possible, even if there are no requests to
			// service.
			timeout = time.NewTimer(adjustFreq)
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
		killChan: make(chan chan bool, 1),
	}

	f.instances.PushBack(linst)

	go linst.Task()
}

func (f *LambdaFunc) Kill() {
	done := make(chan bool)
	f.killChan <- done
	<-done
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

	var sb sandbox.Sandbox = nil
	//var client *http.Client = nil // whenever we create a Sandbox, we init this too
	var proxy *httputil.ReverseProxy = nil // whenever we create a Sandbox, we init this too
	var err error

	for {
		// wait for a request (blocking) before making the
		// Sandbox ready, or kill if we receive that signal
		var req *Invocation
		select {
		case req = <-f.instChan:
		case killed := <-linst.killChan:
			if sb != nil {
				sb.Destroy()
			}
			killed <- true
			return
		}

		// if we have a sandbox, try unpausing it to see if it is still alive
		if sb != nil {
			// Unpause will often fail, because evictors
			// are likely to prefer to evict paused
			// sandboxes rather than inactive sandboxes.
			// Thus, if this fails, we'll try to handle it
			// by just creating a new sandbox.
			if err := sb.Unpause(); err != nil {
				f.printf("discard sandbox %s due to Unpause error: %s", sb.ID())
				sb = nil
			}
		}

		// if we don't already have a Sandbox, create one, and
		// HTTP proxy over the channel
		if sb == nil {
			var parent sandbox.Sandbox
			if f.lmgr.importCache != nil {
				parent = f.lmgr.importCache.GetParent(linst.imports)
			}

			sb, err = f.lmgr.sbPool.Create(parent, true, linst.codeDir, scratchPrefix, linst.imports)
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
				f.printf("discard sandbox %s due to Channel error: %s", sb.ID(), err.Error())
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

		// below here, we're guaranteed (1) sb != nil, (2) proxy != nil, (3) sb is unpaused

		// serve until we incoming queue is empty
		for req != nil {
			// ask Sandbox to respond, via HTTP proxy
			t0 := time.Now()
			proxy.ServeHTTP(req.w, req.r)
			req.execMs = int(time.Now().Sub(t0) / 1000000)
			f.doneChan <- req

			// check whether we should shutdown (non-blocking)
			select {
			case killed := <-linst.killChan:
				sb.Destroy()
				killed <- true
				return
			default:
			}

			// grab another request (non-blocking)
			select {
			case req = <-f.instChan:
			default:
				req = nil
			}
		}

		if err := sb.Pause(); err != nil {
			f.printf("discard sandbox %s due to Pause error: %s", sb.ID())
			sb = nil
		}
	}
}

// signal the instance to die, return chan that can be used to block
// until it's done
func (linst *LambdaInstance) AsyncKill() chan bool {
	done := make(chan bool)
	linst.killChan <- done
	return done
}

// https://blog.sgmansfield.com/2015/12/goroutine-ids/
func getGID() uint64 {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	b = b[:bytes.IndexByte(b, ' ')]
	n, _ := strconv.ParseUint(string(b), 10, 64)
	return n
}
