package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

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
	flags := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	flags.Var(&roots, "root", "Project root; repeat for multiple Projects")
	if err := flags.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}
	if len(roots) == 0 {
		fmt.Fprintln(os.Stderr, "PROJECT REGISTRY IMPORT BLOCKED: at least one Project root is required")
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
