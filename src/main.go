package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	docker "github.com/fsouza/go-dockerclient"
	dutil "github.com/open-lambda/open-lambda/ol/sandbox/dockerutil"

	"github.com/open-lambda/open-lambda/ol/common"
	"github.com/open-lambda/open-lambda/ol/server"
	"github.com/urfave/cli"
)

var client *docker.Client

func getOlPath(ctx *cli.Context) (string, error) {
	olPath := ctx.String("path")
	if olPath == "" {
		olPath = "default-ol"
	}
	return filepath.Abs(olPath)
}

func getOlWorkerProc(ctx *cli.Context) (int, error) {
	// Check all running ol procs, substracting "this" one.
	// Return:
	//	pid:
	// 		Return the process id if one proc is working.
	//		Return 0 if no proc is working.
	//		Return < 0 as error code.
	//	err:
	//		Error message of all kind.

	// Assumption:
	//	1. At most one ol worker is running in the system.
	//  2. This process is the only other ol admin process that could be running.
	// TODO: Assumption 2 is too strong as there could be many `ol status` running.

	// Get the running pid whose program name is `ol` (at least one - this one)
	out, err := exec.Command("ps", "-C", "ol", "-o", "pid=").Output()
	if err != nil {
		return -1, err
	}
	if len(out) == 0 {
		err := fmt.Errorf("No output from commanad: ps -C ol -i pid=")
		return -1, err
	}

	procStr := strings.Split(strings.TrimSpace(strings.Trim(string(out), "\n")), "\n")

	// Maps the pid strings to int, except for `this` process
	this := os.Getpid()
	procs := make([]int, 0, len(procStr))
	for _, v := range procStr {
		if len(v) == 0 {
			continue
		}

		pid, err := strconv.Atoi(strings.Trim(v, " "))
		if err != nil {
			return -1, err
		}
		if pid != this {
			procs = append(procs, pid)
		}
	}

	// Assert there are at most one proc running then return if we have one.
	// TODO: Multiple `ol status` could run in parallel.
	// 		 Should distinguish these worker from the ol worker process.
	if len(procs) > 1 {
		return -2, fmt.Errorf("More than one ol process is running: %s", procs)
	}

	// No ol worker. Return successfully with pid = 0 (no worker).
	if len(procs) == 0 {
		return 0, nil
	}

	// Return the ol worker pid.
	return procs[0], nil
}

func getOlNumSB(olPath string) int {
	// Return the sandbox in the worker/scratch to estimate the progress of kill
	// Error with no such dir returns -1
	scratch := filepath.Join(olPath, "worker", "scratch")
	files, err := ioutil.ReadDir(scratch)
	if err != nil {
		return -1
	}
	return len(files)
}

func getKillTarget(olPath string, createIfNotExist bool) (int, error) {
	// create killPath if not exist.
	// return the number of SB needs to be killed initially.

	// if killPath exist, return the value
	killPath := filepath.Join(common.Conf.Worker_dir, "worker.kill")

	if _, err := os.Stat(killPath); err == nil {

		s, err := ioutil.ReadFile(killPath)
		if err != nil {
			return -1, err
		}

		numSB, err := strconv.Atoi(string(s))
		if err != nil {
			return -1, err
		}
		return numSB, err
	}

	if !createIfNotExist {
		return -1, fmt.Errorf("worker.kill does not exist")
	}

	// killPath not exist, create log file and store the number of sandboxes to kill
	_, err := os.Create(killPath)
	if err != nil {
		return -1, err
	}

	numSB := getOlNumSB(olPath)
	if numSB < 0 {
		// No such directory exist or no numSB.
		return 0, nil
	}
	if err := ioutil.WriteFile(killPath, []byte(strconv.Itoa(numSB)), 0644); err != nil {
		return -1, err
	}
	return numSB, nil
}

func initOLDir(olPath string) (err error) {
	// Add a temporary lock for the init ol dir
	fmt.Printf("Init OL dir at %v\n", olPath)
	if err := os.Mkdir(olPath, 0700); err != nil {
		return err
	}

	// Add a lock file and delete it once initOLDir is done.
	lockPath := filepath.Join(olPath, "lock")
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		return fmt.Errorf("%v is creating by another process (lock file %v exists)", olPath, lockPath)
	}

	f, err := os.Create(lockPath)
	if err != nil {
		return err
	}

	defer func() {
		if _, err := os.Stat(lockPath); os.IsNotExist(err) {
			// Lock file already removed, possibly manually
			return
		}
		err = f.Close()
		if err != nil {
			return
		}
		err := os.Remove(lockPath)
		if err != nil {
			return
		}
	}()

	if err := common.LoadDefaults(olPath); err != nil {
		return err
	}

	confPath := filepath.Join(olPath, "config.json")
	if err := common.SaveConf(confPath); err != nil {
		return err
	}

	if err := os.Mkdir(common.Conf.Worker_dir, 0700); err != nil {
		return err
	}

	if err := os.Mkdir(common.Conf.Registry, 0700); err != nil {
		return err
	}

	// create a base directory to run sock handlers
	base := common.Conf.SOCK_base_path
	fmt.Printf("Create lambda base at %v (may take several minutes)\n", base)
	err = dutil.DumpDockerImage(client, "lambda", base)
	if err != nil {
		return err
	}

	if err := os.Mkdir(path.Join(base, "handler"), 0700); err != nil {
		return err
	}

	if err := os.Mkdir(path.Join(base, "host"), 0700); err != nil {
		return err
	}

	if err := os.Mkdir(path.Join(base, "packages"), 0700); err != nil {
		return err
	}

	// need this because Docker containers don't have a dns server in /etc/resolv.conf
	dnsPath := filepath.Join(base, "etc", "resolv.conf")
	if err := ioutil.WriteFile(dnsPath, []byte("nameserver 8.8.8.8\n"), 0644); err != nil {
		return err
	}

	path := filepath.Join(base, "dev", "null")
	if err := exec.Command("mknod", "-m", "0644", path, "c", "1", "3").Run(); err != nil {
		return err
	}

	path = filepath.Join(base, "dev", "random")
	if err := exec.Command("mknod", "-m", "0644", path, "c", "1", "8").Run(); err != nil {
		return err
	}

	path = filepath.Join(base, "dev", "urandom")
	if err := exec.Command("mknod", "-m", "0644", path, "c", "1", "9").Run(); err != nil {
		return err
	}

	fmt.Printf("Working Directory: %s\n\n", olPath)
	fmt.Printf("Worker Defaults: \n%s\n\n", common.DumpConfStr())
	fmt.Printf("You may modify the defaults here: %s\n\n", confPath)
	fmt.Printf("You may now start a server using the \"ol worker\" command\n")

	return nil
}

// newOL corresponds to the "new" command of the admin tool.
func newOL(ctx *cli.Context) error {
	olPath, err := getOlPath(ctx)
	if err != nil {
		return err
	}

	return initOLDir(olPath)
}

// status corresponds to the "status" command of the admin tool.
func status(ctx *cli.Context) error {
	olPath, err := getOlPath(ctx)
	if err != nil {
		return err
	}

	fmt.Printf("Worker Ping:\n")
	err = common.LoadConf(filepath.Join(olPath, "config.json"))
	if err != nil {
		return err
	}

	if totalNum, err := getKillTarget(olPath, false); err != nil {
		if currNum := getOlNumSB(olPath); currNum != -1 {
			fmt.Printf("%v/%v\n", currNum, totalNum)
			return nil
		}
	}

	url := fmt.Sprintf("http://localhost:%s/status", common.Conf.Worker_port)
	response, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("  Could not send GET to %s\n", url)
	}
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("  Failed to read body from GET to %s\n", url)
	}
	fmt.Printf("  %s => %s [%s]\n", url, body, response.Status)
	fmt.Printf("\n")

	return nil
}

// modify the config.json file based on settings from cmdline: -o opt1=val1,opt2=val2,...
//
// apply changes in optsStr to config from confPath, saving result to overridePath
func overrideOpts(confPath, overridePath, optsStr string) error {
	b, err := ioutil.ReadFile(confPath)
	if err != nil {
		return err
	}
	conf := make(map[string]interface{})
	if err := json.Unmarshal(b, &conf); err != nil {
		return err
	}

	opts := strings.Split(optsStr, ",")
	for _, opt := range opts {
		parts := strings.Split(opt, "=")
		if len(parts) != 2 {
			return fmt.Errorf("Could not parse key=val: '%s'", opt)
		}
		keys := strings.Split(parts[0], ".")
		val := parts[1]

		c := conf
		for i := 0; i < len(keys)-1; i++ {
			sub, ok := c[keys[i]]
			if !ok {
				return fmt.Errorf("key '%s' not found", keys[i])
			}
			switch v := sub.(type) {
			case map[string]interface{}:
				c = v
			default:
				return fmt.Errorf("%s refers to a %T, not a map", keys[i], c[keys[i]])
			}

		}

		key := keys[len(keys)-1]
		prev, ok := c[key]
		if !ok {
			return fmt.Errorf("invalid option: '%s'", key)
		}
		switch prev.(type) {
		case string:
			c[key] = val
		case float64:
			c[key], err = strconv.Atoi(val)
			if err != nil {
				return err
			}
		case bool:
			if strings.ToLower(val) == "true" {
				c[key] = true
			} else if strings.ToLower(val) == "false" {
				c[key] = false
			} else {
				return fmt.Errorf("'%s' for %s not a valid boolean value", val, key)
			}
		default:
			return fmt.Errorf("config values of type %T (%s) must be edited manually in the config file ", prev, key)
		}
	}

	// save back config
	s, err := json.MarshalIndent(conf, "", "\t")
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(overridePath, s, 0644); err != nil {
		return err
	}
	return nil
}

// workers corresponds to the "workers" command of the admin tool.
//
// The JSON config in the cluster template directory will be populated for each
// worker, and their pid will be written to the log directory. worker_exec will
// be called to run the worker processes.
func worker(ctx *cli.Context) error {
	olPath, err := getOlPath(ctx)
	if err != nil {
		return err
	}

	// if `./ol new` not previously run, do that init now
	if _, err := os.Stat(olPath); os.IsNotExist(err) {
		fmt.Printf("no OL directory found at %s\n", olPath)
		if err := initOLDir(olPath); err != nil {
			return err
		}
	}

	// if `lock` file exist, a `ol new` is running.
	lockPath := filepath.Join(olPath, "lock")
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		// Check if it is created by another running proc.
		// If not, then we have a dangling lock file.
		pid, err := getOlWorkerProc(ctx)
		if pid > 0 {
			return fmt.Errorf("lock file %v exists: %v is creating by another process %d", olPath, lockPath, pid)
		}
		if err != nil && pid == -2 {
			// Multiple process running.
			return fmt.Errorf("lock file %v exists: %v is creating by another process", olPath, lockPath)
		}
		if err != nil {
			return fmt.Errorf("Dangling lock file %v exists: %v is supposed to be creating by another process, but that process is not found", olPath, lockPath)
		}

	}
	fmt.Printf("using existing OL directory at %s\n", olPath)

	confPath := ctx.String("file")
	if confPath == "" {
		confPath = filepath.Join(olPath, "config.json")
	}
	confPath, err = filepath.Abs(confPath)
	if err != nil {
		return fmt.Errorf("load config file with error: %s", err)
	}
	fmt.Printf("using config file at %s\n", confPath)

	overrides := ctx.String("options")
	if overrides != "" {
		overridesPath := confPath + ".overrides"
		err = overrideOpts(confPath, overridesPath, overrides)
		if err != nil {
			return err
		}
		confPath = overridesPath
	}

	if err := common.LoadConf(confPath); err != nil {
		return err
	}

	pidPath := filepath.Join(common.Conf.Worker_dir, "worker.pid")
	if _, err := os.Stat(pidPath); err == nil {
		pid, _ := getOlWorkerProc(ctx)
		if pid > 0 {
			return fmt.Errorf("previous worker is running: %v", pid)
		}
		return fmt.Errorf("previous worker may be running (%s already exists)", pidPath)
	}

	// should we run as a background process?
	detach := ctx.Bool("detach")

	if detach {
		// stdout+stderr both go to log
		logPath := filepath.Join(olPath, "worker.out")
		f, err := os.Create(logPath)
		if err != nil {
			return err
		}
		attr := os.ProcAttr{
			Files: []*os.File{nil, f, f},
		}
		cmd := []string{}
		for _, arg := range os.Args {
			if arg != "-d" && arg != "--detach" {
				cmd = append(cmd, arg)
			}
		}
		binPath, err := exec.LookPath(os.Args[0])
		if err != nil {
			return err
		}
		proc, err := os.StartProcess(binPath, cmd, &attr)
		if err != nil {
			return err
		}

		died := make(chan error)
		go func() {
			_, err := proc.Wait()
			died <- err
		}()

		fmt.Printf("Starting worker: pid=%d, port=%s, log=%s\n", proc.Pid, common.Conf.Worker_port, logPath)

		var ping_err error

		for i := 0; i < 300; i++ {
			// check if it has died
			select {
			case err := <-died:
				if err != nil {
					return err
				}
				return fmt.Errorf("worker process %d does not a appear to be running, check worker.out", proc.Pid)
			default:
			}

			// is the worker still alive?
			err := proc.Signal(syscall.Signal(0))
			if err != nil {

			}

			// is it reachable?
			url := fmt.Sprintf("http://localhost:%s/pid", common.Conf.Worker_port)
			response, err := http.Get(url)
			if err != nil {
				ping_err = err
				time.Sleep(100 * time.Millisecond)
				continue
			}
			defer response.Body.Close()

			// are we talking with the expected PID?
			body, err := ioutil.ReadAll(response.Body)
			pid, err := strconv.Atoi(strings.TrimSpace(string(body)))
			if err != nil {
				return fmt.Errorf("/pid did not return an int :: %s", err)
			}

			if pid == proc.Pid {
				fmt.Printf("ready\n")
				return nil // server is started and ready for requests
			} else {
				return fmt.Errorf("expected PID %v but found %v (port conflict?)", proc.Pid, pid)
			}
		}

		return fmt.Errorf("worker still not reachable after 30 seconds :: %s", ping_err)
	} else {
		if err := server.Main(); err != nil {
			return err
		}
	}

	return fmt.Errorf("this code should not be reachable!")
}

// kill corresponds to the "kill" command of the admin tool.
func kill(ctx *cli.Context) error {

	// Check if there is running worker in the system.
	pid, err := getOlWorkerProc(ctx)
	if err != nil {
		return err
	}

	if pid == 0 {
		fmt.Printf("No worker running.\n")
		return nil
	}

	olPath, err := getOlPath(ctx)
	if err != nil {
		return err
	}

	// locate worker.pid, use it to get worker's PID
	configPath := filepath.Join(olPath, "config.json")

	if err := common.LoadConf(configPath); err != nil {
		// If config.json cannot be found, try to see if there are any running process.
		return err
	}

	data, err := ioutil.ReadFile(filepath.Join(common.Conf.Worker_dir, "worker.pid"))
	if err != nil {
		return err
	}
	pidstr := string(data)
	pid, err = strconv.Atoi(pidstr)
	if err != nil {
		return err
	}

	fmt.Printf("Kill worker process with PID %d\n", pid)
	p, err := os.FindProcess(pid)
	if err != nil {
		fmt.Printf("%s\n", err.Error())
		fmt.Printf("Failed to find worker process with PID %d.  May require manual cleanup.\n", pid)
	}
	if err := p.Signal(syscall.SIGINT); err != nil {
		fmt.Printf("%s\n", err.Error())
		fmt.Printf("Failed to kill process with PID %d.  May require manual cleanup.\n", pid)
	}

	// write kill log with the number of sb in scratch if not exist
	totalSB, err := getKillTarget(olPath, true)
	if err != nil {
		fmt.Printf("Get kill log error: %s. Proceed without showing kill progress.", err)
	}

	for i := 0; ; i++ {
		err := p.Signal(syscall.Signal(0))
		if err != nil {
			return nil // good, process must have stopped
		}
		time.Sleep(100 * time.Millisecond)

		if i%10 == 0 {
			numSB, err := getKillTarget(olPath, false)
			if err == nil {
				remainSB := totalSB - numSB
				fmt.Printf("Progress: %v/%v\n", remainSB, totalSB)
			}
		}
	}

	return fmt.Errorf("worker didn't stop after 30s")
}

// main runs the admin tool
func main() {
	if c, err := docker.NewClientFromEnv(); err != nil {
		log.Fatal("failed to get docker client: ", err)
	} else {
		client = c
	}

	cli.CommandHelpTemplate = `NAME:
   {{.HelpName}} - {{if .Description}}{{.Description}}{{else}}{{.Usage}}{{end}}
USAGE:
   {{if .UsageText}}{{.UsageText}}{{else}}{{.HelpName}} command{{if .VisibleFlags}} [command options]{{end}} {{if .ArgsUsage}}{{.ArgsUsage}}{{else}}[arguments...]{{end}}{{end}}
COMMANDS:{{range .VisibleCategories}}{{if .Name}}
   {{.Name}}:{{end}}{{range .VisibleCommands}}
     {{join .Names ", "}}{{"\t"}}{{.Usage}}{{end}}
{{end}}{{if .VisibleFlags}}
OPTIONS:
   {{range .VisibleFlags}}{{.}}
   {{end}}{{end}}
`
	app := cli.NewApp()
	app.Usage = "Admin tool for Open-Lambda"
	app.UsageText = "ol COMMAND [ARG...]"
	app.ArgsUsage = "ArgsUsage"
	app.EnableBashCompletion = true
	app.HideVersion = true
	pathFlag := cli.StringFlag{
		Name:  "path, p",
		Usage: "Path location for OL environment",
	}
	app.Commands = []cli.Command{
		cli.Command{
			Name:        "new",
			Usage:       "Create a OpenLambda environment",
			UsageText:   "ol new [--path=PATH]",
			Description: "A cluster directory of the given name will be created with internal structure initialized.",
			Flags:       []cli.Flag{pathFlag},
			Action:      newOL,
		},
		cli.Command{
			Name:        "worker",
			Usage:       "Start one OL server",
			UsageText:   "ol worker [--path=NAME] [--detach] [--file=CONFIGPATH]",
			Description: "Start a lambda server.",
			Flags: []cli.Flag{
				pathFlag,
				cli.StringFlag{
					Name:  "options, o",
					Usage: "Override options in config.json: -o opt1=val1,opt2=val2,opt3.subopt31=val3. See config.json for more detail.",
				},
				cli.BoolFlag{
					Name:  "detach, d",
					Usage: "Run worker in background",
				},
				cli.StringFlag{
					Name:  "file, f",
					Usage: "Run worker with override config.json file",
				},
			},
			Action: worker,
		},
		cli.Command{
			Name:        "status",
			Usage:       "get worker status",
			UsageText:   "ol status [--path=NAME]",
			Description: "If no cluster name is specified, number of containers of each cluster is printed; otherwise the connection information for all containers in the given cluster will be displayed.",
			Flags:       []cli.Flag{pathFlag},
			Action:      status,
		},
		cli.Command{
			Name:      "kill",
			Usage:     "Kill containers and processes in a cluster",
			UsageText: "ol kill [--path=NAME] [--async]",
			Flags: []cli.Flag{
				pathFlag,
			},
			Action: kill,
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
