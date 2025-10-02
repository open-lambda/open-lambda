package lambda

import (
	"container/list"
	"fmt"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"
	"sync"

	"github.com/open-lambda/open-lambda/go/common"
	"github.com/open-lambda/open-lambda/go/worker/lambda/packages"
	"github.com/open-lambda/open-lambda/go/worker/lambda/zygote"
	"github.com/open-lambda/open-lambda/go/worker/sandbox"
)

// LambdaMgr provides thread-safe getting of lambda functions and collects all
// lambda subsystems (resource pullers and sandbox pools) in one place
type LambdaMgr struct {
	// subsystems (these are thread safe)
	sbPool sandbox.SandboxPool
	*packages.DepTracer
	*packages.PackagePuller // depends on sbPool and DepTracer
	zygote.ZygoteProvider   // depends PackagePuller
	*HandlerPuller          // depends on sbPool and ImportCache[optional]

	// storage dirs that we manage
	codeDirs    *common.DirMaker
	scratchDirs *common.DirMaker

	// thread-safe map from a lambda's name to its LambdaFunc
	mapMutex sync.Mutex
	lfuncMap map[string]*LambdaFunc
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

var lambdaMgr *LambdaMgr
var once sync.Once

// GetLambdaManagerInstance returns a singleton instance of LambdaMgr.
// This is necessary because:
//   - Each LambdaMgr sets up directories and manages shared code storage on the worker,
//     so multiple instances causes the same folders to get created again, leading to conflicting operations.
//   - All triggers (e.g., HTTP, Kafka) on the same worker need to use the same LambdaMgr
//     to run the same code
//   - Using sync.Once ensures that the LambdaMgr is only initialized once per worker process.
func GetLambdaManagerInstance() (*LambdaMgr, error) {
	var err error
	once.Do(func() {
		lambdaMgr, err = newLambdaMgr()
	})
	return lambdaMgr, err
}

// newLambdaMgr creates a new LambdaMgr instance and initializes its subsystems.
// This is private to force packages to use the singleton method GetLambdaManagerInstance
func newLambdaMgr() (res *LambdaMgr, err error) {
	mgr := &LambdaMgr{
		lfuncMap: make(map[string]*LambdaFunc),
	}
	defer func() {
		if err != nil {
			slog.Error(fmt.Sprintf("Cleanup Lambda Manager due to error: %v", err))
			mgr.Cleanup()
		}
	}()

	mgr.codeDirs, err = common.NewDirMaker("code", common.Conf.Storage.Code.Mode())
	if err != nil {
		return nil, err
	}
	mgr.scratchDirs, err = common.NewDirMaker("scratch", common.Conf.Storage.Scratch.Mode())
	if err != nil {
		return nil, err
	}

	slog.Info("Creating SandboxPool")
	mgr.sbPool, err = sandbox.SandboxPoolFromConfig("sandboxes", common.Conf.Mem_pool_mb)
	if err != nil {
		return nil, err
	}

	slog.Info("Creating DepTracer")
	mgr.DepTracer, err = packages.NewDepTracer(filepath.Join(common.Conf.Worker_dir, "dep-trace.json"))
	if err != nil {
		return nil, err
	}

	slog.Info("Creating PackagePuller")
	mgr.PackagePuller, err = packages.NewPackagePuller(mgr.sbPool, mgr.DepTracer)
	if err != nil {
		return nil, err
	}

	if common.Conf.Features.Import_cache != "" {
		slog.Info("Creating ImportCache")
		mgr.ZygoteProvider, err = zygote.NewZygoteProvider(mgr.codeDirs, mgr.scratchDirs, mgr.sbPool, mgr.PackagePuller)
		if err != nil {
			return nil, err
		}
	}

	slog.Info("Creating HandlerPuller")
	mgr.HandlerPuller, err = NewHandlerPuller(mgr.codeDirs)
	if err != nil {
		return nil, err
	}

	return mgr, nil
}

// Get returns an existing LambdaFunc instance or creates a new one if it doesn't exist.
func (mgr *LambdaMgr) Get(name string) (f *LambdaFunc) {
	mgr.mapMutex.Lock()
	defer mgr.mapMutex.Unlock()

	f = mgr.lfuncMap[name]

	if f == nil {
		f = &LambdaFunc{
			lmgr: mgr,
			name: name,
			// TODO make these configurable
			funcChan:  make(chan *Invocation, 1024),
			instChan:  make(chan *Invocation, 1024),
			doneChan:  make(chan *Invocation, 1024),
			instances: list.New(),
			killChan:  make(chan chan bool, 1),
		}

		go f.Task()
		mgr.lfuncMap[name] = f
	}

	return f
}

// Debug returns the debug information of the sandbox pool.
func (mgr *LambdaMgr) Debug() string {
	return mgr.sbPool.DebugString() + "\n"
}

// DumpStatsToLog logs the profiling information of the LambdaMgr.
func (_ *LambdaMgr) DumpStatsToLog() {
	snapshot := common.SnapshotStats()

	sec := func(name string) float64 {
		return float64(snapshot[name+".cnt"]*snapshot[name+".ms-avg"]) / 1000
	}

	time := func(indent int, name string, parent string) {
		selftime := sec(name)
		ptime := sec(parent)
		tabs := strings.Repeat("\t", indent)
		if ptime > 0 {
			slog.Info(fmt.Sprintf("%s%s: %.3f (%.1f%%)", tabs, name, selftime, selftime/ptime*100))
		} else {
			slog.Info(fmt.Sprintf("%s%s: %.3f", tabs, name, selftime))
		}
	}

	slog.Info("Request Profiling (cumulative seconds):")
	time(0, "LambdaFunc.Invoke", "")

	time(1, "LambdaInstance-WaitSandbox", "LambdaFunc.Invoke")
	time(2, "LambdaInstance-WaitSandbox-Unpause", "LambdaInstance-WaitSandbox")
	time(2, "LambdaInstance-WaitSandbox-NoImportCache", "LambdaInstance-WaitSandbox")
	time(2, "ImportCache.Create", "LambdaInstance-WaitSandbox")
	time(3, "ImportCache.root.Lookup", "ImportCache.Create")
	time(3, "ImportCache.createChildSandboxFromNode", "ImportCache.Create")
	time(4, "ImportCache.getSandboxInNode", "ImportCache.createChildSandboxFromNode")
	time(4, "ImportCache.createChildSandboxFromNode:childSandboxPool.Create",
		"ImportCache.createChildSandboxFromNode")
	time(4, "ImportCache.putSandboxInNode", "ImportCache.createChildSandboxFromNode")
	time(5, "ImportCache.putSandboxInNode:Lock", "ImportCache.putSandboxInNode")
	time(5, "ImportCache.putSandboxInNode:Pause", "ImportCache.putSandboxInNode")
	time(1, "LambdaInstance-ServeRequests", "LambdaFunc.Invoke")
	time(2, "LambdaInstance-RoundTrip", "LambdaInstance-ServeRequests")
}

// Cleanup performs cleanup operations for the LambdaMgr and its subsystems.
func (mgr *LambdaMgr) Cleanup() {
	mgr.mapMutex.Lock() // don't unlock, because this shouldn't be used anymore

	mgr.DumpStatsToLog()

	// HandlerPuller+PackagePuller requires no cleanup

	// 1. cleanup handler Sandboxes
	// 2. cleanup Zygote Sandboxes (after the handlers, which depend on the Zygotes)
	// 3. cleanup SandboxPool underlying both of above
	for _, f := range mgr.lfuncMap {
		slog.Info(fmt.Sprintf("Kill function: %s", f.name))
		f.Kill()
	}

	if mgr.ZygoteProvider != nil {
		mgr.ZygoteProvider.Cleanup()
	}

	if mgr.sbPool != nil {
		mgr.sbPool.Cleanup() // assumes all Sandboxes are gone
	}

	// cleanup DepTracer
	if mgr.DepTracer != nil {
		mgr.DepTracer.Cleanup()
	}

	if mgr.codeDirs != nil {
		mgr.codeDirs.Cleanup()
	}

	if mgr.scratchDirs != nil {
		mgr.scratchDirs.Cleanup()
	}
}
