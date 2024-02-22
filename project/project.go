// Package project provides functionality for retrieving Google Cloud project IDs
// and related configuration.
package project

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"golang.org/x/oauth2/google"
)

var (
	defaultTimeout = 30 * time.Second
)

var (
	searchers = defaultSearchers()
)

// ID retrieves the default Google Cloud project ID based on the provided
// options.
//
// It uses the following order when searching:
//  1. Common environment variables like GCP_PROJECT, GCLOUD_PROJECT,
//     GOOGLE_CLOUD_PROJECT.
//  2. The DefaultApplicationCredentials method from the [golang.org/x/oauth2/google]
//     package.
//  3. The default project configured in `gcloud` CLI.
//
// If the project ID is empty and the Strict option is enabled, `ID()`
// panics.
//
// [golang.org/x/oauth2/google]: https://pkg.go.dev/golang.org/x/oauth2/google#FindDefaultCredentials
func ID(opts ...Options) string {
	o := getOptions(opts...)
	var (
		background  = context.Background()
		ctx, cancel = context.WithTimeout(background, o.Timeout)
	)
	defer cancel()

	id, err := defaultProjectID(ctx, o.Scopes...)
	if err != nil {
		panic(err)
	}
	if id == "" && o.Strict {
		msg := "Google Cloud project ID not found; check your credentials " +
			"file, set the GCP_PROJECT environment variable or install the " +
			"`gcloud` CLI and run `gcloud init` to configure your project."
		panic(msg)
	}

	return id
}

// Options represents the configuration options for the ID function.
type Options struct {
	// Default: 30s.
	Timeout time.Duration

	// Scopes is the list OAuth scopes.
	Scopes []string

	// If true, ID() panics when no default project ID is found.
	Strict bool
}

func getOptions(opts ...Options) Options {
	if len(opts) != 0 {
		return opts[0]
	}
	o := Options{
		Timeout: defaultTimeout,
	}
	return o
}

func defaultProjectID(ctx context.Context, scopes ...string) (string, error) {
	for _, s := range searchers {
		id, err := s.ProjectID(ctx, scopes...)
		if err != nil {
			return "", err
		}
		if id != "" {
			return id, nil
		}
	}
	return "", nil
}

func defaultSearchers() []searcher {
	return []searcher{
		// First try: check the registered environment variables.
		// Might work for some environments like Cloud Functions and
		// on premises installations.
		newEnvironmentSearcher(
			"GCP_PROJECT",
			"GCLOUD_PROJECT",
			"GOOGLE_CLOUD_PROJECT",
		),

		// Another possibility: Use the application default credentials.
		// This will search a credentials file on well know locations,
		// or issue a request to the GCE metadata server if running on
		// Google Cloud.
		newCredentialsSearcher(),

		// Last resort: try to find the project id using the gcloud cli. On
		// a local development machine this might be the only way to
		// programmatically get a projectID, if none of the environment
		// variables searched above are set. The ProjectID field of
		// Credentials is the project ID of the role. User-level credentials
		// do not have an associated project. See:
		//  - https://github.com/golang/oauth2/issues/241#issuecomment-447902482
		//  - https://github.com/googleapis/google-cloud-go/issues/1294
		newGCloudSearcher(),
	}
}

// searcher provides a search strategy for project IDs.
type searcher interface {
	ProjectID(ctx context.Context, scopes ...string) (string, error)
}

// Environment Searcher

type environmentSearcher struct {
	envLookupKeys []string
}

var _ searcher = (*environmentSearcher)(nil)

func newEnvironmentSearcher(keys ...string) *environmentSearcher {
	s := environmentSearcher{
		envLookupKeys: keys,
	}
	return &s
}

func (s *environmentSearcher) ProjectID(context.Context, ...string) (string, error) {
	for _, key := range s.envLookupKeys {
		if id := os.Getenv(key); id != "" {
			return id, nil
		}
	}
	return "", nil
}

// Default Credentials Searcher

type credentialsSearcher struct {
	findCredentialsFn func(ctx context.Context, scopes ...string) (
		*google.Credentials, error)
}

var _ searcher = (*credentialsSearcher)(nil)

func newCredentialsSearcher() *credentialsSearcher {
	s := credentialsSearcher{
		findCredentialsFn: google.FindDefaultCredentials,
	}
	return &s
}

func (s *credentialsSearcher) ProjectID(
	ctx context.Context, scopes ...string,
) (
	string, error,
) {
	credentials, err := s.findCredentialsFn(ctx, scopes...)
	if err != nil {
		err = fmt.Errorf("find credentials: %w", err)
		return "", err
	}
	id := credentials.ProjectID
	return id, nil
}

// GCloud Searcher

func commonGCloudPaths() []string {
	p, _ := exec.LookPath("gcloud")
	home, _ := os.UserHomeDir()
	paths := []string{
		p,
		"gcloud",
		path.Join(home, "google-cloud-sdk", "bin", "gcloud"),
	}
	return paths
}

type gcloudSearcher struct {
	executables []string
	output      func(cmd *exec.Cmd) ([]byte, error)
}

var _ searcher = (*gcloudSearcher)(nil)

func newGCloudSearcher() *gcloudSearcher {
	executables := commonGCloudPaths()
	s := gcloudSearcher{
		executables: executables,
		output:      cmdOutput,
	}
	return &s
}

func cmdOutput(cmd *exec.Cmd) ([]byte, error) { return cmd.Output() }

func (s *gcloudSearcher) ProjectID(
	ctx context.Context, _ ...string,
) (
	string, error,
) {
	for _, executable := range s.executables {
		gcloud := executable
		c := exec.CommandContext(
			ctx,
			gcloud,
			"config", "get-value", "project",
		)
		b, err := s.output(c)
		if err != nil {
			// Try the next possible gcloud executable path.
			continue
		}
		if len(b) != 0 {
			id := strings.TrimSpace(string(b))
			return id, nil
		}
	}

	return "", nil
}
