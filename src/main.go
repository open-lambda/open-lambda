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
	"github.com/open-lambda/open-lambda/ol/boss"
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

func getBossOlPath(ctx *cli.Context) (string, error) {
	olPath := ctx.String("path")
	if olPath == "" {
		olPath = "default-boss-ol"
	}
	return filepath.Abs(olPath)
}

func initOLDir(olPath string) (err error) {
	fmt.Printf("Init OL dir at %v\n", olPath)
	if err := os.Mkdir(olPath, 0700); err != nil {
		return err
	}

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

// newBossOL corresponses to the "new-boss" command of the admin tool.
func newBossOL(ctx *cli.Context) error {
	olPath, err := getBossOlPath(ctx)
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
	} else {
		fmt.Printf("using existing OL directory at %s\n", olPath)
	}

	confPath := filepath.Join(olPath, "config.json")
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
	olPath, err := getOlPath(ctx)
	if err != nil {
		return err
	}

	// locate worker.pid, use it to get worker's PID
	configPath := filepath.Join(olPath, "config.json")
	if err := common.LoadConf(configPath); err != nil {
		return err
	}
	data, err := ioutil.ReadFile(filepath.Join(common.Conf.Worker_dir, "worker.pid"))
	if err != nil {
		return err
	}
	pidstr := string(data)
	pid, err := strconv.Atoi(pidstr)
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

	for i := 0; i < 300; i++ {
		err := p.Signal(syscall.Signal(0))
		if err != nil {
			return nil // good, process must have stopped
		}
		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("worker didn't stop after 30s")
}

func gcp_test(ctx *cli.Context) error {
	boss.GCPBossTest()
	return nil
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
			Name:	     "new-boss",
			Usage:       "Create a new Boss",
			UsageText:   "ol new-boss [--path=PATH]",
			Description: "Testing Purposes Right Now",
			Flags:       []cli.Flag{pathFlag},
			Action:      newBossOL,
		},
		cli.Command{
			Name:        "worker",
			Usage:       "Start one OL server",
			UsageText:   "ol worker [--path=NAME] [--detach]",
			Description: "Start a lambda server.",
			Flags: []cli.Flag{
				pathFlag,
				cli.StringFlag{
					Name:  "options, o",
					Usage: "Override options with: -o opt1=val1,opt2=val2/opt3.subopt31=val3",
				},
				cli.BoolFlag{
					Name:  "detach, d",
					Usage: "Run worker in background",
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
			UsageText: "ol kill [--path=NAME]",
			Flags:     []cli.Flag{pathFlag},
			Action:    kill,
		},
		cli.Command{
			Name:      "gcp-test",
			Usage:     "Start a GCP VM running the OL worker",
			UsageText: "ol gcp-test",
			Flags:     []cli.Flag{},
			Action:    gcp_test,
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
