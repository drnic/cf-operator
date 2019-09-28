package cmd

import (
	"fmt"

	"code.cloudfoundry.org/cf-operator/pkg/kube/controllers/extendedjob"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	persistOutputFileFailedMessage = "persist-output command failed."
)

// persistOutputCmd is the persist-output command.
var persistOutputCmd = &cobra.Command{
	Use:   "persist-output [flags]",
	Short: "Persist a file into a kube secret",
	Long: `Persists a log file created  by containers in a pod of extendedjob 
	
into a versionsed secret or kube native secret using flags specified to this command.
`,
	RunE: func(cmd *cobra.Command, args []string) (err error) {

		namespace := viper.GetString("cf-operator-namespace")
		if len(namespace) == 0 {
			return errors.Errorf("%s cf-operator-namespace flag is empty.", persistOutputFileFailedMessage)
		}

		outputPrefix := viper.GetString("output-prefix")

		fmt.Println(extendedjob.ConvertOutputToSecret(namespace, outputPrefix))
		return nil
	},
}

func init() {
	utilCmd.AddCommand(persistOutputCmd)

	persistOutputCmd.Flags().StringP("output-prefix", "", "", "Name to be prefixed to the secret name.")

	viper.BindPFlag("output-prefix", persistOutputCmd.Flags().Lookup("output-prefix"))

	argToEnv := map[string]string{
		"output-prefix": "OUTPUT_PREFIX",
	}
	AddEnvToUsage(persistOutputCmd, argToEnv)
}
