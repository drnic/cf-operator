package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

const (
	persistLogFileFailedMessage = "persist-log-file command failed."
)

// persistLogFileCmd is the persist-log-file command.
var persistLogFileCmd = &cobra.Command{
	Use:   "persist-log-file [flags]",
	Short: "Persist a log file",
	Long: `Persists a log file into secret.

Based on the values of the flag it will create versioned or normal kubernetes secret.
`,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		fmt.Println("Nothing as of now")
		return nil
	},
}

func init() {
	utilCmd.AddCommand(persistLogFileCmd)
}
