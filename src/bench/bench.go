package bench

import (
	"fmt"
	"net/http"
	"io/ioutil"
	"path/filepath"
	"bytes"
	"time"
	"math/rand"
	
	"github.com/urfave/cli"

	"github.com/open-lambda/open-lambda/ol/common"
)

type Call struct {
        name string
}

func task(task int, reqQ chan Call, errQ chan error) {
        for {
                call, ok := <- reqQ
                if !ok {
                        errQ <- nil
                        break
                }

                url := fmt.Sprintf("http://localhost:%s/run/%s", common.Conf.Worker_port, call.name)
                resp, err := http.Post(url, "text/json", bytes.NewBuffer([]byte("null")))
                if err != nil {
                        errQ <- fmt.Errorf("failed req to %s: %v", url, err)
                        continue
                }

                body, err := ioutil.ReadAll(resp.Body)
                resp.Body.Close()

                if err != nil {
                        errQ <- fmt.Errorf("failed to %s, could not read body: %v", url, err)
                        continue
                }

                if resp.StatusCode != http.StatusOK {
                        errQ <- fmt.Errorf("failed req to %s: status %d, text '%s'", url, resp.StatusCode, string(body))
                        continue
                }

		errQ <- nil
        }
}

func run_benchmark(ctx *cli.Context, name string, tasks int, functions int, func_template string) error {
	// config
	olPath, err := common.GetOlPath(ctx)
	if err != nil {
		return err
	}
	configPath := filepath.Join(olPath, "config.json")
	if err := common.LoadConf(configPath); err != nil {
		return err
	}

	seconds := ctx.Float64("seconds")
	if seconds == 0 {
		seconds = 60.0
	}

	// launch request threads
        reqQ := make(chan Call, tasks)
        errQ := make(chan error, tasks)
        for i := 0; i < tasks; i++ {
                go task(i, reqQ, errQ)
        }

	// warmup: call lambda each once
	// TODO: make warming optional
	fmt.Printf("warming up (calling each lambda once sequentially)\n")
        for i := 0; i<functions; i++ {
		name := fmt.Sprintf(func_template, i)
		fmt.Printf("warmup %s (%d/%d)\n", name, i, functions)
		reqQ <- Call{name: name}
		if err := <- errQ; err != nil {
			return err
		}
        }

	// issue requests for specified number of seconds
	fmt.Printf("start benchmark (%.1f seconds)\n", seconds)
	errors := 0
	successes := 0
	waiting := 0

	start := time.Now()
        for time.Since(start).Seconds() < seconds {
                select {
                case reqQ <- Call{name: fmt.Sprintf(func_template, rand.Intn(functions))}:
			waiting += 1
                case err := <- errQ:
			if err != nil {
				errors += 1
				fmt.Printf("%s\n", err.Error())
			} else {
				successes += 1
			}
			waiting -= 1
                }
        }
	seconds = time.Since(start).Seconds()

	// cleanup request threads
	fmt.Printf("cleanup\n")
        close(reqQ)
	waiting += tasks // each needs to send one last nil to indicate it is done
        for waiting > 0 {
		if err := <- errQ; err != nil {
			errors += 1
			fmt.Printf("%s\n", err.Error())
		}
		waiting -= 1
	}

	if errors > (errors + successes) / 100 {
		panic(fmt.Sprintf(">1% of requests failed (%d/%d)", errors, errors + successes))
	}

        fmt.Printf("{\"benchmark\": \"%s\",\"seconds\": %.3f, \"successes\": %d, \"errors\": %d, \"ops/s\": %.3f}\n",
		name, seconds, successes, errors, float64(successes)/seconds)

	return nil
}

func create_lambdas(ctx *cli.Context) error {	
	olPath, err := common.GetOlPath(ctx)
	if err != nil {
		return err
	}

	configPath := filepath.Join(olPath, "config.json")

	if err := common.LoadConf(configPath); err != nil {
		return err
	}

	for i := 0; i < 64*1024; i++ {
		// noop
		path := filepath.Join(common.Conf.Registry, fmt.Sprintf("bench-py-%d.py", i))
		code := fmt.Sprintf("def f(event):\n\treturn %d", i)

		fmt.Printf("%s\n", path)
		if err := ioutil.WriteFile(path, []byte(code), 0644); err != nil {
			return err
		}

		// simple pandas operation (correlation between two columns in 1000x10 DataFrame)
		path = filepath.Join(common.Conf.Registry, fmt.Sprintf("bench-pd-%d.py", i))
		code = `# ol-install: numpy,pandas
import numpy as np
import pandas as pd

df = None

def f(event):
    global df
    if df is None:
        df = pd.DataFrame(np.random.random((1000,10)))
    col0 = np.random.randint(len(df.columns))
    col1 = np.random.randint(len(df.columns))
    return df[col0].corr(df[col1])
`

		fmt.Printf("%s\n", path)
		if err := ioutil.WriteFile(path, []byte(code), 0644); err != nil {
			return err
		}
	}

	return nil
}

func make_action(name string, tasks int, functions int, func_template string) (func (ctx *cli.Context) error) {
	return func (ctx *cli.Context) error {
		return run_benchmark(ctx, name, tasks, functions, func_template)
	}
}

func BenchCommands() []cli.Command {
	cmds := []cli.Command{
		{
                        Name:  "init",
                        Usage: "creates lambdas for benchmarking",
			UsageText: "ol bench init [--path=NAME]",
                        Action: create_lambdas,
			// TODO: add param to decide how many to create
		},
	}

	for _, kind := range []string{"py", "pd"} {
		for _, functions := range []int{64, 1024, 64*1024} {
			for _, tasks := range []int{1, 32} {
				var parseq string
				var par_usage string
				var usage string
				amt := fmt.Sprintf("%d", functions)

				if tasks == 1 {
					parseq = "seq"
					par_usage = "sequentially"
				} else {
					parseq = "par"
					par_usage = fmt.Sprintf("in parallel (%d clients)", tasks)
				}
				if functions >= 1024 {
					amt = fmt.Sprintf("%dk", functions / 1024)
				}
				if kind == "py" {
					usage = fmt.Sprintf(("invoke noop Python lambdas %s for S seconds (default 60), " +
						"randomly+uniformaly selecting 1 of %d lambdas for each request"), par_usage, functions)
				} else if kind == "pd" {
					usage = fmt.Sprintf(("invoke Pandas lambdas that do correlations %s for S seconds (default 60), " +
						"randomly+uniformaly selecting 1 of %d lambdas for each request"), par_usage, functions)
				}

				name := fmt.Sprintf("%s%s-%s", kind, amt, parseq)
				action := make_action(name, tasks, functions, "bench-"+kind+"-%d")
				cmd := cli.Command{
					Name:  name,
					Usage: usage,
					UsageText: fmt.Sprintf("ol bench %s [--path=NAME] [--seconds=SECONDS]", name),
					Action: action,
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "path, p",
							Usage: "Path location for OL environment",
						},
						cli.Float64Flag{
							Name:  "seconds, s",
							Usage: "Seconds to run (after warmup)",
						},
					},
				}
				cmds = append(cmds, cmd)
			}
		}
	}

	// TODO: add command that runs them all

	return cmds
}
