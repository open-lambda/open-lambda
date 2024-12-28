// Package main is entry point for the `ol` command
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

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

// runBoss corresponses to the "boss" command of the admin tool.
func runBoss(ctx *cli.Context) error {
	if _, err := os.Stat("boss.json"); os.IsNotExist(err) {
		newBossConf()
	}

	confPath := "boss.json"
	overrides := ctx.String("options")
	if overrides != "" {
		overridesPath := confPath + ".overrides"
		err := overrideOpts(confPath, overridesPath, overrides)
		if err != nil {
			return err
		}
		confPath = overridesPath
	}

	if err := boss.LoadConf(confPath); err != nil {
		return err
	}

	return bossStart(ctx)
}

// modify the config.json file based on settings from cmdline: -o opt1=val1,opt2=val2,...
//
// apply changes in optsStr to config from confPath, saving result to overridePath
func overrideOpts(confPath, overridePath, optsStr string) error {
	b, err := ioutil.ReadFile(confPath)
	if err != nil {
		return err
	}
	conf := make(map[string]any)
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
			case map[string]any:
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

	return ioutil.WriteFile(overridePath, s, 0644)
}

// pprofMem corresponds to the "pprof mem" command of the admin tool.
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

// pprofCpuStart corresponds to the "pprof cpu-start" command of the admin tool.
func pprofCpuStart(ctx *cli.Context) error {
	olPath, err := common.GetOlPath(ctx)
	if err != nil {
		return err
	}

	err = common.LoadConf(filepath.Join(olPath, "config.json"))
	if err != nil {
		return err
	}

	url := fmt.Sprintf("http://localhost:%s/pprof/cpu-start", common.Conf.Worker_port)
	response, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("Could not send GET to %s", url)
	}
	defer response.Body.Close()

	if response.StatusCode == 200 {
		fmt.Printf("started cpu profiling\n")
		fmt.Printf("use \"ol pprof cpu-stop\" to stop\n")
		return nil
	}

	if response.StatusCode == 500 {
		return fmt.Errorf("Unknown server error\n")
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("failed to read body from GET to %s", url)
	}
	return fmt.Errorf("Failed to start cpu profiling: %s\n", body)
}

// pprofCpuStop corresponds to the "pprof cpu-stop" command of the admin tool.
func pprofCpuStop(ctx *cli.Context) error {
	olPath, err := common.GetOlPath(ctx)
	if err != nil {
		return err
	}

	err = common.LoadConf(filepath.Join(olPath, "config.json"))
	if err != nil {
		return err
	}

	url := fmt.Sprintf("http://localhost:%s/pprof/cpu-stop", common.Conf.Worker_port)
	response, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("Could not send GET to %s", url)
	}
	defer response.Body.Close()
	if response.StatusCode == 400 {
		return fmt.Errorf("Should call \"ol pprof cpu-start\" first\n")
	} else if response.StatusCode == 500 {
		return fmt.Errorf("Unknown server error\n")
	}

	path := ctx.String("out")
	if path == "" {
		path = "cpu.prof"
	}
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, response.Body); err != nil {
		return err
	}
	fmt.Printf("output saved to %s. Use the following to explore:\n", path)
	fmt.Printf("go tool pprof -http=localhost:8889 %s\n", path)

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
			Name:        "boss",
			Usage:       "Start an OL Boss process",
			UsageText:   "ol boss [OPTIONS...] [--detach]",
			Description: "Start a boss server.",
			Flags: []cli.Flag{
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
			},
			Action: runBoss,
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
				{
					Name:      "cpu-start",
					Usage:     "Starts CPU profiling",
					UsageText: "ol pprof cpu-start ",
					Action:    pprofCpuStart,
				},
				{
					Name:      "cpu-stop",
					Usage:     "Stops CPU profiling if started",
					UsageText: "ol pprof cpu-stop [--out=NAME]",
					Action:    pprofCpuStop,
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
