package lambda

import (
	"bufio"
	"container/list"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/open-lambda/open-lambda/ol/common"
	"github.com/open-lambda/open-lambda/ol/worker/sandbox"
)

// LambdaFunc represents a single lambda function (the code)
type LambdaFunc struct {
	lmgr *LambdaMgr
	name string

	rtType common.RuntimeType

	// lambda code
	lastPull *time.Time
	codeDir  string
	meta     *sandbox.SandboxMeta

	// lambda execution
	funcChan  chan *Invocation // server to func
	instChan  chan *Invocation // func to instances
	doneChan  chan *Invocation // instances to func
	instances *list.List

	// send chan to the kill chan to destroy the instance, then
	// wait for msg on sent chan to block until it is done
	killChan chan chan bool
}

func (f *LambdaFunc) Invoke(w http.ResponseWriter, r *http.Request) {
	t := common.T0("LambdaFunc.Invoke")
	defer t.T1()

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
		req.w.Write([]byte("lambda function queue is full\n"))
	}
}

// add function name to each log message so we know which logs
// correspond to which LambdaFuncs
func (f *LambdaFunc) printf(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	log.Printf("%s [FUNC %s]", strings.TrimRight(msg, "\n"), f.name)
}

// the function code may contain comments such as the following:
//
// # ol-install: parso,jedi,idna,chardet,certifi,requests
// # ol-import: parso,jedi,idna,chardet,certifi,requests,urllib3
//
// The first list should be installed with pip install.  The latter is
// a hint about what may be imported (useful for import cache).
//
// We support exact pkg versions (e.g., pkg==2.0.0), but not < or >.
// If different lambdas import different versions of the same package,
// we will install them, for example, to
// /packages/pkg==1.0.0/files/pkg and /packages/pkg==2.0.0/files/pkg.
// Each .../files path a handler needs is added to its sys.path.
func parseMeta(codeDir string) (meta *sandbox.SandboxMeta, err error) {
	installs := make([]string, 0)
	imports := make([]string, 0)

	path := filepath.Join(codeDir, "f.py")
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scnr := bufio.NewScanner(file)
	for scnr.Scan() {
		line := strings.ReplaceAll(scnr.Text(), " ", "")
		parts := strings.Split(line, ":")
		if parts[0] == "#ol-install" {
			for _, val := range strings.Split(parts[1], ",") {
				val = strings.TrimSpace(val)
				if len(val) > 0 {
					installs = append(installs, val)
				}
			}
		} else if parts[0] == "#ol-import" {
			for _, val := range strings.Split(parts[1], ",") {
				val = strings.TrimSpace(val)
				if len(val) > 0 {
					imports = append(imports, val)
				}
			}
		}
	}

	for i, pkg := range installs {
		installs[i] = normalizePkg(pkg)
	}

	return &sandbox.SandboxMeta{
		Installs: installs,
		Imports:  imports,
	}, nil
}

// if there is any error:
// 1. we won't switch to the new code
// 2. we won't update pull time (so well check for a fix next time)
func (f *LambdaFunc) pullHandlerIfStale() (err error) {
	// check if there is newer code, download it if necessary
	now := time.Now()
	cacheNs := int64(common.Conf.Registry_cache_ms) * 1000000

	// should we check for new code?
	if f.lastPull != nil && int64(now.Sub(*f.lastPull)) < cacheNs {
		return nil
	}

	// is there new code?
	rtType, codeDir, err := f.lmgr.HandlerPuller.Pull(f.name)
	if err != nil {
		return err
	}

	if codeDir == f.codeDir {
		return nil
	}

	f.rtType = rtType

	defer func() {
		if err != nil {
			if err := os.RemoveAll(codeDir); err != nil {
				log.Printf("could not cleanup %s after failed pull\n", codeDir)
			}

			if rtType == common.RT_PYTHON {
				// we dirty this dir (e.g., by setting up
				// symlinks to packages, so we want the
				// HandlerPuller to give us a new one next
				// time, even if the code hasn't changed
				f.lmgr.HandlerPuller.Reset(f.name)
			}
		}
	}()

	if rtType == common.RT_PYTHON {
		// inspect new code for dependencies; if we can install
		// everything necessary, start using new code
		meta, err := parseMeta(codeDir)
		if err != nil {
			return err
		}

		meta.Installs, err = f.lmgr.PackagePuller.InstallRecursive(meta.Installs)
		if err != nil {
			return err
		}
		f.lmgr.DepTracer.TraceFunction(codeDir, meta.Installs)
		f.meta = meta
	} else if rtType == common.RT_NATIVE {
		log.Printf("Got native function")
	}

	f.codeDir = codeDir
	f.lastPull = &now
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
	f.printf("debug: LambdaFunc.Task() runs on goroutine %d", common.GetGoroutineID())

	// we want to perform various cleanup actions, such as killing
	// instances and deleting old code.  We want to do these
	// asynchronously, but in order.  Thus, we use a chan to get
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
	cleanupChan := make(chan any, 32)
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
	execMs := common.NewRollingAvg(10)
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
			if err := f.pullHandlerIfStale(); err != nil {
				f.printf("Error checking for new lambda code at `%s`: %v", f.codeDir, err)
				req.w.WriteHeader(http.StatusInternalServerError)
				req.w.Write([]byte(err.Error() + "\n"))
				req.done <- true
				continue
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

			f.lmgr.DepTracer.TraceInvocation(f.codeDir)

			select {
			case f.instChan <- req:
				// msg: function -> instance
				outstandingReqs++
			default:
				// queue cannot accept more, so reply with backoff
				req.w.WriteHeader(http.StatusTooManyRequests)
				req.w.Write([]byte("lambda instance queue is full\n"))
				req.done <- true
			}
		case req := <-f.doneChan:
			// msg: instance -> function

			execMs.Add(req.execMs)
			outstandingReqs--

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
				//cleanupChan <- f.codeDir
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
		meta:     f.meta,
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
