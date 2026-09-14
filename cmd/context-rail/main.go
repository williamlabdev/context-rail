package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	registryhttp "context-rail/internal/http"
	"context-rail/internal/projectregistry"
)

type rootsFlag []string

func (roots *rootsFlag) String() string {
	return fmt.Sprint([]string(*roots))
}

func (roots *rootsFlag) Set(value string) error {
	*roots = append(*roots, value)
	return nil
}

func main() {
	var roots rootsFlag
	var fixtureRoots rootsFlag
	var addr string
	var fixtureScenario string
	var fixtureDelayMS int
	flags := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	flags.Var(&roots, "root", "Project root for snapshot mode; repeat for multiple Projects")
	flags.Var(&fixtureRoots, "fixture-root", "Project root for local HTTP mode; repeat for multiple Projects")
	flags.StringVar(&addr, "addr", "127.0.0.1:8080", "HTTP listen address in local HTTP mode")
	flags.StringVar(&fixtureScenario, "fixture-scenario", "normal", "Local verification scenario: normal, empty, invalid or unavailable")
	flags.IntVar(&fixtureDelayMS, "fixture-delay-ms", 0, "Delay local fixture responses by this many milliseconds")
	if err := flags.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}
	if len(roots) > 0 && len(fixtureRoots) > 0 {
		fmt.Fprintln(os.Stderr, "CONTEXT RAIL BLOCKED: use either --root or --fixture-root, not both")
		os.Exit(2)
	}
	if len(fixtureRoots) > 0 {
		if !validFixtureScenario(fixtureScenario) || fixtureDelayMS < 0 {
			fmt.Fprintln(os.Stderr, "CONTEXT RAIL BLOCKED: invalid fixture scenario or negative fixture delay")
			os.Exit(2)
		}
		serveLocal(fixtureRoots, addr, registryhttp.ServerOptions{Scenario: fixtureScenario, Delay: time.Duration(fixtureDelayMS) * time.Millisecond})
		return
	}
	if len(roots) == 0 {
		fmt.Fprintln(os.Stderr, "CONTEXT RAIL BLOCKED: provide --root for snapshot mode or --fixture-root for HTTP mode")
		os.Exit(2)
	}
	snapshot, err := projectregistry.Import(roots)
	if err != nil {
		fmt.Fprintf(os.Stderr, "PROJECT REGISTRY IMPORT BLOCKED: %v\n", err)
		os.Exit(2)
	}
	output, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "PROJECT REGISTRY IMPORT BLOCKED: cannot serialize snapshot: %v\n", err)
		os.Exit(2)
	}
	fmt.Println(string(output))
}

func serveLocal(roots rootsFlag, addr string, options registryhttp.ServerOptions) {
	registry := registryhttp.NewServerWithOptions(roots, options)
	mux := http.NewServeMux()
	mux.Handle("/v1/", registry)
	staticRoot := filepath.Join("frontend", "dist")
	mux.Handle("/", http.FileServer(http.Dir(staticRoot)))
	fmt.Fprintf(os.Stderr, "ContextRail local workspace listening on http://%s (fixtures=%v scenario=%s delay_ms=%d)\n", addr, []string(roots), options.Scenario, options.Delay/time.Millisecond)
	if err := http.ListenAndServe(addr, mux); err != nil {
		fmt.Fprintf(os.Stderr, "CONTEXT RAIL SERVER STOPPED: %v\n", err)
		os.Exit(1)
	}
}

func validFixtureScenario(value string) bool {
	switch value {
	case "normal", "empty", "invalid", "unavailable":
		return true
	default:
		return false
	}
}
