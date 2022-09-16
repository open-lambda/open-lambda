package lambda

import (
	"bufio"
	"container/list"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"path/filepath"
	"strings"
	"time"

	"github.com/open-lambda/open-lambda/ol/common"
	"github.com/open-lambda/open-lambda/ol/sandbox"
)

// LambdaFunc represents a single lambda function (the code)
type LambdaFunc struct {
	sync.Mutex

	lmgr *LambdaMgr
	name string
	rtType common.RuntimeType

	// lambda code
	lastPull *time.Time
	codeDir  string
	meta     *sandbox.SandboxMeta

	// sandbox instances (first in, last out)
	idle *list.List
	version int
}

type InvokeError struct {
	httpStatus int
	msg string
	sbMsg string
}

func newInvokeError(httpStatus int, err error, sb sandbox.Sandbox) *InvokeError {
	sbMsg := ""
	if sb != nil {
		sbMsg = sb.DebugString()
	}
	return &InvokeError{
		httpStatus: httpStatus,
		msg: err.Error(),
		sbMsg: sbMsg,
	}
}

func (f *LambdaFunc) Invoke(w http.ResponseWriter, r *http.Request) {
	resp, err := f.InvokeInSandbox(r)
	if err != nil {
		w.WriteHeader(err.httpStatus)
		if _, err := w.Write([]byte(err.msg+"\n")); err != nil {
			f.printf("writing err to http failed: %s\n", err.Error())
		}
		if err.sbMsg != "" {
			if _, err := w.Write([]byte("\nSandbox State: "+err.sbMsg+"\n")); err != nil {
				f.printf("writing err to http failed: %s\n", err.Error())
			}
		}
		return
	}
	defer resp.Body.Close()

	// copy headers (adapted from copyHeaders: https://go.dev/src/net/http/httputil/reverseproxy.go)
	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)

	// copy body
	if _, err := io.Copy(w, resp.Body); err != nil {
		f.printf("reading lambda response failed: "+err.Error()+"\n")
	}
}

func (f *LambdaFunc) InvokeInSandbox(r *http.Request) (*http.Response, *InvokeError) {
	t := common.T0("LambdaFunc.Invoke")
	defer t.T1()

	f.Mutex.Lock()
	defer f.Mutex.Unlock()
	
	// PHASE 1: pre-exec (locked)
	err := f.checkForUpdates()
	if err != nil {
		return nil, newInvokeError(http.StatusInternalServerError, err, nil)
	}
	sb, err := f.getUnpausedSandbox()
	if err != nil {
		return nil, newInvokeError(http.StatusInternalServerError, err, nil)
	}
	version := f.version

	// PHASE 2: exec (concurrent)
	f.Mutex.Unlock()
	resp, err := f.forwardToSandbox(r, sb)
	f.Mutex.Lock()

	// PHASE 3: post exec (locked)
	f.releaseSandbox(sb, version)

	if err != nil {
		return nil, newInvokeError(http.StatusBadGateway, err, sb)
	}
	return resp, nil
}

func (f *LambdaFunc) checkForUpdates() (err error) {
	oldCodeDir := f.codeDir
	if err := f.pullHandlerIfStale(); err != nil {
		return fmt.Errorf("Error checking for new lambda code at `%s`: %v", f.codeDir, err)
	}

	if oldCodeDir != "" && oldCodeDir != f.codeDir {
		f.version++
	}
	return nil
}

func (f *LambdaFunc) forwardToSandbox(r *http.Request, sb sandbox.Sandbox) (*http.Response, error) {
	f.printf("Forwarding request to sandbox")

	// get response from sandbox
	url := "http://root" + r.RequestURI
	httpReq, err := http.NewRequest(r.Method, url, r.Body)
	if err != nil {
		return nil, err
	}

	return sb.Client().Do(httpReq)
}

func (f *LambdaFunc) getUnpausedSandbox() (sb sandbox.Sandbox, err error) {
	// CHOICE 1: try to find an idle sandbox if we can
	for {
		el := f.idle.Front()
		if el == nil {
			break
		}

		f.idle.Remove(el)
		sb := el.Value.(sandbox.Sandbox)
		if err := sb.Unpause(); err == nil {
			return sb, nil
		}
	}

	// CHOICE 2: TODO: should we every wait for a sandbox if requests are pretty short?

	// CHOICE 3: try to create a sandbox from the import cache if we can
	if f.lmgr.ImportCache != nil && f.rtType == common.RT_PYTHON {
		scratchDir := f.lmgr.scratchDirs.Make(f.name)

		// we don't specify parent SB, because ImportCache.Create chooses it for us
		sb, err := f.lmgr.ImportCache.Create(f.lmgr.sbPool, true, f.codeDir, scratchDir, f.meta, f.rtType)
		if err != nil {
			f.printf("failed to get Sandbox from import cache: %s", err.Error())
		} else {
			return sb, err
		}
	}

	// CHOICE 4: create new, parentless sandbox
	f.printf("Creating new sandbox")
	scratchDir := f.lmgr.scratchDirs.Make(f.name)
	return f.lmgr.sbPool.Create(nil, true, f.codeDir, scratchDir, f.meta, f.rtType)
}

func (f *LambdaFunc) releaseSandbox(sb sandbox.Sandbox, version int) {
	if version == f.version {
		f.idle.PushFront(sb)
	} else {
		sb.Destroy("code version outdated")
	}	
	// TODO: can we delete the old code dir?  only after last sandbox delete
}

// add function name to each log message so we know which logs
// correspond to which LambdaFuncs
func (f *LambdaFunc) printf(format string, args ...interface{}) {
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
