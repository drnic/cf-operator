package cmd

import (
	"code.cloudfoundry.org/cf-operator/pkg/kube/controllers/extendedjob"
	"github.com/spf13/cobra"
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

		return extendedjob.ConvertOutputToSecret()
	},
}

func init() {
	utilCmd.AddCommand(persistOutputCmd)

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

	//	argToEnv := map[string]string{
	/*"pod-ip":                  converter.PodIPEnvVar,
	"jobs-dir":                "JOBS_DIR",
	"output-dir":              "OUTPUT_DIR",
	"docker-image-repository": "DOCKER_IMAGE_REPOSITORY",
	"spec-index":              "SPEC_INDEX",
	"az-index":                "AZ_INDEX",
	"pod-ordinal":             "POD_ORDINAL",
	"replicas":                "REPLICAS",*/
	//		"ejob-name": "EJOB_NAME",
	//	}
	//	AddEnvToUsage(persistOutputCmd, argToEnv)
}
