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
	Long: `Persists a log file into a versionsed secret or kube native secret using flags 

specified to this command.
`,
	RunE: func(cmd *cobra.Command, args []string) (err error) {

		fmt.Println("Running command persist output")

		namespace := viper.GetString("namespace")
		if namespace == "" {
			return errors.Errorf("%s namespace is empty", persistOutputFileFailedMessage)
		}

		fmt.Println("got namespace", namespace)

		return extendedjob.ConvertOutputToSecret(namespace)
	},
}

func init() {
	utilCmd.AddCommand(persistOutputCmd)

	persistOutputCmd.Flags().StringP("namespace", "", "", "Kubernetes namespace in which cf-operator runs")

	viper.BindPFlag("namespace", persistOutputCmd.Flags().Lookup("namespace"))

	argToEnv := map[string]string{
		"namespace": "NAMESPACE",
	}
	AddEnvToUsage(persistOutputCmd, argToEnv)

	/*plateRenderCmd.Flags().StringP("output-dir", "d", converter.VolumeJobsDirMountPath, "path to output dir.")
	templateRenderCmd.Flags().IntP("spec-index", "", -1, "index of the instance spec")
	templateRenderCmd.Flags().IntP("az-index", "", -1, "az index")
	templateRenderCmd.Flags().IntP("pod-ordinal", "", -1, "pod ordinal")
	templateRenderCmd.Flags().IntP("replicas", "", -1, "number of replicas")
	templateRenderCmd.Flags().StringP("pod-name", "", "", "pod name")
	templateRenderCmd.Flags().StringP("pod-ip", "", "", "pod IP")

	viper.BindPFlag("jobs-dir", templateRenderCmd.Flags().Lookup("jobs-dir"))
	viper.BindPFlag("output-dir", templateRenderCmd.Flags().Lookup("output-dir"))
	viper.BindPFlag("az-index", templateRenderCmd.Flags().Lookup("az-index"))
	viper.BindPFlag("spec-index", templateRenderCmd.Flags().Lookup("spec-index"))
	viper.BindPFlag("pod-ordinal", templateRenderCmd.Flags().Lookup("pod-ordinal"))
	viper.BindPFlag("replicas", templateRenderCmd.Flags().Lookup("replicas"))
	viper.BindPFlag("pod-ip", templateRenderCmd.Flags().Lookup("pod-ip"))*/

}
