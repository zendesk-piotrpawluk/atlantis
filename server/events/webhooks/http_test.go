package webhooks_test

import (
	"maps"
	"net/http"
	"net/http/httptest"
	"regexp"
	"slices"
	"testing"

	"github.com/runatlantis/atlantis/server/events/models"
	"github.com/runatlantis/atlantis/server/events/webhooks"
	"github.com/runatlantis/atlantis/server/logging"
	. "github.com/runatlantis/atlantis/testing"
)

var httpApplyResult = webhooks.ApplyResult{
	Workspace: "production",
	Repo: models.Repo{
		FullName: "runatlantis/atlantis",
	},
	Pull: models.PullRequest{
		Num:        1,
		URL:        "url",
		BaseBranch: "main",
	},
	User: models.User{
		Username: "lkysow",
	},
	Success: true,
}

func TestHttpWebhookWithAuth(t *testing.T) {
	authHeader := "Bearer token"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		Equals(t, r.Header.Get("Content-Type"), "application/json")
		Equals(t, r.Header.Get("Authorization"), authHeader)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	webhook := webhooks.HttpWebhook{
		Client:         webhooks.NewHttpClient(authHeader),
		URL:            server.URL,
		WorkspaceRegex: regexp.MustCompile(".*"),
		BranchRegex:    regexp.MustCompile(".*"),
	}

	err := webhook.Send(logging.NewNoopLogger(t), httpApplyResult)
	Ok(t, err)
}

func TestHttpWebhookNoAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		Equals(t, r.Header.Get("Content-Type"), "application/json")
		Assert(t, !slices.Contains(slices.Collect(maps.Keys(r.Header)), "Authorization"), "Authorization header should be absent")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	webhook := webhooks.HttpWebhook{
		Client:         webhooks.NewHttpClient(""),
		URL:            server.URL,
		WorkspaceRegex: regexp.MustCompile(".*"),
		BranchRegex:    regexp.MustCompile(".*"),
	}

	err := webhook.Send(logging.NewNoopLogger(t), httpApplyResult)
	Ok(t, err)
}

func TestHttpWebhookDefaultClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		Equals(t, r.Header.Get("Content-Type"), "application/json")
		Assert(t, !slices.Contains(slices.Collect(maps.Keys(r.Header)), "Authorization"), "Authorization header should be absent")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	webhook := webhooks.HttpWebhook{
		Client:         http.DefaultClient,
		URL:            server.URL,
		WorkspaceRegex: regexp.MustCompile(".*"),
		BranchRegex:    regexp.MustCompile(".*"),
	}

	err := webhook.Send(logging.NewNoopLogger(t), httpApplyResult)
	Ok(t, err)
}

func TestHttpWebhook500(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	webhook := webhooks.HttpWebhook{
		Client:         webhooks.NewHttpClient(""),
		URL:            server.URL,
		WorkspaceRegex: regexp.MustCompile(".*"),
		BranchRegex:    regexp.MustCompile(".*"),
	}

	err := webhook.Send(logging.NewNoopLogger(t), httpApplyResult)
	ErrContains(t, "sending webhook", err)
}

func TestHttpNoRegexMatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		Assert(t, false, "webhook should not be sent")
	}))
	defer server.Close()

	tt := []struct {
		name string
		wr   *regexp.Regexp
		br   *regexp.Regexp
	}{
		{
			name: "no workspace match",
			wr:   regexp.MustCompile("other"),
			br:   regexp.MustCompile(".*"),
		},
		{
			name: "no branch match",
			wr:   regexp.MustCompile(".*"),
			br:   regexp.MustCompile("other"),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			webhook := webhooks.HttpWebhook{
				Client:         webhooks.NewHttpClient(""),
				URL:            server.URL,
				WorkspaceRegex: tc.wr,
				BranchRegex:    tc.br,
			}
			err := webhook.Send(logging.NewNoopLogger(t), httpApplyResult)
			Ok(t, err)
		})
	}
}
