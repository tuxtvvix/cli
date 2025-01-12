package view

import (
	"fmt"

	"github.com/MakeNowJust/heredoc"
	"github.com/cli/cli/v2/internal/browser"
	"github.com/cli/cli/v2/internal/ghrepo"
	"github.com/cli/cli/v2/pkg/cmd/repo/autolink/domain"
	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/cli/cli/v2/pkg/iostreams"
	"github.com/spf13/cobra"
)

type viewOptions struct {
	BaseRepo       func() (ghrepo.Interface, error)
	Browser        browser.Browser
	AutolinkClient AutolinkViewClient
	IO             *iostreams.IOStreams
	Exporter       cmdutil.Exporter

	ID string
}

type AutolinkViewClient interface {
	View(repo ghrepo.Interface, id string) (*domain.Autolink, error)
}

func NewCmdView(f *cmdutil.Factory, runF func(*viewOptions) error) *cobra.Command {
	opts := &viewOptions{
		Browser: f.Browser,
		IO:      f.IOStreams,
	}

	cmd := &cobra.Command{
		Use:   "view <id>",
		Short: "View an autolink reference",
		Long: heredoc.Docf(`
			View an autolink reference for a repository.

			Information about autolinks is only available to repository administrators.
		`),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.BaseRepo = f.BaseRepo
			httpClient, err := f.HttpClient()

			if err != nil {
				return err
			}

			opts.ID = args[0]

			if err != nil {
				return err
			}

			opts.AutolinkClient = &AutolinkViewer{HTTPClient: httpClient}

			if runF != nil {
				return runF(opts)
			}

			return viewRun(opts)
		},
	}

	cmdutil.AddJSONFlags(cmd, &opts.Exporter, domain.AutolinkFields)

	return cmd
}

func viewRun(opts *viewOptions) error {
	repo, err := opts.BaseRepo()
	if err != nil {
		return err
	}
	cs := opts.IO.ColorScheme()

	autolink, err := opts.AutolinkClient.View(repo, opts.ID)

	if err != nil {
		return fmt.Errorf("%s %w", cs.Red("error viewing autolink:"), err)
	}

	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IO, autolink)
	}

	msg := heredoc.Docf(`
			Autolink in %s

			ID: %d
			Key Prefix: %s
			URL Template: %s
			Alphanumeric: %t
		`,
		ghrepo.FullName(repo),
		autolink.ID,
		autolink.KeyPrefix,
		autolink.URLTemplate,
		autolink.IsAlphanumeric,
	)
	fmt.Fprint(opts.IO.Out, msg)

	return nil
}
