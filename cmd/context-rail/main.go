package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

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
	flags := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	flags.Var(&roots, "root", "Project root for snapshot mode; repeat for multiple Projects")
	flags.Var(&fixtureRoots, "fixture-root", "Project root for local HTTP mode; repeat for multiple Projects")
	flags.StringVar(&addr, "addr", "127.0.0.1:8080", "HTTP listen address in local HTTP mode")
	if err := flags.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}
	if len(roots) > 0 && len(fixtureRoots) > 0 {
		fmt.Fprintln(os.Stderr, "CONTEXT RAIL BLOCKED: use either --root or --fixture-root, not both")
		os.Exit(2)
	}
	if len(fixtureRoots) > 0 {
		serveLocal(fixtureRoots, addr)
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

func serveLocal(roots rootsFlag, addr string) {
	registry := registryhttp.NewServer(roots)
	mux := http.NewServeMux()
	mux.Handle("/v1/", registry)
	staticRoot := filepath.Join("frontend", "dist")
	mux.Handle("/", http.FileServer(http.Dir(staticRoot)))
	fmt.Fprintf(os.Stderr, "ContextRail local workspace listening on http://%s (fixtures=%v)\n", addr, []string(roots))
	if err := http.ListenAndServe(addr, mux); err != nil {
		fmt.Fprintf(os.Stderr, "CONTEXT RAIL SERVER STOPPED: %v\n", err)
		os.Exit(1)
	}
}
