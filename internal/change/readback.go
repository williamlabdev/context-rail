package change

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ReadBackRequest names what to observe from a source provider.
type ReadBackRequest struct {
	Source     string `json:"source"`     // github
	Repository string `json:"repository"` // owner/repo
	Base       string `json:"base"`       // base branch
	Head       string `json:"head"`       // candidate branch
}

// ReadBack observes a candidate from a provider. Observations replace the
// operator's declaration for branch, commits, changed paths, checks and
// reviews; declared compensating controls are kept because no provider
// exposes them.
type ReadBack interface {
	Name() string
	Observe(ctx context.Context, request ReadBackRequest) (*Observation, error)
}

// GitHubReadBack reads the diff, pull request, reviews and check runs of a
// branch through the GitHub REST API. It needs a token with read access to
// the repository (GITHUB_TOKEN).
//
// NOT exercised in the verification workspace: the proxy there binds
// sessions to configured repositories and the demo repository does not exist
// yet. Treat as UNVERIFIED until a real read-back is recorded as evidence.
type GitHubReadBack struct {
	Token   string
	BaseURL string
	Client  *http.Client
}

func NewGitHubReadBack(token string) *GitHubReadBack {
	return &GitHubReadBack{Token: token, BaseURL: "https://api.github.com", Client: &http.Client{Timeout: 30 * time.Second}}
}

func (reader *GitHubReadBack) Name() string { return "github" }

func (reader *GitHubReadBack) get(ctx context.Context, path string, out any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, reader.BaseURL+path, nil)
	if err != nil {
		return err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	if reader.Token != "" {
		request.Header.Set("Authorization", "Bearer "+reader.Token)
	}
	response, err := reader.Client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(response.Body, 4<<20))
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("github %s returned HTTP %d", path, response.StatusCode)
	}
	return json.Unmarshal(raw, out)
}

func (reader *GitHubReadBack) Observe(ctx context.Context, request ReadBackRequest) (*Observation, error) {
	if reader.Token == "" {
		return nil, errors.New("GITHUB_TOKEN not configured")
	}
	repo := strings.TrimPrefix(strings.TrimSpace(request.Repository), "github.com/")
	if !strings.Contains(repo, "/") || request.Head == "" || request.Base == "" {
		return nil, errors.New("read-back needs repository owner/repo, base and head")
	}
	observation := &Observation{Source: "github", Repository: "github.com/" + repo, Branch: request.Head, BaseBranch: request.Base, Checks: []CheckResult{}, Reviews: []ReviewEvidence{}}

	var compare struct {
		BaseCommit struct {
			SHA string `json:"sha"`
		} `json:"base_commit"`
		Commits []struct {
			SHA string `json:"sha"`
		} `json:"commits"`
		Files []struct {
			Filename string `json:"filename"`
		} `json:"files"`
	}
	if err := reader.get(ctx, fmt.Sprintf("/repos/%s/compare/%s...%s?per_page=250", repo, url.PathEscape(request.Base), url.PathEscape(request.Head)), &compare); err != nil {
		return nil, err
	}
	observation.BaseCommit = compare.BaseCommit.SHA
	if len(compare.Commits) > 0 {
		observation.HeadCommit = compare.Commits[len(compare.Commits)-1].SHA
	}
	for _, file := range compare.Files {
		observation.ChangedPaths = append(observation.ChangedPaths, file.Filename)
	}

	owner := strings.SplitN(repo, "/", 2)[0]
	var pulls []struct {
		Number  int    `json:"number"`
		HTMLURL string `json:"html_url"`
		State   string `json:"state"`
		User    struct {
			Login string `json:"login"`
		} `json:"user"`
	}
	if err := reader.get(ctx, fmt.Sprintf("/repos/%s/pulls?state=all&head=%s&per_page=5", repo, url.QueryEscape(owner+":"+request.Head)), &pulls); err == nil && len(pulls) > 0 {
		observation.PullRequest = pulls[0].HTMLURL
		var reviews []struct {
			State       string `json:"state"`
			SubmittedAt string `json:"submitted_at"`
			HTMLURL     string `json:"html_url"`
			User        struct {
				Login string `json:"login"`
			} `json:"user"`
		}
		if err := reader.get(ctx, fmt.Sprintf("/repos/%s/pulls/%d/reviews?per_page=100", repo, pulls[0].Number), &reviews); err == nil {
			for _, review := range reviews {
				verdict := ""
				switch review.State {
				case "APPROVED":
					verdict = "APPROVED"
				case "CHANGES_REQUESTED":
					verdict = "CHANGES_REQUESTED"
				default:
					continue
				}
				observation.Reviews = append(observation.Reviews, ReviewEvidence{Reviewer: review.User.Login, Kind: "human", Verdict: verdict, EvidenceRef: review.HTMLURL, At: review.SubmittedAt})
			}
		}
	}

	if observation.HeadCommit != "" {
		var checks struct {
			CheckRuns []struct {
				Name        string `json:"name"`
				Conclusion  string `json:"conclusion"`
				HTMLURL     string `json:"html_url"`
				CompletedAt string `json:"completed_at"`
			} `json:"check_runs"`
		}
		if err := reader.get(ctx, fmt.Sprintf("/repos/%s/commits/%s/check-runs?per_page=100", repo, observation.HeadCommit), &checks); err == nil {
			for _, run := range checks.CheckRuns {
				status := "FAIL"
				if run.Conclusion == "success" {
					status = "PASS"
				}
				observation.Checks = append(observation.Checks, CheckResult{Name: run.Name, Status: status, EvidenceRef: run.HTMLURL, ObservedAt: run.CompletedAt})
			}
		}
	}
	observation.ObservedAt = time.Now().UTC().Format(time.RFC3339)
	return observation, nil
}

// MergeObservation overlays a provider observation on the declared one:
// provider facts win for identity, diff, checks and reviews; declared
// compensating controls and any declared checks the provider did not see are
// kept so local evidence (e.g. a test log) is not lost.
func MergeObservation(declared Observation, observed *Observation) Observation {
	if observed == nil {
		return declared
	}
	merged := *observed
	if merged.Repository == "" {
		merged.Repository = declared.Repository
	}
	merged.CompensatingControls = declared.CompensatingControls
	seen := map[string]bool{}
	for _, check := range merged.Checks {
		seen[strings.ToLower(check.Name)] = true
	}
	for _, check := range declared.Checks {
		if !seen[strings.ToLower(check.Name)] {
			check.EvidenceRef = firstNonEmpty(check.EvidenceRef, "declared")
			merged.Checks = append(merged.Checks, check)
		}
	}
	for _, review := range declared.Reviews {
		if strings.EqualFold(review.Kind, "ai") {
			merged.Reviews = append(merged.Reviews, review)
		}
	}
	return merged
}
