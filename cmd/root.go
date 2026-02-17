package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "data-channel-service",
	Short: "Data channel: WebSocket chat/annotations, history, files",
	RunE:  runAPI,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(apiCmd)
	rootCmd.AddCommand(migrateCmd)
}
