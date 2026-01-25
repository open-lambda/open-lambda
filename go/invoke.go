// Minimal ol invoke (POST JSON only)
package main

import (
	"bytes"
	"path/filepath"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/urfave/cli/v2"
	"github.com/open-lambda/open-lambda/go/common"
)

func invokeAction(ctx *cli.Context) error {
	// Usage: ol invoke <func> [json] [-p PROJECT]
	if ctx.NArg() < 1 || ctx.NArg() > 2 {
		return cli.Exit("usage: ol invoke <func> [json] [-p PROJECT]", 2)
	}

	funcName := ctx.Args().Get(0)
	jsonArg := ""
	if ctx.NArg() == 2 {
		jsonArg = ctx.Args().Get(1)
	}

	// Resolve worker URL like `ol worker up` (stub for now; wired below)
	project := ctx.String("project")
	baseURL, err := resolveWorkerURL(ctx, project)
	if err != nil {
		return cli.Exit(fmt.Sprintf("failed to resolve worker URL: %v", err), 1)
	}

	url := strings.TrimRight(baseURL, "/") + "/invoke/" + funcName

	// Body: no arg → "null"; with arg → must be valid JSON
	var body []byte
	if jsonArg == "" {
		body = []byte("null")
	} else {
		var tmp any
		if err := json.Unmarshal([]byte(jsonArg), &tmp); err != nil {
			return cli.Exit(fmt.Sprintf("invalid JSON: %v", err), 2)
		}
		body = []byte(jsonArg)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: time.Duration(ctx.Int("timeout")) * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	out, _ := io.ReadAll(resp.Body)

	if ctx.Bool("pretty") {
		var js any
		if json.Unmarshal(out, &js) == nil {
			b, _ := json.MarshalIndent(js, "", "  ")
			fmt.Println(string(b))
		} else {
			fmt.Println(string(out))
		}
	} else {
		fmt.Println(string(out))
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return nil
}

// TODO: wire to the same project->URL resolver used by `ol worker up`
func resolveWorkerURL(ctx *cli.Context, project string) (string, error) {
	// If -p/--project is set (or defaults), reuse the same deploy path logic as `ol worker up`:
	// 1) Resolve deploy dir via common.GetOlPath(ctx)
	// 2) Load its config.json -> common.Conf.Worker_port
	// 3) Return http://localhost:<port>
	olPath, err := common.GetOlPath(ctx)
	if err == nil {
		confPath := filepath.Join(olPath, "config.json")
		if err2 := common.LoadGlobalConfig(confPath); err2 == nil && common.Conf.Worker_port != "" {
			return "http://localhost:" + common.Conf.Worker_port, nil
		}
	}
	// Fallbacks: OL_URL env, else 127.0.0.1:5000 (useful for local/mock testing)
	if v := os.Getenv("OL_URL"); v != "" {
		return v, nil
	}
	return "http://127.0.0.1:5000", nil
}

func invokeCommand() *cli.Command {
	return &cli.Command{
		Name:        "invoke",
		Usage:       "Invoke a function on an OL worker (minimal: POST JSON only)",
		UsageText:   "ol invoke <func> [json] [-p PROJECT]",
		Description: "POSTs JSON (or null) to /invoke/<func>. Will resolve worker URL same as 'ol worker up'.",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "project", Aliases: []string{"p"}, Usage: "Project/deploy name (like 'ol worker up')"},
			&cli.IntFlag{Name: "timeout", Value: 15, Usage: "HTTP timeout seconds"},
			&cli.BoolFlag{Name: "pretty", Usage: "Pretty-print JSON responses"},
		},
		Action: invokeAction,
	}
}
