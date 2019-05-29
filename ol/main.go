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
	"syscall"

	docker "github.com/fsouza/go-dockerclient"
	dutil "github.com/open-lambda/open-lambda/ol/sandbox/dockerutil"

	"github.com/open-lambda/open-lambda/ol/config"
	"github.com/open-lambda/open-lambda/ol/server"
	"github.com/urfave/cli"
)

var client *docker.Client

// TODO: notes about setup process
// TODO: notes about creating a directory in local

// Parse parses the cluster name. If required is true but
// the cluster name is empty, program will exit with an error.
func getOlPath(ctx *cli.Context) (string, error) {
	olPath := ctx.String("path")
	if olPath == "" {
		olPath = "default"
	}
	return filepath.Abs(olPath)
}

// newOL corresponds to the "new" command of the admin tool.
func newOL(ctx *cli.Context) error {
	olPath, err := getOlPath(ctx)
	if err != nil {
		return err
	}

	fmt.Printf("Init OL dir at %v\n", olPath)

	if err := os.Mkdir(olPath, 0700); err != nil {
		return err
	}

	registryDir := filepath.Join(olPath, "registry")
	if err := os.Mkdir(registryDir, 0700); err != nil {
		return err
	}

	packagesDir := filepath.Join(olPath, "packages")
	if err := os.Mkdir(packagesDir, 0700); err != nil {
		return err
	}

	// create a base directory to run sock handlers
	baseImgDir := filepath.Join(olPath, "lambda")
	fmt.Printf("Create lambda base at %v (may take several minutes)\n", baseImgDir)
	err = dutil.DumpDockerImage(client, "lambda", baseImgDir)
	if err != nil {
		return err
	}

	// need this because Docker containers don't have a dns server in /etc/resolv.conf
	dnsPath := filepath.Join(baseImgDir, "etc/resolv.conf")
	if err := ioutil.WriteFile(dnsPath, []byte("nameserver 8.8.8.8\n"), 0644); err != nil {
		return err
	}

	// config dir and template
	c := &config.Config{
		Worker_dir:     olPath,
		Cluster_name:   olPath, // TODO: why?
		Worker_port:    "5000",
		Registry:       registryDir,
		Sandbox:        "sock",
		Pkgs_dir:       packagesDir,
		Sandbox_config: map[string]interface{}{"processes": 10},
		SOCK_base_path: baseImgDir,
	}

	if err := c.Defaults(); err != nil {
		return err
	}

	if err := c.Save(filepath.Join(olPath, "config.json")); err != nil {
		return err
	}

	fmt.Printf("Working Directory: %s\n\n", olPath)
	fmt.Printf("Worker Defaults: \n%s\n\n", c.DumpStr())
	fmt.Printf("You may now start a server using the \"worker\" command\n")

	return nil
}

// status corresponds to the "status" command of the admin tool.
func status(ctx *cli.Context) error {
	olPath, err := getOlPath(ctx)
	if err != nil {
		return err
	}

	fmt.Printf("Worker Ping:\n")
	c, err := config.ParseConfig(filepath.Join(olPath, "config.json"))
	if err != nil {
		return err
	}

	url := fmt.Sprintf("http://localhost:%s/status", c.Worker_port)
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

	// should we run as a background process?
	detach := ctx.Bool("detach")

	if !detach {
		server.Main(olPath)
	} else {
		conf, err := config.ParseConfig(filepath.Join(olPath, "config.json"))
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
			"-detach",
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

		fmt.Printf("Started worker: pid %d, port %s, log at %s\n", proc.Pid, conf.Worker_port, logPath)
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

	return nil
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

	if c, err := config.ParseConfig(configPath); err != nil {
		return err
	} else if err := json.Unmarshal([]byte(ctx.Args()[0]), c); err != nil {
		return fmt.Errorf("failed to set config options :: %v", err)
	} else if err := c.Save(configPath); err != nil {
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
			Flags: []cli.Flag{pathFlag},
			Action: newOL,
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
			UsageText:   "ol worker [--path=NAME]",
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
