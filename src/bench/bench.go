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

func run_benchmark(ctx *cli.Context, name string, seconds float64, tasks int, functions int, func_template string) error {
	olPath, err := common.GetOlPath(ctx)
	if err != nil {
		return err
	}
	configPath := filepath.Join(olPath, "config.json")
	if err := common.LoadConf(configPath); err != nil {
		return err
	}

	// launch request threads
        reqQ := make(chan Call, tasks)
        errQ := make(chan error, tasks)
        for i := 0; i < tasks; i++ {
                go task(i, reqQ, errQ)
        }

	// warmup: call lambda each once
	fmt.Printf("warming up (calling each lambda once sequentially)\n")
        for i := 0; i<functions; i++ {
		name := fmt.Sprintf(func_template, i)
		fmt.Printf("warmup %s (%d/%d)\n", name, i, functions)
                select {
                case reqQ <- Call{name: name}:
                case err := <- errQ:
			if err != nil {
				return err
			}
                }
        }

	// issue requests for specified number of seconds
	fmt.Printf("start benchmark\n")
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
			fmt.Printf(err.Error())
		}
		waiting -= 1
	}

	if errors > (errors + successes) / 100 {
		panic(fmt.Sprintf(">1% of requests failed (%d/%d)", errors, errors + successes))
	}

        fmt.Printf("{\"benchmark\": %s,\"seconds\": %.3f, \"successes\": %d, \"errors\": %d, \"ops/s\": %.3f}",
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
		path := filepath.Join(common.Conf.Registry, fmt.Sprintf("bench-py-%d.py", i))
		code := fmt.Sprintf("def f(event):\n\treturn %d", i)

		fmt.Printf("%s\n", path)
		if err := ioutil.WriteFile(path, []byte(code), 0644); err != nil {
			return err
		}
	}

	return nil
}

func make_action(name string, seconds float64, tasks int, functions int, func_template string) (func (ctx *cli.Context) error) {
	return func (ctx *cli.Context) error {
		return run_benchmark(ctx, name, seconds, tasks, functions, func_template)
	}
}

func BenchCommands() []cli.Command {
	cmds := []cli.Command{
		{
                        Name:  "init",
                        Usage: "creates lambdas for benchmarking",
			UsageText: "ol bench init [--path=NAME]",
                        Action: create_lambdas,
		},
	}

	seconds := 60.0 // TODO: add param

	// TODO: add one that uses pandas (pd) instead of just the .py option

	for _, functions := range []int{64, 1024, 64*1024} {
		for _, tasks := range []int{1, 32} {
			var parseq string
			if tasks == 1 {
				parseq = "seq"
			} else {
				parseq = "par"
			}
			amt := fmt.Sprintf("%d", functions)
			if functions >= 1024 {
				amt = fmt.Sprintf("%dk", functions / 1024)
			}
			name := fmt.Sprintf("py%s-%s", amt, parseq)
			action := make_action(name, seconds, tasks, functions, "bench-py-%d")
			usage := fmt.Sprintf(("invoke noop Python lambdas sequentially for %d seconds, " +
				"randomly+uniformaly selecting 1 of %d lambdas for each request"), int(seconds), functions)
			cmd := cli.Command{
				Name:  name,
				Usage: usage,
				UsageText: fmt.Sprintf("ol bench init [--path=NAME]"),
				Action: action,
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "path, p",
						Usage: "Path location for OL environment",
					},
				},
			}
			cmds = append(cmds, cmd)
		}
	}

	return cmds
}
