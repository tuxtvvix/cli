package delete

import (
	"bytes"
	"errors"
	"net/http"
	"testing"

	"github.com/MakeNowJust/heredoc"
	"github.com/cli/cli/v2/internal/browser"
	"github.com/cli/cli/v2/internal/ghrepo"
	"github.com/cli/cli/v2/internal/prompter"
	"github.com/cli/cli/v2/pkg/cmd/repo/autolink/shared"
	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/cli/cli/v2/pkg/iostreams"
	"github.com/google/shlex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCmdDelete(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		output  deleteOptions
		wantErr bool
		errMsg  string
	}{
		{
			name:    "no argument",
			input:   "",
			wantErr: true,
			errMsg:  "accepts 1 arg(s), received 0",
		},
		{
			name:   "id provided",
			input:  "123",
			output: deleteOptions{ID: "123"},
		},
		{
			name:   "yes flag",
			input:  "123 --yes",
			output: deleteOptions{ID: "123", Confirmed: true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ios, _, _, _ := iostreams.Test()
			f := &cmdutil.Factory{
				IOStreams: ios,
			}
			f.HttpClient = func() (*http.Client, error) {
				return &http.Client{}, nil
			}

			argv, err := shlex.Split(tt.input)
			require.NoError(t, err)

			var gotOpts *deleteOptions
			cmd := NewCmdDelete(f, func(opts *deleteOptions) error {
				gotOpts = opts
				return nil
			})

			cmd.SetArgs(argv)
			cmd.SetIn(&bytes.Buffer{})
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetErr(&bytes.Buffer{})

			_, err = cmd.ExecuteC()
			if tt.wantErr {
				require.EqualError(t, err, tt.errMsg)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.output.ID, gotOpts.ID)
				assert.Equal(t, tt.output.Confirmed, gotOpts.Confirmed)
			}
		})
	}
}

type stubAutolinkDeleter struct {
	err error
}

func (d *stubAutolinkDeleter) Delete(repo ghrepo.Interface, id string) error {
	return d.err
}

type stubAutolinkViewer struct {
	autolink *shared.Autolink
	err      error
}

func (g stubAutolinkViewer) View(repo ghrepo.Interface, id string) (*shared.Autolink, error) {
	return g.autolink, g.err
}

var errTestPrompt = errors.New("prompt error")
var errTestAutolinkClientView = errors.New("autolink client view error")
var errTestAutolinkClientDelete = errors.New("autolink client delete error")

func TestDeleteRun(t *testing.T) {
	tests := []struct {
		name          string
		opts          *deleteOptions
		stubDeleter   stubAutolinkDeleter
		stubViewer    stubAutolinkViewer
		prompterStubs func(*prompter.PrompterMock)

		wantStdout     string
		wantStderr     string
		expectedErr    error
		expectedErrMsg string
	}{
		{
			name: "delete",
			opts: &deleteOptions{
				ID: "123",
			},
			stubViewer: stubAutolinkViewer{
				autolink: &shared.Autolink{
					ID:             123,
					KeyPrefix:      "TICKET-",
					URLTemplate:    "https://example.com/TICKET?query=<num>",
					IsAlphanumeric: true,
				},
			},
			stubDeleter: stubAutolinkDeleter{
				err: nil,
			},
			prompterStubs: func(pm *prompter.PrompterMock) {
				pm.ConfirmDeletionFunc = func(_ string) error {
					return nil
				}
			},
			wantStdout: heredoc.Doc(`
				Autolink 123 has key prefix TICKET-.
				✓ Autolink 123 deleted from OWNER/REPO
			`),
		},
		{
			name: "delete with confirm flag",
			opts: &deleteOptions{
				ID:        "123",
				Confirmed: true,
			},
			stubViewer: stubAutolinkViewer{
				autolink: &shared.Autolink{
					ID:             123,
					KeyPrefix:      "TICKET-",
					URLTemplate:    "https://example.com/TICKET?query=<num>",
					IsAlphanumeric: true,
				},
			},
			stubDeleter: stubAutolinkDeleter{
				err: nil,
			},
			wantStdout: "✓ Autolink 123 deleted from OWNER/REPO\n",
		},
		{
			name: "confirmation fails",
			opts: &deleteOptions{
				ID: "123",
			},
			stubViewer: stubAutolinkViewer{
				autolink: &shared.Autolink{
					ID:             123,
					KeyPrefix:      "TICKET-",
					URLTemplate:    "https://example.com/TICKET?query=<num>",
					IsAlphanumeric: true,
				},
			},
			stubDeleter: stubAutolinkDeleter{
				err: nil,
			},
			prompterStubs: func(pm *prompter.PrompterMock) {
				pm.ConfirmDeletionFunc = func(_ string) error {
					return errTestPrompt
				}
			},
			expectedErr:    errTestPrompt,
			expectedErrMsg: errTestPrompt.Error(),
			wantStdout:     "Autolink 123 has key prefix TICKET-.\n",
		},
		{
			name: "view error",
			opts: &deleteOptions{
				ID: "123",
			},
			stubViewer: stubAutolinkViewer{
				err: errTestAutolinkClientView,
			},
			stubDeleter: stubAutolinkDeleter{
				err: nil,
			},
			expectedErr:    errTestAutolinkClientView,
			expectedErrMsg: "error deleting autolink: autolink client view error",
		},
		{
			name: "delete error",
			opts: &deleteOptions{
				ID: "123",
			},
			stubViewer: stubAutolinkViewer{
				autolink: &shared.Autolink{
					ID:             123,
					KeyPrefix:      "TICKET-",
					URLTemplate:    "https://example.com/TICKET?query=<num>",
					IsAlphanumeric: true,
				},
			},
			stubDeleter: stubAutolinkDeleter{
				err: errTestAutolinkClientDelete,
			},
			prompterStubs: func(pm *prompter.PrompterMock) {
				pm.ConfirmDeletionFunc = func(_ string) error {
					return nil
				}
			},
			expectedErr:    errTestAutolinkClientDelete,
			expectedErrMsg: errTestAutolinkClientDelete.Error(),
			wantStdout:     "Autolink 123 has key prefix TICKET-.\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ios, _, stdout, stderr := iostreams.Test()

			opts := tt.opts
			opts.IO = ios
			opts.Browser = &browser.Stub{}

			opts.IO = ios
			opts.BaseRepo = func() (ghrepo.Interface, error) { return ghrepo.New("OWNER", "REPO"), nil }

			opts.AutolinkDeleteClient = &tt.stubDeleter
			opts.AutolinkViewClient = &tt.stubViewer

			pm := &prompter.PrompterMock{}
			if tt.prompterStubs != nil {
				tt.prompterStubs(pm)
			}
			tt.opts.Prompter = pm

			err := deleteRun(opts)

			if tt.expectedErr != nil {
				require.Error(t, err, "expected error but got none")
				assert.ErrorIs(t, err, tt.expectedErr, "unexpected error")
				assert.Equal(t, tt.expectedErrMsg, err.Error(), "unexpected error message")
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.wantStdout, stdout.String(), "unexpected stdout")
			assert.Equal(t, tt.wantStderr, stderr.String(), "unexpected stderr")
		})
	}
}
