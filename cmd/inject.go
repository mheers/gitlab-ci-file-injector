package cmd

import (
	"github.com/mheers/gitlab-ci-file-injector/helpers"
	"github.com/mheers/gitlab-ci-file-injector/injector"
	"github.com/spf13/cobra"
)

var (
	injectCmd = &cobra.Command{
		Use:   "inject",
		Short: "injects the file contents",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			helpers.SetLogLevel(LogLevelFlag)

			helpers.PrintInfo()
			printVersion()

			sf, err := injector.ReadServiceFiles(InputFileFlag)
			if err != nil {
				return err
			}
			err = injector.InjectFilesIntoGitlabCIYaml(sf, OutputFileFlag)
			if err != nil {
				return err
			}

			return nil
		},
	}
)
