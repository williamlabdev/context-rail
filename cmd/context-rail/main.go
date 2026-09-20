package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"context-rail/internal/change"
	registryhttp "context-rail/internal/http"
	"context-rail/internal/projectregistry"
	"context-rail/internal/release"
	"context-rail/internal/topology"
)

// Environment variables honoured in HTTP mode. They let a container image run
// without flags while keeping every Project root explicit; there is still no
// arbitrary filesystem traversal, remote discovery or write path.
const (
	envPort         = "PORT"                       // Cloud Run container contract
	envFixtureRoots = "CONTEXT_RAIL_FIXTURE_ROOTS" // comma-separated Project roots
	envStaticDir    = "CONTEXT_RAIL_STATIC_DIR"    // built workspace directory
	envScenario     = "CONTEXT_RAIL_FIXTURE_SCENARIO"
	envStateDir     = "CONTEXT_RAIL_STATE_DIR" // governance state (topology versions, change ledger); JSON files in P0
	envGeminiKey    = "GEMINI_API_KEY"         // optional: enables the Gemini decision advisor (candidates only)
	envGeminiModel  = "CONTEXT_RAIL_GEMINI_MODEL"
	envGitHubToken  = "GITHUB_TOKEN" // optional: enables GitHub read-back of candidates
)

const shutdownGrace = 10 * time.Second

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
	var staticDir string
	var stateDir string
	var fixtureScenario string
	var fixtureDelayMS int
	flags := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	flags.Var(&roots, "root", "Project root for snapshot mode; repeat for multiple Projects")
	flags.Var(&fixtureRoots, "fixture-root", "Project root for HTTP mode; repeat for multiple Projects (or set "+envFixtureRoots+")")
	flags.StringVar(&addr, "addr", "", "HTTP listen address; defaults to 127.0.0.1:8080, or 0.0.0.0:$PORT when "+envPort+" is set")
	flags.StringVar(&staticDir, "static-dir", "", "Built workspace directory; defaults to frontend/dist or "+envStaticDir)
	flags.StringVar(&stateDir, "state-dir", "", "Governance state directory for topology versions; defaults to .context-rail-state or "+envStateDir)
	flags.StringVar(&fixtureScenario, "fixture-scenario", "", "Verification scenario: normal, empty, invalid or unavailable (or "+envScenario+")")
	flags.IntVar(&fixtureDelayMS, "fixture-delay-ms", 0, "Delay fixture responses by this many milliseconds")
	if err := flags.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}

	if len(fixtureRoots) == 0 {
		fixtureRoots = splitRoots(os.Getenv(envFixtureRoots))
	}
	if len(roots) > 0 && len(fixtureRoots) > 0 {
		fmt.Fprintln(os.Stderr, "CONTEXT RAIL BLOCKED: use either --root or --fixture-root, not both")
		os.Exit(2)
	}
	if len(fixtureRoots) > 0 {
		if fixtureScenario == "" {
			fixtureScenario = firstNonEmpty(os.Getenv(envScenario), "normal")
		}
		if !validFixtureScenario(fixtureScenario) || fixtureDelayMS < 0 {
			fmt.Fprintln(os.Stderr, "CONTEXT RAIL BLOCKED: invalid fixture scenario or negative fixture delay")
			os.Exit(2)
		}
		if staticDir == "" {
			staticDir = firstNonEmpty(os.Getenv(envStaticDir), filepath.Join("frontend", "dist"))
		}
		if stateDir == "" {
			stateDir = firstNonEmpty(os.Getenv(envStateDir), ".context-rail-state")
		}
		if addr == "" {
			addr = listenAddress(os.Getenv(envPort))
		}
		serveHTTP(fixtureRoots, addr, staticDir, stateDir, registryhttp.ServerOptions{Scenario: fixtureScenario, Delay: time.Duration(fixtureDelayMS) * time.Millisecond})
		return
	}
	if len(roots) == 0 {
		fmt.Fprintln(os.Stderr, "CONTEXT RAIL BLOCKED: provide --root for snapshot mode, or --fixture-root / "+envFixtureRoots+" for HTTP mode")
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

// serveHTTP runs the read-only registry API and the built workspace until the
// process receives SIGTERM or SIGINT, then drains in-flight requests. Cloud Run
// sends SIGTERM before stopping an instance; local runs stop with Ctrl-C.
func serveHTTP(roots rootsFlag, addr, staticDir, stateDir string, options registryhttp.ServerOptions) {
	mode := "local"
	if os.Getenv(envPort) != "" {
		mode = "container"
	}
	topologyStore, err := topology.NewFileStore(stateDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "CONTEXT RAIL BLOCKED: state directory unusable: %v\n", err)
		os.Exit(2)
	}
	changeStore, err := change.NewFileStore(stateDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "CONTEXT RAIL BLOCKED: state directory unusable: %v\n", err)
		os.Exit(2)
	}
	releaseStore, err := release.NewFileStore(stateDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "CONTEXT RAIL BLOCKED: state directory unusable: %v\n", err)
		os.Exit(2)
	}
	topologyService := topology.NewService(topologyStore, topology.NewRegistrySource(roots))
	var advisor change.Advisor = change.RuleAdvisor{}
	advisorName := "rule-advisor"
	if key := os.Getenv(envGeminiKey); key != "" {
		gemini := change.NewGeminiAdvisor(key, os.Getenv(envGeminiModel))
		advisor = gemini
		advisorName = gemini.Name() + " (falls back to rule-advisor on error)"
	}
	changeService := change.NewService(changeStore, change.NewRegistrySource(roots, topologyService), advisor)
	mux := http.NewServeMux()
	registryhttp.NewTopologyHandler(topologyService).Register(mux)
	changesHandler := registryhttp.NewChangesHandler(changeService)
	readBackName := "none"
	if token := os.Getenv(envGitHubToken); token != "" {
		changesHandler = changesHandler.WithReadBack(change.NewGitHubReadBack(token))
		readBackName = "github"
	}
	changesHandler.Register(mux)
	registryhttp.NewReleasesHandler(release.NewService(releaseStore, changeService, topologyService)).Register(mux)
	mux.Handle("/v1/", registryhttp.NewServerWithOptions(roots, options))
	mux.Handle("/healthz", registryhttp.NewHealthHandler(mode))
	mux.Handle("/", registryhttp.NewStaticHandler(staticDir))

	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	errs := make(chan error, 1)
	go func() {
		fmt.Fprintf(os.Stderr, "ContextRail workspace listening on http://%s (mode=%s fixtures=%v static=%s state=%s advisor=%s read_back=%s scenario=%s delay_ms=%d)\n",
			addr, mode, []string(roots), staticDir, stateDir, advisorName, readBackName, options.Scenario, options.Delay/time.Millisecond)
		errs <- server.ListenAndServe()
	}()

	select {
	case err := <-errs:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Fprintf(os.Stderr, "CONTEXT RAIL SERVER STOPPED: %v\n", err)
			os.Exit(1)
		}
	case <-ctx.Done():
		fmt.Fprintln(os.Stderr, "ContextRail workspace shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownGrace)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			fmt.Fprintf(os.Stderr, "CONTEXT RAIL SERVER SHUTDOWN INCOMPLETE: %v\n", err)
			os.Exit(1)
		}
	}
}

// listenAddress applies the Cloud Run contract: when PORT is set the process
// must accept traffic on every interface at that port. Without PORT the
// workspace stays loopback-only for local verification.
func listenAddress(port string) string {
	if port == "" {
		return "127.0.0.1:8080"
	}
	return "0.0.0.0:" + port
}

func splitRoots(value string) rootsFlag {
	var roots rootsFlag
	for _, item := range strings.Split(value, ",") {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			roots = append(roots, trimmed)
		}
	}
	return roots
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func validFixtureScenario(value string) bool {
	switch value {
	case "normal", "empty", "invalid", "unavailable":
		return true
	default:
		return false
	}
}
