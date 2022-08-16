package lambda

import (
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/open-lambda/open-lambda/ol/common"
	"github.com/open-lambda/open-lambda/ol/sandbox"
)

// This is essentially a virtual sandbox.  It is backed by a real
// Sandbox (when it is allowed to allocate one).  It pauses/unpauses
// based on usage, and starts fresh instances when they die.
type LambdaInstance struct {
	lfunc *LambdaFunc

	// snapshot of LambdaFunc, at the time the LambdaInstance is created
	codeDir string
	meta    *sandbox.SandboxMeta

	// send chan to the kill chan to destroy the instance, then
	// wait for msg on sent chan to block until it is done
	killChan chan chan bool
}

// this Task manages a single Sandbox (at any given time), and
// forwards requests from the function queue to that Sandbox.
// when there are no requests, the Sandbox is paused.
//
// These errors are handled as follows by Task:
//
// 1. Sandbox.Pause/Unpause: discard Sandbox, create new one to handle request
// 2. Sandbox.Create/Channel: discard Sandbox, propagate HTTP 500 to client
// 3. Error inside Sandbox: simply propagate whatever occurred to the client (TODO: restart Sandbox)
func (linst *LambdaInstance) Task() {
	f := linst.lfunc

	var sb sandbox.Sandbox = nil
	var err error

	for {
		// wait for a request (blocking) before making the
		// Sandbox ready, or kill if we receive that signal

		var req *Invocation
		select {
		case req = <-f.instChan:
		case killed := <-linst.killChan:
			if sb != nil {
				rtLog := sb.GetRuntimeLog()
				proxyLog := sb.GetProxyLog()
				sb.Destroy("Lambda instance kill signal received")

				log.Printf("Stopped sandbox")

				if common.Conf.Log_output {
					if rtLog != "" {
						log.Printf("Runtime output is:")

						for _, line := range strings.Split(rtLog, "\n") {
							log.Printf("   %s", line)
						}
					}

					if proxyLog != "" {
						log.Printf("Proxy output is:")

						for _, line := range strings.Split(proxyLog, "\n") {
							log.Printf("   %s", line)
						}
					}
				}
			}
			killed <- true
			return
		}

		t := common.T0("LambdaInstance-WaitSandbox")
		// if we have a sandbox, try unpausing it to see if it is still alive
		if sb != nil {
			// Unpause will often fail, because evictors
			// are likely to prefer to evict paused
			// sandboxes rather than inactive sandboxes.
			// Thus, if this fails, we'll try to handle it
			// by just creating a new sandbox.
			t2 := common.T0("LambdaInstance-WaitSandbox-Unpause")
			if err := sb.Unpause(); err != nil {
				f.printf("discard sandbox %s due to Unpause error: %v", sb.ID(), err)
				sb = nil
			}
			t2.T1()
			
		}

		// if we don't already have a Sandbox, create one, and
		// HTTP proxy over the channel
		if sb == nil {
			sb = nil

			if f.lmgr.ImportCache != nil && f.rtType == common.RT_PYTHON {
				scratchDir := f.lmgr.scratchDirs.Make(f.name)

				// we don't specify parent SB, because ImportCache.Create chooses it for us
				sb, err = f.lmgr.ImportCache.Create(f.lmgr.sbPool, true, linst.codeDir, scratchDir, linst.meta, f.rtType)
				if err != nil {
					f.printf("failed to get Sandbox from import cache")
					sb = nil
				}
			}

			log.Printf("Creating new sandbox")

			// import cache is either disabled or it failed
			if sb == nil {
				t2 := common.T0("LambdaInstance-WaitSandbox-NoImportCache")
				scratchDir := f.lmgr.scratchDirs.Make(f.name)
				sb, err = f.lmgr.sbPool.Create(nil, true, linst.codeDir, scratchDir, linst.meta, f.rtType)
				t2.T1()
			}

			if err != nil {
				linst.TrySendError(req, http.StatusInternalServerError, "could not create Sandbox: "+err.Error()+"\n", nil)
				f.doneChan <- req
				continue // wait for another request before retrying
			}
		}
		t.T1()

		// below here, we're guaranteed (1) sb != nil, (2) proxy != nil, (3) sb is unpaused

		// serve until we incoming queue is empty
		t = common.T0("LambdaInstance-ServeRequests")
		for req != nil {
			f.printf("Forwarding request to sandbox")

			t2 := common.T0("LambdaInstance-RoundTrip")

			// get response from sandbox
			url := "http://root" + req.r.RequestURI
			httpReq, err := http.NewRequest(req.r.Method, url, req.r.Body)
			if err != nil {
				linst.TrySendError(req, http.StatusInternalServerError, "Could not create NewRequest: "+err.Error(), sb)
			} else {
				resp, err := sb.Client().Do(httpReq)

				// copy response out
				if err != nil {
					linst.TrySendError(req, http.StatusBadGateway, "RoundTrip failed: "+err.Error()+"\n", sb)
				} else {
					// copy headers
					// (adapted from copyHeaders: https://go.dev/src/net/http/httputil/reverseproxy.go)
					for k, vv := range resp.Header {
						for _, v := range vv {
							req.w.Header().Add(k, v)
						}
					}
					req.w.WriteHeader(resp.StatusCode)

					// copy body
					if _, err := io.Copy(req.w, resp.Body); err != nil {
						// already used WriteHeader, so can't use that to surface on error anymore
						msg := "reading lambda response failed: "+err.Error()+"\n"
						f.printf("error: "+msg)
						linst.TrySendError(req, 0, msg, sb)
					}

					resp.Body.Close()
				}
			}

			// notify instance that we're done
			t2.T1()
			v := int(t2.Milliseconds)
			req.execMs = v
			f.doneChan <- req

			// check whether we should shutdown (non-blocking)
			select {
			case killed := <-linst.killChan:
				rtLog := sb.GetRuntimeLog()
				sb.Destroy("Lambda instance kill signal received")

				log.Printf("Stopped sandbox")

				if common.Conf.Log_output {
					if rtLog != "" {
						log.Printf("Runtime output is:")

						for _, line := range strings.Split(rtLog, "\n") {
							log.Printf("   %s", line)
						}
					}
				}

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

		if sb != nil {
			if err := sb.Pause(); err != nil {
				f.printf("discard sandbox %s due to Pause error: %v", sb.ID(), err)
				sb = nil
			}
		}

		t.T1()
	}
}

func (linst *LambdaInstance) TrySendError(req *Invocation, statusCode int, msg string, sb sandbox.Sandbox) {
	if statusCode > 0 {
		req.w.WriteHeader(statusCode)
	}

	var err error
	if sb != nil {
		_, err = req.w.Write([]byte(msg+"\nSandbox State: "+sb.DebugString()+"\n"))
	} else {
		_, err = req.w.Write([]byte(msg+"\n"))
	}

	if err != nil {
		linst.lfunc.printf("TrySendError failed: %s\n", err.Error())
	}
}

// AsyncKill signals the instance to die, return chan that can be used to block
// until it's done
func (linst *LambdaInstance) AsyncKill() chan bool {
	done := make(chan bool)
	linst.killChan <- done
	return done
}
