package lambda

import (
	"bufio"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/open-lambda/open-lambda/go/common"
	"github.com/open-lambda/open-lambda/go/worker/lambda/packages"
	"github.com/open-lambda/open-lambda/go/worker/sandbox"
)

type FunctionMeta struct {
	Sandbox *sandbox.SandboxMeta `json:"sandbox"` // Existing sandbox metadata
	Config  *common.LambdaConfig `json:"config"`  // New Lambda config (from YAML)
}

// LambdaFunc represents a single lambda function (the code)
type LambdaFunc struct {
	lmgr *LambdaMgr
	name string

	rtType common.RuntimeType

	// lambda code
	lastPull *time.Time
	codeDir  string
	Meta     *FunctionMeta

	// lambda execution
	funcChan   chan *Invocation // server to func
	instChan   chan *Invocation // func to instances
	doneChan   chan *Invocation // instances to func
	nInstances int

	// send chan to the kill chan to destroy the instance, then
	// wait for msg on sent chan to block until it is done
	killChan chan chan bool

	// killChan shared with each invocation.
	invocationKillChan chan chan bool
}

// Invoke handles the invocation of the lambda function.
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
	slog.Info(fmt.Sprintf("%s [FUNC %s]", strings.TrimRight(msg, "\n"), f.name))
}

// parseMeta reads in a requirements.txt file that was built from pip-compile
func parseMeta(codeDir string) (*FunctionMeta, error) {
	sandboxMeta := &sandbox.SandboxMeta{
		Installs: []string{},
		Imports:  []string{},
	}

	path := filepath.Join(codeDir, "requirements.txt")
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		// having a requirements.txt is optional
	} else if err != nil {
		return nil, err
	}
	defer file.Close()

	scnr := bufio.NewScanner(file)
	for scnr.Scan() {
		line := strings.ReplaceAll(scnr.Text(), " ", "")
		pkg := strings.Split(line, "#")[0]
		if pkg != "" {
			pkg = packages.NormalizePkg(pkg)
			sandboxMeta.Installs = append(sandboxMeta.Installs, pkg)
		}
	}

	// Load Lambda configuration from ol.yaml
	lambdaConfig, err := common.LoadLambdaConfig(codeDir)
	if err != nil {
		return nil, fmt.Errorf("failed to parse lambda configuration file: %v", err)
	}

	// Return combined FunctionMeta
	return &FunctionMeta{
		Sandbox: sandboxMeta,
		Config:  lambdaConfig,
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
				slog.Error(fmt.Sprintf("could not cleanup %s after failed pull", codeDir))
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

		// make sure all specified dependencies are installed
		// (but don't recursively find others)
		for _, pkg := range meta.Sandbox.Installs {
			if _, err := f.lmgr.PackagePuller.GetPkg(pkg); err != nil {
				return err
			}
		}

		f.lmgr.DepTracer.TraceFunction(codeDir, meta.Sandbox.Installs)
		f.Meta = meta
	} else if rtType == common.RT_NATIVE {
		slog.Info("Got native function")

		// Initialize f.Meta for native functions for consistensy.
		f.Meta = &FunctionMeta{
			Sandbox: nil,                              // Sandbox is nil for native functions
			Config:  common.LoadDefaultLambdaConfig(), // Load default configuration
		}
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
	var lastScaling *time.Time
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

			// Check if the HTTP method is valid
			if !f.Meta.Config.IsHTTPMethodAllowed(req.r.Method) {
				req.w.WriteHeader(http.StatusMethodNotAllowed)
				req.w.Write([]byte(fmt.Sprintf(
					"HTTP method not allowed. Sent: %s, Allowed: %v\n",
					req.r.Method,
					f.Meta.Config.AllowedHTTPMethods(),
				)))

				req.done <- true
				continue
			}

			if oldCodeDir != "" && oldCodeDir != f.codeDir {
				for i := 0; i < f.nInstances; i++ {
					waitChan := f.AsyncKillOneInvocation()
					cleanupChan <- waitChan
				}
				f.nInstances = 0

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
			for i := 0; i < f.nInstances; i++ {
				waitChan := f.AsyncKillOneInvocation()
				cleanupChan <- waitChan
			}
			if f.codeDir != "" {
				// cleanupChan <- f.codeDir
			}
			close(cleanupChan)
			<-cleanupTaskDone
			done <- true
			return
		}

		// POLICY: how many instances (i.e., virtual sandboxes) should we allocate?

		// AUTOSCALING STEP 1: decide how many instances we want

		// let's aim to have 1 sandbox per 10ms of outstanding work
		// TODO make this configurable
		inProgressWorkMs := outstandingReqs * execMs.Avg
		desiredInstances := inProgressWorkMs / 10

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

		// make at most one scaling adjustment per 100ms
		adjustFreq := time.Millisecond * 100
		now := time.Now()
		if lastScaling != nil {
			elapsed := now.Sub(*lastScaling)
			if elapsed < adjustFreq {
				if desiredInstances != f.nInstances {
					timeout = time.NewTimer(adjustFreq - elapsed)
				}
				continue
			}
		}

		// kill or start at most one instance to get closer to
		// desired number
		if f.nInstances < desiredInstances {
			f.printf("increase instances to %d", f.nInstances+1)
			f.newInstance()
			lastScaling = &now
		} else if f.nInstances > desiredInstances {
			f.printf("reduce instances to %d", f.nInstances-1)
			waitChan := f.AsyncKillOneInvocation()
			f.nInstances--
			cleanupChan <- waitChan
			lastScaling = &now
		}

		if f.nInstances != desiredInstances {
			// we can only adjust quickly, so we want to
			// run through this loop again as soon as
			// possible, even if there are no requests to
			// service.
			timeout = time.NewTimer(adjustFreq)
		}
	}
}

// newInstance creates a new lambda instance.
func (f *LambdaFunc) newInstance() {
	if f.codeDir == "" {
		panic("cannot start instance until code has been fetched")
	}

	linst := &LambdaInstance{
		lfunc:    f,
		codeDir:  f.codeDir,
		meta:     f.Meta,
		killChan: f.invocationKillChan,
	}

	f.nInstances++

	go linst.Task()
}

// Kill signals the lambda function to terminate all instances and perform cleanup.
func (f *LambdaFunc) Kill() {
	done := make(chan bool)
	f.killChan <- done
	<-done
}

func (f *LambdaFunc) AsyncKillOneInvocation() chan bool {
	done := make(chan bool)
	f.invocationKillChan <- done
	return done
}
