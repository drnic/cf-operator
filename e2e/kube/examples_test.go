package kube_test

import (
	b64 "encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	ejv1 "code.cloudfoundry.org/quarks-job/pkg/kube/apis/extendedjob/v1alpha1"
	"code.cloudfoundry.org/cf-operator/testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Examples", func() {

	Describe("when examples are specified in the docs", func() {

		kubectlHelper := testing.NewKubectl()

		const examplesDir = "../../docs/examples/"

		Context("all examples must be working", func() {
			It("extended-statefulset configs example must work", func() {
				yamlFilePath := examplesDir + "extended-statefulset/exstatefulset_configs.yaml"

				By("Creating exstatefulset configs")
				kubectlHelper := testing.NewKubectl()
				err := testing.Create(namespace, yamlFilePath)
				Expect(err).ToNot(HaveOccurred())

				By("Checking for pods")
				err = kubectlHelper.Wait(namespace, "ready", "pod/example-extendedstatefulset-v1-0", kubectlHelper.PollTimeout)
				Expect(err).ToNot(HaveOccurred())

				err = kubectlHelper.Wait(namespace, "ready", "pod/example-extendedstatefulset-v1-1", kubectlHelper.PollTimeout)
				Expect(err).ToNot(HaveOccurred())

				yamlUpdatedFilePath := examplesDir + "extended-statefulset/exstatefulset_configs_updated.yaml"

				By("Updating the config value used by pods")
				err = testing.Apply(namespace, yamlUpdatedFilePath)
				Expect(err).ToNot(HaveOccurred())

				By("Checking for pods")
				err = kubectlHelper.Wait(namespace, "ready", "pod/example-extendedstatefulset-v2-0", kubectlHelper.PollTimeout)
				Expect(err).ToNot(HaveOccurred())

				err = kubectlHelper.Wait(namespace, "ready", "pod/example-extendedstatefulset-v2-1", kubectlHelper.PollTimeout)
				Expect(err).ToNot(HaveOccurred())

				By("Checking the updated value in the env")
				err = kubectlHelper.RunCommandWithCheckString(namespace, "example-extendedstatefulset-v2-0", "env", "SPECIAL_KEY=value1Updated")
				Expect(err).ToNot(HaveOccurred())

				err = kubectlHelper.RunCommandWithCheckString(namespace, "example-extendedstatefulset-v2-1", "env", "SPECIAL_KEY=value1Updated")
				Expect(err).ToNot(HaveOccurred())
			})

			It("bosh-deployment service example must work", func() {
				yamlFilePath := examplesDir + "bosh-deployment/boshdeployment-with-service.yaml"

				By("Creating bosh deployment")
				kubectlHelper := testing.NewKubectl()
				err := testing.Create(namespace, yamlFilePath)
				Expect(err).ToNot(HaveOccurred())

				By("Checking for pods")
				err = kubectlHelper.Wait(namespace, "ready", "pod/nats-deployment-nats-v1-0", kubectlHelper.PollTimeout)
				Expect(err).ToNot(HaveOccurred())

				err = kubectlHelper.Wait(namespace, "ready", "pod/nats-deployment-nats-v1-1", kubectlHelper.PollTimeout)
				Expect(err).ToNot(HaveOccurred())

				err = kubectlHelper.WaitForService(namespace, "nats-service")
				Expect(err).ToNot(HaveOccurred())

				ip0, err := testing.GetData(namespace, "pod", "nats-deployment-nats-v1-0", "go-template={{.status.podIP}}")
				Expect(err).ToNot(HaveOccurred())

				ip1, err := testing.GetData(namespace, "pod", "nats-deployment-nats-v1-1", "go-template={{.status.podIP}}")
				Expect(err).ToNot(HaveOccurred())

				out, err := testing.GetData(namespace, "endpoints", "nats-service", "go-template=\"{{(index .subsets 0).addresses}}\"")
				Expect(err).ToNot(HaveOccurred())
				Expect(out).To(ContainSubstring(string(ip0)))
				Expect(out).To(ContainSubstring(string(ip1)))
			})

			It("bosh-deployment example must work", func() {
				yamlFilePath := examplesDir + "bosh-deployment/boshdeployment.yaml"

				By("Creating bosh deployment")
				kubectlHelper := testing.NewKubectl()
				err := testing.Create(namespace, yamlFilePath)
				Expect(err).ToNot(HaveOccurred())

				By("Checking for pods")
				err = kubectlHelper.Wait(namespace, "ready", "pod/nats-deployment-nats-v1-0", kubectlHelper.PollTimeout)
				Expect(err).ToNot(HaveOccurred())

				err = kubectlHelper.Wait(namespace, "ready", "pod/nats-deployment-nats-v1-1", kubectlHelper.PollTimeout)
				Expect(err).ToNot(HaveOccurred())

			})

			When("restarting operator", func() {
				It("should not create unexpected resources", func() {
					yamlFilePath := examplesDir + "bosh-deployment/boshdeployment-with-custom-variable.yaml"

					By("Creating bosh deployment")
					kubectlHelper := testing.NewKubectl()
					err := testing.Create(namespace, yamlFilePath)
					Expect(err).ToNot(HaveOccurred(), "error creating instance")

					By("Checking for pods")
					err = kubectlHelper.Wait(namespace, "ready", "pod/nats-deployment-nats-v1-0", kubectlHelper.PollTimeout)
					Expect(err).ToNot(HaveOccurred(), "error waiting for pod/nats-deployment-nats-v1-0")

					err = kubectlHelper.Wait(namespace, "ready", "pod/nats-deployment-nats-v1-1", kubectlHelper.PollTimeout)
					Expect(err).ToNot(HaveOccurred(), "error waiting for pod/nats-deployment-nats-v1-1")

					err = testing.RestartOperator(namespace)
					Expect(err).ToNot(HaveOccurred(), "error restarting cf-operator")

					By("Checking for pods not created")
					err = kubectlHelper.Wait(namespace, "ready", "pod/nats-deployment-nats-v2-0", 10*time.Second)
					Expect(err).To(HaveOccurred(), "error unexpected version of instance group is created")

					By("Checking for secrets not created")
					exist, err := kubectlHelper.SecretExists(namespace, "nats-deployment.bpm.nats-v2")
					Expect(err).ToNot(HaveOccurred(), "error getting secret/nats-deployment.bpm.nats-v2")
					Expect(exist).To(BeFalse(), "error unexpected bpm info secret is created")

					exist, err = kubectlHelper.SecretExists(namespace, "nats-deployment.desired-manifest-v2")
					Expect(err).ToNot(HaveOccurred(), "error getting secret/nats-deployment.desired-manifest-v2")
					Expect(exist).To(BeFalse(), "error unexpected desire manifest is created")

					exist, err = kubectlHelper.SecretExists(namespace, "nats-deployment.ig-resolved.nats-v2")
					Expect(err).ToNot(HaveOccurred(), "error getting secret/nats-deployment.ig-resolved.nats-v2")
					Expect(exist).To(BeFalse(), "error unexpected properties secret is created")
				})
			})

			It("bosh-deployment with a custom variable example must work", func() {
				yamlFilePath := examplesDir + "bosh-deployment/boshdeployment-with-custom-variable.yaml"

				By("Creating bosh deployment")
				kubectlHelper := testing.NewKubectl()
				err := testing.Create(namespace, yamlFilePath)
				Expect(err).ToNot(HaveOccurred())

				By("Checking for pods")
				err = kubectlHelper.Wait(namespace, "ready", "pod/nats-deployment-nats-v1-0", kubectlHelper.PollTimeout)
				Expect(err).ToNot(HaveOccurred())

				err = kubectlHelper.Wait(namespace, "ready", "pod/nats-deployment-nats-v1-1", kubectlHelper.PollTimeout)
				Expect(err).ToNot(HaveOccurred())

				By("Checking the value in the config file")
				outFile, err := testing.RunCommandWithOutput(namespace, "nats-deployment-nats-v1-1", "awk 'NR == 18 {print substr($2,2,17)}' /var/vcap/jobs/nats/config/nats.conf")
				Expect(err).ToNot(HaveOccurred())

				outSecret, err := testing.GetData(namespace, "secret", "nats-deployment.var-custom-password", "go-template={{.data.password}}")
				Expect(err).ToNot(HaveOccurred())
				outSecretDecoded, _ := b64.StdEncoding.DecodeString(string(outSecret))
				Expect(strings.TrimSuffix(outFile, "\n")).To(ContainSubstring(string(outSecretDecoded)))
			})

			It("bosh-deployment with a custom variable and logging sidecar disable example must work", func() {
				yamlFilePath := examplesDir + "bosh-deployment/boshdeployment-with-custom-variable-disable-sidecar.yaml"

				By("Creating bosh deployment")
				kubectlHelper := testing.NewKubectl()
				err := testing.Create(namespace, yamlFilePath)
				Expect(err).ToNot(HaveOccurred())

				By("Checking for pods")
				err = kubectlHelper.Wait(namespace, "ready", "pod/nats-deployment-nats-v1-0", kubectlHelper.PollTimeout)
				Expect(err).ToNot(HaveOccurred())

				err = kubectlHelper.Wait(namespace, "ready", "pod/nats-deployment-nats-v1-1", kubectlHelper.PollTimeout)
				Expect(err).ToNot(HaveOccurred())

				By("Ensure only one container exists")
				containerName, err := testing.GetData(namespace, "pod", "nats-deployment-nats-v1-0", "jsonpath={range .spec.containers[*]}{.name}")
				Expect(err).ToNot(HaveOccurred())
				Expect(containerName).To(ContainSubstring("nats-nats"))
				Expect(containerName).ToNot(ContainSubstring("logs"))

				containerName, err = testing.GetData(namespace, "pod", "nats-deployment-nats-v1-1", "jsonpath={range .spec.containers[*]}{.name}")
				Expect(err).ToNot(HaveOccurred())
				Expect(containerName).To(ContainSubstring("nats-nats"))
				Expect(containerName).ToNot(ContainSubstring("logs"))
			})

			It("bosh-deployment with a implicit variable example must work", func() {
				yamlFilePath := examplesDir + "bosh-deployment/boshdeployment-with-implicit-variable.yaml"

				By("Creating bosh deployment")
				kubectlHelper := testing.NewKubectl()
				err := testing.Create(namespace, yamlFilePath)
				Expect(err).ToNot(HaveOccurred())

				By("Checking for pods")
				err = kubectlHelper.Wait(namespace, "ready", "pod/nats-deployment-nats-v1-0", kubectlHelper.PollTimeout)
				Expect(err).ToNot(HaveOccurred())

				By("Updating implicit variable")
				implicitVariablePath := examplesDir + "bosh-deployment/implicit-variable-updated.yaml"
				err = testing.Apply(namespace, implicitVariablePath)

				Expect(err).ToNot(HaveOccurred())
				By("Checking for new pods")
				err = kubectlHelper.Wait(namespace, "ready", "pod/nats-deployment-nats-v2-0", kubectlHelper.PollTimeout)
				Expect(err).ToNot(HaveOccurred())
			})

			It("extended-job auto errand delete example must work", func() {
				yamlFilePath := examplesDir + "extended-job/exjob_auto-errand-deletes-pod.yaml"

				By("Creating exjob")
				kubectlHelper := testing.NewKubectl()
				err := testing.Create(namespace, yamlFilePath)
				Expect(err).ToNot(HaveOccurred())

				By("Checking for pods")
				err = kubectlHelper.WaitLabelFilter(namespace, "ready", "pod", fmt.Sprintf("%s=deletes-pod-1", ejv1.LabelEJobName))
				Expect(err).ToNot(HaveOccurred())

				err = kubectlHelper.WaitLabelFilter(namespace, "terminate", "pod", fmt.Sprintf("%s=deletes-pod-1", ejv1.LabelEJobName))
				Expect(err).ToNot(HaveOccurred())
			})

			It("extended-job auto errand example must work", func() {
				yamlFilePath := examplesDir + "extended-job/exjob_auto-errand.yaml"

				By("Creating exjob")
				kubectlHelper := testing.NewKubectl()
				err := testing.Create(namespace, yamlFilePath)
				Expect(err).ToNot(HaveOccurred())

				By("Checking for pods")
				err = kubectlHelper.WaitLabelFilter(namespace, "ready", "pod", fmt.Sprintf("%s=one-time-sleep", ejv1.LabelEJobName))
				Expect(err).ToNot(HaveOccurred())

				err = kubectlHelper.WaitLabelFilter(namespace, "complete", "pod", fmt.Sprintf("%s=one-time-sleep", ejv1.LabelEJobName))
				Expect(err).ToNot(HaveOccurred())
			})

			It("extended-job auto errand update example must work", func() {
				yamlFilePath := examplesDir + "extended-job/exjob_auto-errand-updating.yaml"

				By("Creating exjob")
				kubectlHelper := testing.NewKubectl()
				err := testing.Create(namespace, yamlFilePath)
				Expect(err).ToNot(HaveOccurred())

				By("Checking for pods")
				err = kubectlHelper.WaitLabelFilter(namespace, "ready", "pod", fmt.Sprintf("%s=auto-errand-sleep-again", ejv1.LabelEJobName))
				Expect(err).ToNot(HaveOccurred())

				err = kubectlHelper.WaitLabelFilter(namespace, "complete", "pod", fmt.Sprintf("%s=auto-errand-sleep-again", ejv1.LabelEJobName))
				Expect(err).ToNot(HaveOccurred())

				By("Delete the pod")
				err = testing.DeleteLabelFilter(namespace, "pod", fmt.Sprintf("%s=auto-errand-sleep-again", ejv1.LabelEJobName))
				Expect(err).ToNot(HaveOccurred())

				By("Update the config change")
				yamlFilePath = examplesDir + "extended-job/exjob_auto-errand-updating_updated.yaml"

				err = testing.Apply(namespace, yamlFilePath)
				Expect(err).ToNot(HaveOccurred())

				err = kubectlHelper.WaitLabelFilter(namespace, "ready", "pod", fmt.Sprintf("%s=auto-errand-sleep-again", ejv1.LabelEJobName))
				Expect(err).ToNot(HaveOccurred())

				err = kubectlHelper.WaitLabelFilter(namespace, "complete", "pod", fmt.Sprintf("%s=auto-errand-sleep-again", ejv1.LabelEJobName))
				Expect(err).ToNot(HaveOccurred())
			})

			It("extended-job errand example must work", func() {
				yamlFilePath := examplesDir + "extended-job/exjob_errand.yaml"

				By("Creating exjob")
				kubectlHelper := testing.NewKubectl()
				err := testing.Create(namespace, yamlFilePath)
				Expect(err).ToNot(HaveOccurred())

				By("Updating exjob to trigger now")
				yamlFilePath = examplesDir + "extended-job/exjob_errand_updated.yaml"
				err = testing.Apply(namespace, yamlFilePath)
				Expect(err).ToNot(HaveOccurred())

				By("Checking for pods")
				err = kubectlHelper.WaitLabelFilter(namespace, "ready", "pod", fmt.Sprintf("%s=manual-sleep", ejv1.LabelEJobName))
				Expect(err).ToNot(HaveOccurred())

				err = kubectlHelper.WaitLabelFilter(namespace, "complete", "pod", fmt.Sprintf("%s=manual-sleep", ejv1.LabelEJobName))
				Expect(err).ToNot(HaveOccurred())
			})

			It("extended-job output example must work", func() {
				yamlFilePath := examplesDir + "extended-job/exjob_output.yaml"

				By("Creating exjob")
				kubectlHelper := testing.NewKubectl()
				err := testing.Create(namespace, yamlFilePath)
				Expect(err).ToNot(HaveOccurred())

				By("Checking for pods")
				err = kubectlHelper.WaitLabelFilter(namespace, "complete", "pod", fmt.Sprintf("%s=myfoo", ejv1.LabelEJobName))
				Expect(err).ToNot(HaveOccurred())

				By("Checking for secret")
				err = kubectlHelper.WaitForSecret(namespace, "foo-json")
				Expect(err).ToNot(HaveOccurred())

				By("Checking the secret data created")
				outSecret, err := testing.GetData(namespace, "secret", "foo-json", "go-template={{.data.foo}}")
				Expect(err).ToNot(HaveOccurred())
				outSecretDecoded, _ := b64.StdEncoding.DecodeString(string(outSecret))
				Expect(string(outSecretDecoded)).To(Equal("1"))
			})

			It("extended-secret example must work", func() {
				yamlFilePath := examplesDir + "extended-secret/password.yaml"

				By("Creating an ExtendedSecret")
				err := testing.Create(namespace, yamlFilePath)
				Expect(err).ToNot(HaveOccurred())

				By("Checking the generated password")
				err = testing.SecretCheckData(namespace, "gen-secret1", ".data.password")
				Expect(err).ToNot(HaveOccurred())
			})

			It("API server signed certificate example must work", func() {
				yamlFilePath := examplesDir + "extended-secret/certificate.yaml"

				By("Creating an ExtendedSecret")
				err := testing.Create(namespace, yamlFilePath)
				Expect(err).ToNot(HaveOccurred())

				By("Checking the generated certificate")
				err = kubectlHelper.WaitForSecret(namespace, "gen-certificate")
				Expect(err).ToNot(HaveOccurred(), "error waiting for secret")
				err = testing.SecretCheckData(namespace, "gen-certificate", ".data.certificate")
				Expect(err).ToNot(HaveOccurred(), "error getting for secret")
			})

			It("Self signed certificate example must work", func() {
				caYamlFilePath := examplesDir + "extended-secret/loggregator-ca-cert.yaml"
				certYamlFilePath := examplesDir + "extended-secret/loggregator-tls-agent-cert.yaml"

				By("Creating ExtendedSecrets")
				err := testing.Create(namespace, caYamlFilePath)
				Expect(err).ToNot(HaveOccurred())
				err = testing.Create(namespace, certYamlFilePath)
				Expect(err).ToNot(HaveOccurred())

				By("Checking the generated certificates")
				err = kubectlHelper.WaitForSecret(namespace, "example.var-loggregator-ca")
				Expect(err).ToNot(HaveOccurred(), "error waiting for ca secret")
				err = kubectlHelper.WaitForSecret(namespace, "example.var-loggregator-tls-agent")
				Expect(err).ToNot(HaveOccurred(), "error waiting for cert secret")

				By("Checking the generated certificates")
				outSecret, err := testing.GetData(namespace, "secret", "example.var-loggregator-ca", "go-template={{.data.certificate}}")
				Expect(err).ToNot(HaveOccurred())
				rootPEM, _ := b64.StdEncoding.DecodeString(string(outSecret))

				outSecret, err = testing.GetData(namespace, "secret", "example.var-loggregator-tls-agent", "go-template={{.data.certificate}}")
				Expect(err).ToNot(HaveOccurred())
				certPEM, _ := b64.StdEncoding.DecodeString(string(outSecret))

				By("Verify the certificates")
				dnsName := "metron"
				err = testing.CertificateVerify(rootPEM, certPEM, dnsName)
				Expect(err).ToNot(HaveOccurred(), "error verifying certificates")
			})

			It("Test cases must be written for all example use cases in docs", func() {
				countFile := 0
				err := filepath.Walk(examplesDir, func(path string, info os.FileInfo, err error) error {
					if !info.IsDir() {
						countFile = countFile + 1
					}
					return nil
				})
				Expect(err).NotTo(HaveOccurred())
				// If this testcase fails that means a test case is missing for an example in the docs folder
				Expect(countFile).To(Equal(27))
			})
		})
	})
})
