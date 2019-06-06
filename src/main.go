package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	docker "github.com/fsouza/go-dockerclient"
	dutil "github.com/open-lambda/open-lambda/ol/sandbox/dockerutil"

	"github.com/open-lambda/open-lambda/ol/config"
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

func initOLDir(olPath string) (err error) {
	fmt.Printf("Init OL dir at %v\n", olPath)
	if err := os.Mkdir(olPath, 0700); err != nil {
		return err
	}

	if err := config.LoadDefaults(olPath); err != nil {
		return err
	}

	if err := config.Save(filepath.Join(olPath, "config.json")); err != nil {
		return err
	}

	if err := os.Mkdir(config.Conf.Worker_dir, 0700); err != nil {
		return err
	}

	if err := os.Mkdir(config.Conf.Registry, 0700); err != nil {
		return err
	}

	// create a base directory to run sock handlers
	fmt.Printf("Create lambda base at %v (may take several minutes)\n", config.Conf.SOCK_base_path)
	err = dutil.DumpDockerImage(client, "lambda", config.Conf.SOCK_base_path)
	if err != nil {
		return err
	}

	// need this because Docker containers don't have a dns server in /etc/resolv.conf
	dnsPath := filepath.Join(config.Conf.SOCK_base_path, "etc", "resolv.conf")
	if err := ioutil.WriteFile(dnsPath, []byte("nameserver 8.8.8.8\n"), 0644); err != nil {
		return err
	}

	fmt.Printf("Working Directory: %s\n\n", olPath)
	fmt.Printf("Worker Defaults: \n%s\n\n", config.DumpStr())
	fmt.Printf("You may now start a server using the \"worker\" command\n")

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
	err = config.LoadFile(filepath.Join(olPath, "config.json"))
	if err != nil {
		return err
	}

	url := fmt.Sprintf("http://localhost:%s/status", config.Conf.Worker_port)
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

	if _, err := os.Stat(olPath); os.IsNotExist(err) {
		fmt.Printf("no OL directory found at %s\n", olPath)
		if err := initOLDir(olPath); err != nil {
			return err
		}
	} else {
		fmt.Printf("using existing OL directory at %s\n", olPath)
	}

	confPath := filepath.Join(olPath, "config.json")

	// should we run as a background process?
	detach := ctx.Bool("detach")

	if detach {
		err := config.LoadFile(confPath)
		if err != nil {
			return err
		}

		// stdout+stderr both go to log
		logPath := filepath.Join(olPath, "worker.out")
		f, err := os.Create(logPath)
		if err != nil {
			return err
		}
		attr := os.ProcAttr{
			Files: []*os.File{nil, f, f},
		}
		cmd := []string{
			os.Args[0],
			"worker",
			"-path=" + olPath,
		}
		proc, err := os.StartProcess(os.Args[0], cmd, &attr)
		if err != nil {
			return err
		}

		pidPath := filepath.Join(olPath, "worker.pid")
		if err := ioutil.WriteFile(pidPath, []byte(fmt.Sprintf("%d", proc.Pid)), 0644); err != nil {
			return err
		}

		fmt.Printf("Starting worker: pid=%d, port=%s, log=%s\n", proc.Pid, config.Conf.Worker_port, logPath)

		var ping_err error

		for i := 0; i < 300; i++ {
			// is the worker still alive?
			err := proc.Signal(syscall.Signal(0))
			if err != nil {
				return fmt.Errorf("worker process %d does not a appear to be running :: %s", proc.Pid, err)
			}

			// is it reachable?
			url := fmt.Sprintf("http://localhost:%s/pid", config.Conf.Worker_port)
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
		server.Main(confPath)
	}

	return nil
}

// kill corresponds to the "kill" command of the admin tool.
func kill(ctx *cli.Context) error {
	olPath, err := getOlPath(ctx)
	if err != nil {
		return err
	}

	data, err := ioutil.ReadFile(filepath.Join(olPath, "worker.pid"))
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

// setconf sets a configuration option in the cluster's template
func setconf(ctx *cli.Context) error {
	olPath, err := getOlPath(ctx)
	if err != nil {
		return err
	}

	configPath := filepath.Join(olPath, "config.json")

	if len(ctx.Args()) != 1 {
		log.Fatal("Usage: admin setconf <json_options>")
	}

	if err := config.LoadFile(configPath); err != nil {
		return err
	} else if err := json.Unmarshal([]byte(ctx.Args()[0]), config.Conf); err != nil {
		return fmt.Errorf("failed to set config options :: %v", err)
	} else if err := config.Save(configPath); err != nil {
		return err
	}

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
			Name:      "setconf",
			Usage:     "Set a configuration option in the cluster's template.",
			UsageText: "ol setconf [--path=NAME] options (options is JSON string)",
			Flags:     []cli.Flag{pathFlag},
			Action:    setconf,
		},
		cli.Command{
			Name:        "worker",
			Usage:       "Start one OL server",
			UsageText:   "ol worker [--path=NAME] [--detach]",
			Description: "Start one or more workers in cluster using the same config template.",
			Flags: []cli.Flag{
				pathFlag,
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
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
