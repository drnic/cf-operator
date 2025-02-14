package converter_test

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"code.cloudfoundry.org/cf-operator/pkg/bosh/bpm"
	"code.cloudfoundry.org/cf-operator/pkg/bosh/converter"
	"code.cloudfoundry.org/cf-operator/pkg/bosh/converter/fakes"
	"code.cloudfoundry.org/cf-operator/pkg/bosh/disk"
	"code.cloudfoundry.org/cf-operator/pkg/bosh/manifest"
	ejv1 "code.cloudfoundry.org/cf-operator/pkg/kube/apis/extendedjob/v1alpha1"
	essv1 "code.cloudfoundry.org/cf-operator/pkg/kube/apis/extendedstatefulset/v1alpha1"
	"code.cloudfoundry.org/cf-operator/pkg/kube/util"
	"code.cloudfoundry.org/cf-operator/testing"
	"code.cloudfoundry.org/cf-operator/testing/boshreleases"
)

var _ = Describe("kube converter", func() {
	var (
		m                *manifest.Manifest
		volumeFactory    *fakes.FakeVolumeFactory
		containerFactory *fakes.FakeContainerFactory
		env              testing.Catalog
		err              error
	)

	Context("BPMResources", func() {
		act := func(bpmConfigs bpm.Configs, instanceGroup *manifest.InstanceGroup) (*converter.BPMResources, error) {
			kubeConverter := converter.NewKubeConverter(
				"foo",
				volumeFactory,
				func(manifestName string, instanceGroupName string, version string, disableLogSidecar bool, releaseImageProvider converter.ReleaseImageProvider, bpmConfigs bpm.Configs) converter.ContainerFactory {
					return containerFactory
				})
			resources, err := kubeConverter.BPMResources(m.Name, "1", instanceGroup, m, bpmConfigs)
			return resources, err
		}

		BeforeEach(func() {
			m, err = env.DefaultBOSHManifest()
			Expect(err).NotTo(HaveOccurred())

			volumeFactory = &fakes.FakeVolumeFactory{}
			containerFactory = &fakes.FakeContainerFactory{}
		})

		Context("when a BPM config is present", func() {
			var bpmConfigs []bpm.Configs

			BeforeEach(func() {
				c, err := bpm.NewConfig([]byte(boshreleases.DefaultBPMConfig))
				Expect(err).ShouldNot(HaveOccurred())

				bpmConfigs = []bpm.Configs{
					{"redis-server": c},
					{"cflinuxfs3-rootfs-setup": c},
				}

			})

			Context("when the lifecycle is set to errand", func() {
				It("handles an error when generating bpm disks", func() {
					volumeFactory.GenerateBPMDisksReturns(disk.BPMResourceDisks{}, errors.New("fake-bpm-disk-error"))
					_, err := act(bpmConfigs[0], m.InstanceGroups[0])
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("Generate of BPM disks failed for manifest name %s, instance group %s.", m.Name, m.InstanceGroups[0].Name))
				})

				It("handles an error when converting jobs to initContainers", func() {
					containerFactory.JobsToInitContainersReturns([]corev1.Container{}, errors.New("fake-container-factory-error"))
					_, err := act(bpmConfigs[0], m.InstanceGroups[0])
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("building initContainers failed for instance group %s", m.InstanceGroups[0].Name))
				})

				It("handles an error when converting jobs to containers", func() {
					containerFactory.JobsToContainersReturns([]corev1.Container{}, errors.New("fake-container-factory-error"))
					_, err := act(bpmConfigs[0], m.InstanceGroups[0])
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("building containers failed for instance group %s", m.InstanceGroups[0].Name))
				})

				It("converts the instance group to an ExtendedJob", func() {
					resources, err := act(bpmConfigs[0], m.InstanceGroups[0])
					Expect(err).ShouldNot(HaveOccurred())
					Expect(resources.Errands).To(HaveLen(1))

					// Test labels and annotations in the extended job
					eJob := resources.Errands[0]
					Expect(eJob.Name).To(Equal("foo-deployment-redis-slave"))
					Expect(eJob.GetLabels()).To(HaveKeyWithValue(manifest.LabelDeploymentName, m.Name))
					Expect(eJob.GetLabels()).To(HaveKeyWithValue(manifest.LabelInstanceGroupName, m.InstanceGroups[0].Name))
					Expect(eJob.GetLabels()).To(HaveKeyWithValue(manifest.LabelDeploymentVersion, "1"))
					Expect(eJob.GetLabels()).To(HaveKeyWithValue("custom-label", "foo"))
					Expect(eJob.GetAnnotations()).To(HaveKeyWithValue("custom-annotation", "bar"))

					// Test affinity
					Expect(eJob.Spec.Template.Spec.Affinity).To(BeNil())
				})

				It("converts the instance group to an ExtendedJob when this the lifecycle is set to auto-errand", func() {
					m.InstanceGroups[0].LifeCycle = manifest.IGTypeAutoErrand
					resources, err := act(bpmConfigs[0], m.InstanceGroups[0])
					Expect(err).ShouldNot(HaveOccurred())
					Expect(resources.Errands).To(HaveLen(1))

					// Test trigger strategy
					eJob := resources.Errands[0]
					Expect(eJob.Spec.Trigger.Strategy).To(Equal(ejv1.TriggerOnce))
				})

				It("converts the AgentEnvBoshConfig information", func() {
					affinityCase := corev1.Affinity{
						NodeAffinity: &corev1.NodeAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
								NodeSelectorTerms: []corev1.NodeSelectorTerm{
									{
										MatchExpressions: []corev1.NodeSelectorRequirement{
											{
												Key:      "fake-key",
												Operator: corev1.NodeSelectorOpIn,
												Values:   []string{"fake-label"},
											},
										},
									},
								},
							},
						},
					}
					serviceAccount := "fake-service-account"
					automountServiceAccountToken := true
					m.InstanceGroups[0].Env.AgentEnvBoshConfig.Agent.Settings.Affinity = &affinityCase
					m.InstanceGroups[0].Env.AgentEnvBoshConfig.Agent.Settings.ServiceAccountName = serviceAccount
					m.InstanceGroups[0].Env.AgentEnvBoshConfig.Agent.Settings.AutomountServiceAccountToken = &automountServiceAccountToken
					resources, err := act(bpmConfigs[0], m.InstanceGroups[0])
					Expect(err).ShouldNot(HaveOccurred())
					Expect(resources.Errands).To(HaveLen(1))

					// Test AgentEnvBoshConfig
					eJob := resources.Errands[0]
					Expect(*eJob.Spec.Template.Spec.Affinity).To(Equal(affinityCase))
					Expect(eJob.Spec.Template.Spec.ServiceAccountName).To(Equal(serviceAccount))
					Expect(*eJob.Spec.Template.Spec.AutomountServiceAccountToken).To(Equal(automountServiceAccountToken))
				})
			})

			Context("when the lifecycle is set to service", func() {
				It("handles an error when converting jobs to initContainers", func() {
					containerFactory.JobsToInitContainersReturns([]corev1.Container{}, errors.New("fake-container-factory-error"))
					_, err := act(bpmConfigs[1], m.InstanceGroups[1])
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("building initContainers failed for instance group %s", m.InstanceGroups[1].Name))
				})

				It("handles an error when converting jobs to containers", func() {
					containerFactory.JobsToContainersReturns([]corev1.Container{}, errors.New("fake-container-factory-error"))
					_, err := act(bpmConfigs[1], m.InstanceGroups[1])
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("building containers failed for instance group %s", m.InstanceGroups[1].Name))
				})

				It("converts the instance group to an ExtendedStatefulSet", func() {
					resources, err := act(bpmConfigs[1], m.InstanceGroups[1])
					Expect(err).ShouldNot(HaveOccurred())

					// Test labels and annotation in the extended statefulSet
					extStS := resources.InstanceGroups[0]
					Expect(extStS.Name).To(Equal(fmt.Sprintf("%s-%s", m.Name, "diego-cell")))
					Expect(extStS.GetLabels()).To(HaveKeyWithValue(manifest.LabelDeploymentName, m.Name))
					Expect(extStS.GetLabels()).To(HaveKeyWithValue(manifest.LabelInstanceGroupName, "diego-cell"))
					Expect(extStS.GetLabels()).To(HaveKeyWithValue(manifest.LabelDeploymentVersion, "1"))

					// Test ESts spec
					Expect(extStS.Spec.Zones).To(Equal(m.InstanceGroups[1].AZs))

					stS := extStS.Spec.Template.Spec.Template
					Expect(stS.Name).To(Equal("diego-cell"))

					// Test services for the extended statefulSet
					service0 := resources.Services[0]
					Expect(service0.Name).To(Equal(fmt.Sprintf("%s-%s-0", m.Name, stS.Name)))
					Expect(service0.Spec.Selector).To(Equal(map[string]string{
						manifest.LabelInstanceGroupName: stS.Name,
						essv1.LabelAZIndex:              "0",
						essv1.LabelPodOrdinal:           "0",
					}))
					Expect(service0.Spec.Ports).To(Equal([]corev1.ServicePort{
						{
							Name:     "rep-server",
							Protocol: corev1.ProtocolTCP,
							Port:     1801,
						},
					}))

					service1 := resources.Services[1]
					Expect(service1.Name).To(Equal(fmt.Sprintf("%s-%s-1", m.Name, stS.Name)))
					Expect(service1.Spec.Selector).To(Equal(map[string]string{
						manifest.LabelInstanceGroupName: stS.Name,
						essv1.LabelAZIndex:              "1",
						essv1.LabelPodOrdinal:           "0",
					}))
					Expect(service1.Spec.Ports).To(Equal([]corev1.ServicePort{
						{
							Name:     "rep-server",
							Protocol: corev1.ProtocolTCP,
							Port:     1801,
						},
					}))

					service2 := resources.Services[2]
					Expect(service2.Name).To(Equal(fmt.Sprintf("%s-%s-2", m.Name, stS.Name)))
					Expect(service2.Spec.Selector).To(Equal(map[string]string{
						manifest.LabelInstanceGroupName: stS.Name,
						essv1.LabelAZIndex:              "0",
						essv1.LabelPodOrdinal:           "1",
					}))
					Expect(service2.Spec.Ports).To(Equal([]corev1.ServicePort{
						{
							Name:     "rep-server",
							Protocol: corev1.ProtocolTCP,
							Port:     1801,
						},
					}))

					service3 := resources.Services[3]
					Expect(service3.Name).To(Equal(fmt.Sprintf("%s-%s-3", m.Name, stS.Name)))
					Expect(service3.Spec.Selector).To(Equal(map[string]string{
						manifest.LabelInstanceGroupName: stS.Name,
						essv1.LabelAZIndex:              "1",
						essv1.LabelPodOrdinal:           "1",
					}))
					Expect(service3.Spec.Ports).To(Equal([]corev1.ServicePort{
						{
							Name:     "rep-server",
							Protocol: corev1.ProtocolTCP,
							Port:     1801,
						},
					}))

					headlessService := resources.Services[4]
					Expect(headlessService.Name).To(Equal(fmt.Sprintf("%s-%s", m.Name, stS.Name)))
					Expect(headlessService.Spec.Selector).To(Equal(map[string]string{
						manifest.LabelInstanceGroupName: stS.Name,
					}))
					Expect(headlessService.Spec.Ports).To(Equal([]corev1.ServicePort{
						{
							Name:     "rep-server",
							Protocol: corev1.ProtocolTCP,
							Port:     1801,
						},
					}))
					Expect(headlessService.Spec.ClusterIP).To(Equal("None"))

					// Test affinity
					Expect(stS.Spec.Affinity).To(BeNil())
				})
			})

			It("converts the AgentEnvBoshConfig information", func() {
				serviceAccount := "fake-service-account"
				automountServiceAccountToken := true
				m.InstanceGroups[1].Env.AgentEnvBoshConfig.Agent.Settings.ServiceAccountName = serviceAccount
				m.InstanceGroups[1].Env.AgentEnvBoshConfig.Agent.Settings.AutomountServiceAccountToken = &automountServiceAccountToken
				resources, err := act(bpmConfigs[1], m.InstanceGroups[1])
				Expect(err).ShouldNot(HaveOccurred())
				Expect(resources.InstanceGroups).To(HaveLen(1))

				// Test AgentEnvBoshConfig
				eJob := resources.InstanceGroups[0]
				Expect(eJob.Spec.Template.Spec.Template.Spec.ServiceAccountName).To(Equal(serviceAccount))
				Expect(*eJob.Spec.Template.Spec.Template.Spec.AutomountServiceAccountToken).To(Equal(automountServiceAccountToken))
			})
		})

		Context("when multiple BPM processes exist", func() {
			var (
				bpmConfigs []bpm.Configs
				bpmConfig1 bpm.Config
				bpmConfig2 bpm.Config
			)

			BeforeEach(func() {
				var err error
				m, err = env.BOSHManifestWithMultiBPMProcesses()
				Expect(err).NotTo(HaveOccurred())

				bpmConfig1, err = bpm.NewConfig([]byte(boshreleases.DefaultBPMConfig))
				Expect(err).ShouldNot(HaveOccurred())
				bpmConfig2, err = bpm.NewConfig([]byte(boshreleases.MultiProcessBPMConfig))
				Expect(err).ShouldNot(HaveOccurred())

				bpmConfigs = []bpm.Configs{
					{
						"fake-errand-a": bpmConfig1,
						"fake-errand-b": bpmConfig2,
					},
					{
						"fake-job-a": bpmConfig1,
						"fake-job-b": bpmConfig1,
						"fake-job-c": bpmConfig2,
					},
					{
						"fake-job-a": bpmConfig1,
						"fake-job-b": bpmConfig1,
						"fake-job-c": bpmConfig2,
						"fake-job-d": bpmConfig2,
					},
				}
			})

			It("creates a k8s container for each BPM process", func() {
				resources, err := act(bpmConfigs[0], m.InstanceGroups[0])
				Expect(err).ShouldNot(HaveOccurred())
				Expect(resources.Errands).To(HaveLen(1))

				resources, err = act(bpmConfigs[1], m.InstanceGroups[1])
				Expect(resources.InstanceGroups).To(HaveLen(1))
				Expect(err).ShouldNot(HaveOccurred())

				resources, err = act(bpmConfigs[2], m.InstanceGroups[2])
				Expect(err).ShouldNot(HaveOccurred())
				Expect(resources.InstanceGroups).To(HaveLen(1))
			})
		})

		Context("when the instance group name contains an underscore", func() {
			var bpmConfigs []bpm.Configs

			BeforeEach(func() {
				m, err = env.BOSHManifestCFRouting()
				Expect(err).NotTo(HaveOccurred())

				c, err := bpm.NewConfig([]byte(boshreleases.CFRouting))
				Expect(err).ShouldNot(HaveOccurred())

				bpmConfigs = []bpm.Configs{
					{"route_registrar": c},
				}

			})

			Context("when the lifecycle is set to errand", func() {
				It("converts the instance group to an ExtendedJob", func() {
					resources, err := act(bpmConfigs[0], m.InstanceGroups[0])
					Expect(err).ShouldNot(HaveOccurred())
					Expect(resources.InstanceGroups).To(HaveLen(1))
				})
			})
		})

		Context("when the job contains a persistent disk declaration", func() {
			var bpmConfigs []bpm.Configs

			BeforeEach(func() {
				m, err = env.BOSHManifestWithBPMRelease()
				Expect(err).NotTo(HaveOccurred())

				c, err := bpm.NewConfig([]byte(boshreleases.EnablePersistentDiskBPMConfig))
				Expect(err).ShouldNot(HaveOccurred())

				bpmConfigs = []bpm.Configs{
					{"test-server": c},
				}

			})

			It("converts the disks and volume declarations when instance group has persistent disk declaration", func() {
				volumeFactory.GenerateBPMDisksReturns(disk.BPMResourceDisks{
					{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaim{
							ObjectMeta: metav1.ObjectMeta{
								Name: "fake-pvc",
							},
							Spec: corev1.PersistentVolumeClaimSpec{
								StorageClassName: util.String("fake-storage-class"),
								AccessModes: []corev1.PersistentVolumeAccessMode{
									"ReadWriteOnce",
								},
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										corev1.ResourceName(corev1.ResourceStorage): resource.MustParse("1G"),
									},
								},
							},
						},
					},
				}, nil)
				resources, err := act(bpmConfigs[0], m.InstanceGroups[0])
				Expect(err).ShouldNot(HaveOccurred())

				// Test pvcs
				pvcs := resources.PersistentVolumeClaims
				Expect(pvcs[0]).NotTo(Equal(nil))
				Expect(pvcs[0].Name).To(Equal("fake-pvc"))
			})

			Context("when multiple BPM processes exist", func() {
				var (
					bpmConfigs []bpm.Configs
					bpmConfig1 bpm.Config
					bpmConfig2 bpm.Config
				)

				BeforeEach(func() {
					var err error
					m, err = env.BOSHManifestWithMultiBPMProcessesAndPersistentDisk()
					Expect(err).NotTo(HaveOccurred())

					bpmConfig1, err = bpm.NewConfig([]byte(boshreleases.DefaultBPMConfig))
					Expect(err).ShouldNot(HaveOccurred())
					bpmConfig2, err = bpm.NewConfig([]byte(boshreleases.MultiProcessBPMConfigWithPersistentDisk))
					Expect(err).ShouldNot(HaveOccurred())

					bpmConfigs = []bpm.Configs{
						{
							"fake-errand-a": bpmConfig1,
							"fake-errand-b": bpmConfig2,
						},
						{
							"fake-job-a": bpmConfig1,
							"fake-job-b": bpmConfig1,
							"fake-job-c": bpmConfig2,
						},
						{
							"fake-job-a": bpmConfig1,
							"fake-job-b": bpmConfig1,
							"fake-job-c": bpmConfig2,
							"fake-job-d": bpmConfig2,
						},
					}
				})

				It("converts correct disks and volume declarations", func() {
					containerFactory.JobsToContainersReturns([]corev1.Container{
						{
							Name: "fake-container",
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "fake-volume-name",
									MountPath: "fake-mount-path",
								},
							},
						},
					}, nil)
					resources, err := act(bpmConfigs[0], m.InstanceGroups[0])
					Expect(err).ShouldNot(HaveOccurred())
					Expect(resources.Errands).To(HaveLen(1))
					containers := resources.Errands[0].Spec.Template.Spec.Containers

					// Test shared volume setup
					Expect(containers[0].VolumeMounts[0].Name).To(Equal("fake-volume-name"))
					Expect(containers[0].VolumeMounts[0].MountPath).To(Equal("fake-mount-path"))
				})
			})
		})

		Context("when affinity is provided", func() {
			var bpmConfigs []bpm.Configs

			BeforeEach(func() {
				m, err = env.BPMReleaseWithAffinity()
				Expect(err).NotTo(HaveOccurred())

				c, err := bpm.NewConfig([]byte(boshreleases.DefaultBPMConfig))
				Expect(err).ShouldNot(HaveOccurred())

				bpmConfigs = []bpm.Configs{
					{"test-server": c},
				}

			})

			It("adds affinity into the pod's definition", func() {
				r1, err := act(bpmConfigs[0], m.InstanceGroups[0])
				Expect(err).ShouldNot(HaveOccurred())

				// Test node affinity
				ig1 := r1.InstanceGroups[0]
				Expect(ig1.Spec.Template.Spec.Template.Spec.Affinity.NodeAffinity).To(Equal(&corev1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
						NodeSelectorTerms: []corev1.NodeSelectorTerm{
							{
								MatchExpressions: []corev1.NodeSelectorRequirement{
									{
										Key:      "beta.kubernetes.io/os",
										Operator: "In",
										Values: []string{
											"linux",
											"darwin",
										},
									},
								},
							},
						},
					},
				}))

				r2, err := act(bpmConfigs[0], m.InstanceGroups[1])
				Expect(err).ShouldNot(HaveOccurred())

				// Test pod affinity
				ig2 := r2.InstanceGroups[0]
				Expect(ig2.Spec.Template.Spec.Template.Spec.Affinity.PodAffinity).To(Equal(&corev1.PodAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
						{
							LabelSelector: &metav1.LabelSelector{
								MatchExpressions: []metav1.LabelSelectorRequirement{
									{
										Key:      "instance-name",
										Operator: "In",
										Values: []string{
											"bpm2",
										},
									},
								},
							},
							TopologyKey: "beta.kubernetes.io/os",
						},
					},
				}))

				r3, err := act(bpmConfigs[0], m.InstanceGroups[2])
				Expect(err).ShouldNot(HaveOccurred())

				// Test pod anti-affinity
				ig3 := r3.InstanceGroups[0]
				Expect(ig3.Spec.Template.Spec.Template.Spec.Affinity.PodAntiAffinity).To(Equal(&corev1.PodAntiAffinity{
					PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
						{
							Weight: 100,
							PodAffinityTerm: corev1.PodAffinityTerm{
								LabelSelector: &metav1.LabelSelector{
									MatchExpressions: []metav1.LabelSelectorRequirement{
										{
											Key:      "instance-name",
											Operator: "In",
											Values: []string{
												"bpm3",
											},
										},
									},
								},
								TopologyKey: "beta.kubernetes.io/os",
							},
						},
					},
				}))
			})
		})
	})
})
