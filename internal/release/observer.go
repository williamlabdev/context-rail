package release

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// SourceLocal marks a deployment whose identity ContextRail reads back from
// the running service instead of taking it from the request (DR-017). The
// other sources stay what the operator declares: "declared" (fixtures,
// EB-008/EB-009) and "cloud-run" (record-promotion.sh, dormant since DR-015).
const SourceLocal = "local"

// LiveObservation is what one read-back of a running service reports. Every
// field comes from the service's own /healthz answer; nothing is declared.
type LiveObservation struct {
	ServiceURL  string `json:"service_url"`
	HTTPStatus  int    `json:"http_status"`
	Status      string `json:"status"`      // the service's own status field ("ok")
	Service     string `json:"service"`     // service name as the instance reports it
	Environment string `json:"environment"` // environment label as the instance reports it
	Revision    string `json:"revision"`
	BuildDigest string `json:"build_digest"`
	StartedAt   string `json:"started_at"`
	// TargetRef is derived from the answer: local/<service>-<environment>. It
	// is what the target_drift gate compares with the environment's declared
	// target_ref, so a prod-demo instance answering on the staging URL is
	// caught by the label it reports, not by the port it was reached on.
	TargetRef  string `json:"target_ref"`
	ObservedAt string `json:"observed_at"`
}

// Observer reads a running service back. The release service calls it for
// every deployment recorded with source "local" and never for a replay.
type Observer interface {
	Name() string
	Observe(ctx context.Context, serviceURL string) (*LiveObservation, error)
}

// LocalReadBack observes a service on this machine through GET /healthz.
// It refuses any host that is not a loopback address: "local" means this
// machine, and the release service must not become a generic HTTP client
// that an operator can point at arbitrary network targets.
type LocalReadBack struct {
	Client *http.Client
	now    func() time.Time
}

const (
	localReadBackTimeout = 5 * time.Second
	maxHealthzBody       = 64 << 10
)

func NewLocalReadBack() *LocalReadBack {
	// Redirects are never followed: a loopback service answering 3xx must not
	// be able to point the read-back at another host.
	client := &http.Client{Timeout: localReadBackTimeout, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	return &LocalReadBack{Client: client, now: func() time.Time { return time.Now().UTC() }}
}

func (reader *LocalReadBack) Name() string { return "loopback" }

// Observe reads <serviceURL>/healthz. A non-200 answer is returned as an
// observation with HTTPStatus set (the deployment is reachable but not
// healthy); a connection failure or a non-loopback URL is an error.
func (reader *LocalReadBack) Observe(ctx context.Context, serviceURL string) (*LiveObservation, error) {
	parsed, err := loopbackURL(serviceURL)
	if err != nil {
		return nil, err
	}
	target := strings.TrimRight(parsed.String(), "/") + "/healthz"
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, fmt.Errorf("local read-back: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	client := reader.Client
	if client == nil {
		client = &http.Client{Timeout: localReadBackTimeout}
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("local read-back: %s unreachable: %w", target, err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, maxHealthzBody))
	if err != nil {
		return nil, fmt.Errorf("local read-back: reading %s: %w", target, err)
	}
	now := reader.now
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	observation := &LiveObservation{ServiceURL: strings.TrimRight(parsed.String(), "/"), HTTPStatus: response.StatusCode, ObservedAt: now().Format(time.RFC3339)}
	var answer struct {
		Status      string `json:"status"`
		Service     string `json:"service"`
		Environment string `json:"environment"`
		Revision    string `json:"revision"`
		BuildDigest string `json:"build_digest"`
		StartedAt   string `json:"started_at"`
	}
	if json.Unmarshal(body, &answer) == nil {
		observation.Status = strings.TrimSpace(answer.Status)
		observation.Service = strings.TrimSpace(answer.Service)
		observation.Environment = strings.TrimSpace(answer.Environment)
		observation.Revision = strings.TrimSpace(answer.Revision)
		observation.BuildDigest = strings.TrimSpace(answer.BuildDigest)
		observation.StartedAt = strings.TrimSpace(answer.StartedAt)
	}
	if observation.Service != "" && observation.Environment != "" {
		observation.TargetRef = LocalTargetRef(observation.Service, observation.Environment)
	}
	return observation, nil
}

// LocalTargetRef is the target_ref a local instance answers for:
// local/<service>-<environment>, parallel to cloud-run/<service>-<env>.
func LocalTargetRef(service, environment string) string {
	return "local/" + service + "-" + environment
}

// loopbackURL accepts http(s)://127.0.0.0/8, [::1] or localhost only, with
// no user info, query or fragment.
func loopbackURL(raw string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return nil, fmt.Errorf("local read-back: service_url is not a URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("local read-back: service_url must be http or https, got %q", raw)
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, fmt.Errorf("local read-back: service_url must be a bare origin, got %q", raw)
	}
	host := parsed.Hostname()
	if host == "" {
		return nil, fmt.Errorf("local read-back: service_url has no host: %q", raw)
	}
	if !strings.EqualFold(host, "localhost") {
		ip := net.ParseIP(host)
		if ip == nil || !ip.IsLoopback() {
			return nil, fmt.Errorf("local read-back: %q is not a loopback address; a local deployment is read back on this machine only", host)
		}
	}
	return parsed, nil
}
