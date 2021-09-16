package cmd

import (
	"github.com/mheers/gitlab-ci-file-injector/helpers"
	"github.com/spf13/cobra"
)

var (
	// LogLevelFlag describes the verbosity of logs
	LogLevelFlag string

	// InputFileFlag holds the path to the input file
	InputFileFlag string

	// OutputFileFlag holds the path to the output file
	OutputFileFlag string

	// // Config holds the read config
	// Config *models.Config

	rootCmd = &cobra.Command{
		Use:   "glabci-fi",
		Short: "Gitlab CI file injector",
		Long:  ``,
		Run: func(cmd *cobra.Command, args []string) {
			helpers.PrintInfo()
			cmd.Help()
		},
	}
)

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	injectCmd.PersistentFlags().StringVarP(&InputFileFlag, "input", "i", "serviceFile.yml", "list of files that will be injected")
	injectCmd.PersistentFlags().StringVarP(&OutputFileFlag, "output", "o", ".gitlab-ci.yml", "gitlab-ci yaml file that will be augmented")
	rootCmd.PersistentFlags().StringVarP(&LogLevelFlag, "log-level", "l", "error", "possible values are debug, error, fatal, panic, info, trace")
	rootCmd.AddCommand(injectCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(completionCmd)
}
