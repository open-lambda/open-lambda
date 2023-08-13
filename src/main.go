package main

import (
	"fmt"
	"github.com/open-lambda/open-lambda/ol/websocket"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/open-lambda/open-lambda/ol/bench"
	"github.com/open-lambda/open-lambda/ol/boss"
	"github.com/open-lambda/open-lambda/ol/common"
	"github.com/open-lambda/open-lambda/ol/worker"

	"github.com/urfave/cli/v2"
)

func newBossConf() error {
	if err := boss.LoadDefaults(); err != nil {
		return err
	}

	if err := boss.SaveConf("boss.json"); err != nil {
		return err
	}

	fmt.Printf("populated boss.json with default settings\n")
	return nil
}

// newBoss corresponses to the "new-boss" command of the admin tool.
func newBoss(ctx *cli.Context) error {
	return newBossConf()
}

// runBoss corresponses to the "boss" command of the admin tool.
func runBoss(ctx *cli.Context) error {
	if _, err := os.Stat("boss.json"); os.IsNotExist(err) {
		newBossConf()
	}

	if err := boss.LoadConf("boss.json"); err != nil {
		return err
	}

	return bossStart(ctx)
}

func startWebSocketAPI(ctx *cli.Context) error {
	// start the websocket API server
	if err := websocket.Start(ctx); err != nil {
		return err
	}
	return nil
}

// corresponds to the "pprof mem" command of the admin tool.
func pprofMem(ctx *cli.Context) error {
	olPath, err := common.GetOlPath(ctx)
	if err != nil {
		return err
	}

	err = common.LoadConf(filepath.Join(olPath, "config.json"))
	if err != nil {
		return err
	}

	url := fmt.Sprintf("http://localhost:%s/pprof/mem", common.Conf.Worker_port)
	response, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("could not send GET to %s", url)
	}
	defer response.Body.Close()

	path := ctx.String("out")
	if path == "" {
		path = "mem.prof"
	}
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, response.Body); err != nil {
		return err
	}
	fmt.Printf("output saved to %s.  Use the following to explore:\n", path)
	fmt.Printf("go tool pprof -http=localhost:8888 %s\n", path)

	return nil
}

func bossStart(ctx *cli.Context) error {
	detach := ctx.Bool("detach")

	// If detach is specified, we start another ol-process with the worker argument
	if detach {
		// stdout+stderr both go to log
		logPath := "boss.out"
		// creates a worker.out file
		f, err := os.Create(logPath)
		if err != nil {
			return err
		}
		// holds attributes that will be used when os.StartProcess
		attr := os.ProcAttr{
			Files: []*os.File{nil, f, f},
		}
		cmd := []string{}
		for _, arg := range os.Args {
			if arg != "-d" && arg != "--detach" {
				cmd = append(cmd, arg)
			}
		}

		// Get the path of this binary
		binPath, err := exec.LookPath(os.Args[0])
		if err != nil {
			return err
		}
		// start the worker process
		fmt.Printf("starting process: binpath= %s, cmd=%s\n", binPath, cmd)
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

		fmt.Printf("Starting boss: pid=%d, port=%s, log=%s\n", proc.Pid, boss.Conf.Boss_port, logPath)
		return nil // TODO: ping status to make sure it is actually running?
	}

	if err := boss.BossMain(); err != nil {
		return err
	}

	return fmt.Errorf("this code should not be reachable")
}

func gcpTest(ctx *cli.Context) error {
	boss.GCPBossTest()
	return nil
}

func azureTest(ctx *cli.Context) error {
	boss.AzureMain("default contents")
	return nil
}

// main runs the admin tool
func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

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
	app.Commands = []*cli.Command{
		&cli.Command{
			Name:        "new-boss",
			Usage:       "Create an OL Boss config (boss.json)",
			UsageText:   "ol new-boss [--path=PATH] [--detach]",
			Description: "Create config for new boss",
			Action:      newBoss,
		},
		&cli.Command{
			Name:        "boss",
			Usage:       "Start an OL Boss process",
			UsageText:   "ol boss [--path=PATH] [--detach]",
			Description: "Start a boss server.",
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:    "detach",
					Aliases: []string{"d"},
					Usage:   "Run worker in background",
				},
			},
			Action: runBoss,
		},
		&cli.Command{
			Name:        "websocket-api",
			Usage:       "Start the WebSocket API server",
			UsageText:   "ol websocket-api [--port=PORT] [--host=HOST]",
			Description: "Start a WebSocket API server to provide real-time communication",
			Action:      startWebSocketAPI,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "port, p",
					Value: "4999", // default port
					Usage: "Port on which the WebSocket API server will listen",
				},
				&cli.StringFlag{
					Name:  "host, H",
					Value: "localhost", // default host
					Usage: "Host on which the WebSocket API server will listen",
				},
			},
		},
		&cli.Command{
			Name:      "gcp-test",
			Usage:     "Developer use only.  Start a GCP VM running the OL worker",
			UsageText: "ol gcp-test",
			Flags:     []cli.Flag{},
			Action:    gcpTest,
		},
		&cli.Command{
			Name:      "azure-test",
			Usage:     "Developer use only.  Start an Azure Blob ",
			UsageText: "ol zure-test",
			Flags:     []cli.Flag{},
			Action:    azureTest,
		},
		&cli.Command{
			Name:        "worker",
			Usage:       "Run OL worker commands.",
			UsageText:   "ol worker <cmd>",
			Subcommands: worker.WorkerCommands(),
		},
		&cli.Command{
			Name:        "bench",
			Usage:       "Run benchmarks against an OL worker.",
			UsageText:   "ol bench <cmd>",
			Subcommands: bench.BenchCommands(),
		},
		&cli.Command{
			Name:      "pprof",
			Usage:     "Profile OL worker",
			UsageText: "ol pprof <cmd>",
			Subcommands: []*cli.Command{
				{
					Name:      "mem",
					Usage:     "creates lambdas for benchmarking",
					UsageText: "ol pprof mem [--out=NAME]",
					Action:    pprofMem,
					Flags: []cli.Flag{
						&cli.StringFlag{
							Name:    "out",
							Aliases: []string{"o"},
						},
					},
				},
			},
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
