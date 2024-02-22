package project

import (
	"context"
	"errors"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2/google"
)

func TestID(t *testing.T) {
	tests := []struct {
		name        string
		opts        Options
		expectedID  bool
		expectError bool
		expectPanic bool
	}{
		{
			name:        "Default project ID found",
			opts:        Options{},
			expectedID:  true,
			expectError: false,
			expectPanic: false,
		},
		{
			name:        "Empty default project ID",
			opts:        Options{},
			expectedID:  false,
			expectError: false,
			expectPanic: false,
		},
		{
			name:        "Error retrieving default project ID",
			opts:        Options{},
			expectedID:  false,
			expectError: true,
			expectPanic: true,
		},
		{
			name:        "Empty default project ID and strict mode",
			opts:        Options{Strict: true},
			expectedID:  false,
			expectError: false,
			expectPanic: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			searchers = []searcher{
				newSearcherMock(test.expectedID, test.expectError),
			}

			if test.expectPanic {
				assert.Panics(t, func() { ID(test.opts) })
			}
			if test.expectedID {
				assert.NotEmpty(t, ID(test.opts))
			}
		})
	}
}

type searcherMock struct {
	projectID string
	wantError bool
}

var _ searcher = (*searcherMock)(nil)

func (s *searcherMock) ProjectID(context.Context, ...string) (string, error) {
	if s.wantError {
		return "", errors.New("test error")
	}
	return s.projectID, nil
}

func newSearcherMock(wantID, wantError bool) searcher {
	s := searcherMock{
		wantError: wantError,
	}
	if wantID {
		s.projectID = "gcp-project-id"
	}
	return &s
}

// Environment Searcher

func Test_environmentSearcher_ProjectID(t *testing.T) {
	type fields struct {
		newEnvLookupKeys func() []string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "",
			fields: fields{
				newEnvLookupKeys: func() []string {
					key := "__GCP_PROJECT_ID_TEST__"
					t.Setenv(key, "gcp-id-test")
					return []string{key}
				},
			},
			want: "gcp-id-test",
		},
		{
			name: "",
			fields: fields{
				newEnvLookupKeys: func() []string { return nil },
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &environmentSearcher{
				envLookupKeys: tt.fields.newEnvLookupKeys(),
			}

			got, err := s.ProjectID(context.Background())

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

// Default Credentials Searcher

func Test_credentialsSearcher_ProjectID(t *testing.T) {
	type fields struct {
		findCredentialsFn func(ctx context.Context, scopes ...string) (
			*google.Credentials, error)
	}
	type args struct {
		ctx    context.Context
		scopes []string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{
		{
			name: "",
			fields: fields{
				findCredentialsFn: func(context.Context, ...string) (
					*google.Credentials, error,
				) {
					c := google.Credentials{
						ProjectID: "gcp-id-test",
					}
					return &c, nil
				},
			},
			args: args{ctx: context.Background(), scopes: nil},
			want: "gcp-id-test",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &credentialsSearcher{
				findCredentialsFn: tt.fields.findCredentialsFn,
			}

			got, err := s.ProjectID(tt.args.ctx, tt.args.scopes...)

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_credentialsSearcher_ProjectID_Error(t *testing.T) {
	type fields struct {
		findCredentialsFn func(ctx context.Context, scopes ...string) (
			*google.Credentials, error)
	}
	type args struct {
		ctx    context.Context
		scopes []string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "",
			fields: fields{
				findCredentialsFn: func(context.Context, ...string) (
					*google.Credentials, error,
				) {
					return nil, errors.New("test error")
				},
			},
			args: args{ctx: context.Background(), scopes: nil},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &credentialsSearcher{
				findCredentialsFn: tt.fields.findCredentialsFn,
			}

			_, err := s.ProjectID(tt.args.ctx, tt.args.scopes...)

			require.Error(t, err)
		})
	}
}

// GCloud Searcher

func Test_gcloudSearcher_ProjectID(t *testing.T) {
	var useGCloud bool
	gcloud, _ := exec.LookPath("gcloud")
	if gcloud != "" {
		useGCloud = true
	} else {
		t.Log("[WARN] gcloud command not found in PATH. Is it installed?" +
			"Tests will run with a mock.")
	}

	t.Run("", func(t *testing.T) {
		s := &gcloudSearcher{
			executables: []string{gcloud},
			output:      cmdOutput,
		}
		if !useGCloud {
			s.output = func(cmd *exec.Cmd) ([]byte, error) {
				return []byte("gcp-id-test"), nil
			}
		}

		got, err := s.ProjectID(context.Background())

		require.NoError(t, err)
		assert.NotEmpty(t, got)
	})

	t.Run("", func(t *testing.T) {
		s := &gcloudSearcher{
			executables: commonGCloudPaths(),
			output:      cmdOutput,
		}

		got, err := s.ProjectID(context.Background())

		require.NoError(t, err)
		assert.NotEmpty(t, got)
	})

	t.Run("", func(t *testing.T) {
		s := &gcloudSearcher{
			executables: []string{"_"},
			output:      cmdOutput,
		}

		got, err := s.ProjectID(context.Background())

		require.NoError(t, err)
		assert.Empty(t, got)
	})
}

// Other

func TestGetOptions(t *testing.T) {
	tests := []struct {
		name     string
		input    []Options
		expected Options
	}{
		{
			name:     "No options provided",
			input:    nil,
			expected: Options{Timeout: defaultTimeout},
		},
		{
			name:     "Timeout option provided",
			input:    []Options{{Timeout: 10 * time.Second}},
			expected: Options{Timeout: 10 * time.Second},
		},
		{
			name:     "Timeout and Scopes options provided",
			input:    []Options{{Timeout: 5 * time.Second, Scopes: []string{"read"}}},
			expected: Options{Timeout: 5 * time.Second, Scopes: []string{"read"}},
		},
		{
			name:     "Multiple options provided, only first should be considered",
			input:    []Options{{Timeout: 15 * time.Second}, {Timeout: 20 * time.Second}},
			expected: Options{Timeout: 15 * time.Second},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := getOptions(test.input...)
			assert.Equal(t, test.expected, actual)
		})
	}
}
