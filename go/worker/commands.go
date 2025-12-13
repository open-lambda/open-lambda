package worker

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/open-lambda/open-lambda/go/common"
	"github.com/open-lambda/open-lambda/go/worker/event"

	"github.com/urfave/cli/v2"
)

// performs an HTTP GET request to the worker over its UDS
func udsGet(requestPath string) (*http.Response, error) {
	sockPath := filepath.Join(common.Conf.Worker_dir, "ol.sock")

	// create a transport that dials the socket
	tr := &http.Transport{}
	tr.DialContext = func(ctx context.Context, _, _ string) (net.Conn, error) {
		var d net.Dialer
		return d.DialContext(ctx, "unix", sockPath)
	}
	client := &http.Client{Transport: tr, Timeout: 2 * time.Second}

	// perform HTTP GET request with custom client
	url := "http://unix" + requestPath
	return client.Get(url)
}

// initCmd corresponds to the "init" command of the admin tool.
func initCmd(ctx *cli.Context) error {
	olPath, err := common.GetOlPath(ctx)
	if err != nil {
		return err
	}

	if err := common.LoadDefaults(olPath); err != nil {
		return err
	}

	if err := initOLDir(olPath, ctx.String("image"), ctx.Bool("newbase")); err != nil {
		return err
	}
	fmt.Printf("\nYou may optionally modify the defaults here: %s\n\n",
		filepath.Join(olPath, "config.json"))
	fmt.Printf("Next start a worker using the \"ol worker up\" command.\n")
	return nil
}

// upCmd corresponds to the "up" command of the admin tool.
// If it returns a non-nil error at any point, `urfave/cli` will
// automatically catch it, print the error message to stderr.
// Then we exit the program and return to main, where we call os.Exit(1)
func upCmd(ctx *cli.Context) error {
	// get path of worker files
	olPath, err := common.GetOlPath(ctx)

	if err != nil {
		return err
	}

	// PREP STEP 1: make sure we have a worker directory
	if _, err := os.Stat(olPath); os.IsNotExist(err) {
		// need to init worker dir first
		fmt.Printf("Did not find OL directory at %s\n", olPath)
		if err := common.LoadDefaults(olPath); err != nil {
			return err
		}

		if err := initOLDir(olPath, ctx.String("image"), false); err != nil {
			return err
		}
	}

	// PREP STEP 2: load config file and apply any command-line overrides
	confPath := filepath.Join(olPath, "config.json")
	overrides := ctx.String("options")
	if overrides != "" {
		overridesPath := confPath + ".overrides"
		if err := overrideOpts(confPath, overridesPath, overrides); err != nil {
			return err
		}
		confPath = overridesPath
	}
	if err := common.LoadGlobalConfig(confPath); err != nil {
		return err
	}

	// Rootless preflight info/warnings
	preflightRootless()

	// PREP STEP 3: ensure Open Lambda is in the StoppedClean state
	if err := bringToStoppedClean(olPath); err != nil {
		return err
	}

	// should we run as a background process?
	detach := ctx.Bool("detach")
	if !detach && ctx.Bool("rootless") {
		fmt.Println("NOTE: --rootless currently applied only in --detach mode.")
	}

	if detach {
		// stdout+stderr both go to log
		logPath := filepath.Join(olPath, "worker.out")
		// creates a worker.out file
		f, err := os.Create(logPath)
		if err != nil {
			return err
		}

		uid := os.Getuid()
		gid := os.Getgid()

		// holds attributes that will be used when os.StartProcess.
		// legacy used CLONE_NEWNS so mounts don't spam systemd.
		// now, if --rootless, also create a user+UTS namespace and map uid/gid -> 0 (fake root).
		attr := os.ProcAttr{
			Files: []*os.File{nil, f, f},
		}
		if ctx.Bool("rootless") {
			attr.Sys = &syscall.SysProcAttr{
				Unshareflags: syscall.CLONE_NEWUSER | syscall.CLONE_NEWNS | syscall.CLONE_NEWUTS,
				UidMappings: []syscall.SysProcIDMap{
					{ContainerID: 0, HostID: uid, Size: 1},
				},
				GidMappingsEnableSetgroups: false,
				GidMappings: []syscall.SysProcIDMap{
					{ContainerID: 0, HostID: gid, Size: 1},
				},
			}
		} else {
			// Legacy: do NOT unshare mount ns alone (Ubuntu EPERM). No unshare in legacy.
			attr.Sys = &syscall.SysProcAttr{}
		}

		// build args for child (strip -d/--detach)
		cmd := []string{}
		for _, arg := range os.Args {
			if arg != "-d" && arg != "--detach" {
				cmd = append(cmd, arg)
			}
		}
		// looks for ./ol path
		binPath, err := exec.LookPath(os.Args[0])
		if err != nil {
			return err
		}
		if abs, err := filepath.Abs(binPath); err == nil {
			binPath = abs
		}

		// start the worker process
		fmt.Printf("Starting worker in %s and waiting until it's ready.\n", olPath)
		proc, err := os.StartProcess(binPath, cmd, &attr)
		if err != nil {
			return err
		}

		// died is error message
		died := make(chan error)
		go func() {
			_, err := proc.Wait()
			died <- err
		}()

		fmt.Printf("\tPID: %d\n\tPort: %s\n\tLog File: %s\n", proc.Pid, common.Conf.Worker_port, logPath)
		fmt.Printf("\tRootless: %v (uid=%d gid=%d)\n", ctx.Bool("rootless"), uid, gid)

		var pingErr error

		for i := 0; i < 300; i++ {
			// check if it has died
			select {
			case err := <-died:
				if err != nil {
					return err
				}
				return fmt.Errorf("worker process %d does not appear to be running; check worker.out", proc.Pid)
			default:
			}

			// is the worker still alive?
			err := proc.Signal(syscall.Signal(0))
			if err != nil {

			}

			// is it reachable?
			response, err := udsGet("/pid")
			if err != nil {
				pingErr = err
				time.Sleep(100 * time.Millisecond)
				continue
			}
			defer response.Body.Close()

			// are we talking with the expected PID?
			body, err := io.ReadAll(response.Body)
			pid, err := strconv.Atoi(strings.TrimSpace(string(body)))
			if err != nil {
				return fmt.Errorf("/pid did not return an int:  %s", err)
			}

			if pid == proc.Pid {
				fmt.Printf("Ready!\n")
				return nil // server is started and ready for requests
			}

			return fmt.Errorf("pid mismatch: expected %v but found %v (another worker running?)", proc.Pid, pid)
		}

		return fmt.Errorf("worker still not reachable after 30 seconds: %w", pingErr)
	}

	// server had clean shutdown
	return event.Main()
}

func preflightRootless() {
	// Warn if Ubuntu blocks unprivileged user namespaces
	if b, err := os.ReadFile("/proc/sys/kernel/unprivileged_userns_clone"); err == nil {
		if strings.TrimSpace(string(b)) != "1" {
			fmt.Println("WARNING: rootless user namespaces appear disabled (kernel.unprivileged_userns_clone=0).")
			fmt.Println("         To enable: sudo sysctl kernel.unprivileged_userns_clone=1")
		}
	}
	// Note systemd presence (used for rootless cgroup delegation)
	if _, err := os.Stat("/run/systemd/system"); err == nil {
		fmt.Println("INFO: systemd detected (cgroup v2 delegation likely available).")
	} else {
		fmt.Println("INFO: systemd not detected; rootless cgroup delegation may be unavailable.")
	}
}

// statusCmd corresponds to the "status" command of the admin tool.
func statusCmd(ctx *cli.Context) error {
	olPath, err := common.GetOlPath(ctx)
	if err != nil {
		return err
	}
	err = common.LoadGlobalConfig(filepath.Join(olPath, "config.json"))
	if err != nil {
		return err
	}

	fmt.Printf("Worker Ping:\n")
	url := fmt.Sprintf("http://localhost:%s/status", common.Conf.Worker_port)
	response, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("could not send GET to %s", url)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("failed to read body from GET to %s", url)
	}
	fmt.Printf("  %s => %s [%s]\n", url, body, response.Status)
	fmt.Printf("\n")

	return nil
}

// downCmd corresponds to the "down" command of the admin tool.
func downCmd(ctx *cli.Context) error {
	olPath, err := common.GetOlPath(ctx)
	if err != nil {
		return err
	}
	err = common.LoadGlobalConfig(filepath.Join(olPath, "config.json"))
	if err != nil {
		return err
	}
	return bringToStoppedClean(olPath)
}

// cleanupCmd corresponds to the "force-cleanup" command of the admin tool.
func cleanupCmd(ctx *cli.Context) error {
	olPath, err := common.GetOlPath(ctx)
	if err != nil {
		return err
	}
	err = common.LoadGlobalConfig(filepath.Join(olPath, "config.json"))
	if err != nil {
		return fmt.Errorf("failed to load OL config: %s", err)
	}
	return bringToStoppedClean(olPath)
}

// WorkerCommands returns a list of CLI commands for the worker.
func WorkerCommands() []*cli.Command {
	pathFlag := cli.StringFlag{
		Name:    "path",
		Aliases: []string{"p"},
		Usage:   "Path location for OL environment",
	}
	dockerImgFlag := cli.StringFlag{
		Name:    "image",
		Aliases: []string{"i"},
		Usage:   "Name of Docker image to use for base",
	}

	cmds := []*cli.Command{
		&cli.Command{
			Name:        "init",
			Usage:       "Create an OL worker environment, including default config and dump of base image",
			UsageText:   "ol init [OPTIONS...]",
			Description: "A cluster directory of the given name will be created with internal structure initialized.",
			Flags: []cli.Flag{
				&pathFlag,
				&dockerImgFlag,
				&cli.BoolFlag{
					Name:    "newbase",
					Aliases: []string{"b"},
					Usage:   "Overwrite base directory if it already exists",
				},
			},
			Action: initCmd,
		},
		&cli.Command{
			Name:        "up",
			Usage:       "Start an OL worker process (automatically calls 'init' and uses default if that wasn't already done)",
			UsageText:   "ol up [OPTIONS...] [--detach]",
			Description: "Start an OL worker.",
			Flags: []cli.Flag{
				&pathFlag,
				&dockerImgFlag,
				&cli.StringFlag{
					Name:    "options",
					Aliases: []string{"o"},
					Usage:   "Override options with: -o opt1=val1,opt2=val2/opt3.subopt31=val3",
				},
				&cli.BoolFlag{
					Name:    "detach",
					Aliases: []string{"d"},
					Usage:   "Run worker in background",
				},
				&cli.BoolFlag{
					Name:  "rootless",
					Usage: "Enable rootless user namespace for worker (recommended)",
					Value: true,
				},
			},
			Action: upCmd,
		},
		&cli.Command{
			Name:      "down",
			Usage:     "Kill containers and processes of the worker",
			UsageText: "ol down [OPTIONS...]",
			Flags:     []cli.Flag{&pathFlag},
			Action:    downCmd,
		},
		&cli.Command{
			Name:        "status",
			Usage:       "check status of an OL worker process",
			UsageText:   "ol status [OPTIONS...]",
			Description: "If no cluster name is specified, number of containers of each cluster is printed; otherwise the connection information for all containers in the given cluster will be displayed.",
			Flags:       []cli.Flag{&pathFlag},
			Action:      statusCmd,
		},
		&cli.Command{
			Name:      "force-cleanup",
			Usage:     "Developer use only.  Cleanup cgroups and mount points (only needed when OL halted unexpectedly or there's a bug)",
			UsageText: "ol force-cleanup [OPTIONS...]",
			Flags:     []cli.Flag{&pathFlag},
			Action:    cleanupCmd,
		},
	}

	return cmds
}
