// go/invoke.go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/urfave/cli/v2"
)

func invokeAction(ctx *cli.Context) error {
	// Accept function name either positionally or via --func (not both)
	fnFlag := ctx.String("func")
	var funcName string
	if ctx.NArg() > 0 {
		funcName = ctx.Args().First()
	}
	if funcName != "" && fnFlag != "" {
		return cli.Exit("pass function as positional OR --func, not both", 2)
	}
	if funcName == "" {
		funcName = fnFlag
	}
	if funcName == "" {
		return cli.Exit("usage: ol invoke <func> [--data JSON | --json FILE | --file PATH] [--header 'K: V' ...] [--timeout N] [--pretty] [--url BASE] [--path TEMPLATE]", 2)
	}

	// Base URL: flag > env > default
	base := ctx.String("url")
	if base == "" {
		base = os.Getenv("OL_URL")
		if base == "" {
			base = "http://127.0.0.1:5000"
		}
	}

	// Route template
	pathTpl := ctx.String("path")
	if pathTpl == "" {
		pathTpl = "/invoke/{func}"
	}
	route := strings.ReplaceAll(pathTpl, "{func}", funcName)
	fullURL := strings.TrimRight(base, "/") + route

	// Choose body + content-type
	var body []byte
	contentType := "application/json"

	if f := ctx.String("file"); f != "" {
		b, err := os.ReadFile(f)
		if err != nil {
			return err
		}
		body = b
		contentType = "application/octet-stream"
	} else if jf := ctx.String("json"); jf != "" {
		b, err := os.ReadFile(jf)
		if err != nil {
			return err
		}
		body = b
	} else if d := ctx.String("data"); d != "" {
		body = []byte(d)
	} else {
		body = []byte(`{}`)
	}

	req, err := http.NewRequest(http.MethodPost, fullURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", contentType)

	// Extra headers: --header "K: V" (repeatable)
	for _, h := range ctx.StringSlice("header") {
		parts := strings.SplitN(h, ":", 2)
		if len(parts) != 2 {
			return cli.Exit(fmt.Sprintf("invalid --header %q (use 'K: V')", h), 2)
		}
		k := strings.TrimSpace(parts[0])
		v := strings.TrimSpace(parts[1])
		if k == "" {
			return cli.Exit(fmt.Sprintf("invalid --header %q (empty key)", h), 2)
		}
		req.Header.Set(k, v)
	}

	client := &http.Client{
		Timeout: time.Duration(ctx.Int("timeout")) * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	out, _ := io.ReadAll(resp.Body)

	// Pretty or raw output
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

	// Non-2xx => fail (handy for scripts)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return cli.Exit(fmt.Sprintf("HTTP %d", resp.StatusCode), 1)
	}
	return nil
}

func invokeCommand() *cli.Command {
	return &cli.Command{
		Name:        "invoke",
		Usage:       "Invoke a function on an OL worker over HTTP",
		UsageText:   "ol invoke <func> [--data JSON | --json FILE | --file PATH] [--header 'K: V' ...] [--timeout N] [--pretty] [--url BASE] [--path TEMPLATE]",
		Description: "Sends a POST to BASE/path with the function name substituted. Defaults: BASE from --url or $OL_URL or http://127.0.0.1:5000; path=/invoke/{func}.",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "func", Usage: "Function name (alternative to positional)"},
			&cli.StringFlag{Name: "url", Usage: "Base worker URL (overrides OL_URL)"},
			&cli.IntFlag{Name: "timeout", Value: 15, Usage: "HTTP timeout seconds"},
			&cli.BoolFlag{Name: "pretty", Usage: "Pretty-print JSON responses"},
			&cli.StringFlag{Name: "data", Usage: "Inline JSON payload"},
			&cli.StringFlag{Name: "json", Usage: "Path to JSON file payload"},
			&cli.StringFlag{Name: "file", Usage: "Path to binary file payload"},
			&cli.StringSliceFlag{Name: "header", Usage: `Extra header "K: V" (repeatable)`},
			&cli.StringFlag{Name: "path", Usage: `Route template (default "/invoke/{func}")`},
		},
		Action: invokeAction,
	}
}
