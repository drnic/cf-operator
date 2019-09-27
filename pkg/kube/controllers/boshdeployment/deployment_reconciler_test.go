package boshdeployment_test

import (
	"context"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	crc "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"code.cloudfoundry.org/cf-operator/pkg/bosh/converter/fakes"
	bdm "code.cloudfoundry.org/cf-operator/pkg/bosh/manifest"
	bdv1 "code.cloudfoundry.org/cf-operator/pkg/kube/apis/boshdeployment/v1alpha1"
	ejv1 "code.cloudfoundry.org/quarks-job/pkg/kube/apis/extendedjob/v1alpha1"
	"code.cloudfoundry.org/cf-operator/pkg/kube/controllers"
	cfd "code.cloudfoundry.org/cf-operator/pkg/kube/controllers/boshdeployment"
	cfakes "code.cloudfoundry.org/cf-operator/pkg/kube/controllers/fakes"
	cfcfg "code.cloudfoundry.org/cf-operator/pkg/kube/util/config"
	"code.cloudfoundry.org/cf-operator/pkg/kube/util/ctxlog"
	helper "code.cloudfoundry.org/cf-operator/pkg/testhelper"
)

var _ = Describe("ReconcileBoshDeployment", func() {
	var (
		manager    *cfakes.FakeManager
		reconciler reconcile.Reconciler
		recorder   *record.FakeRecorder
		request    reconcile.Request
		ctx        context.Context
		resolver   fakes.FakeResolver
		jobFactory fakes.FakeJobFactory
		manifest   *bdm.Manifest
		log        *zap.SugaredLogger
		config     *cfcfg.Config
		client     *cfakes.FakeClient
		instance   *bdv1.BOSHDeployment
		dmEJob     *ejv1.ExtendedJob
		igEJob     *ejv1.ExtendedJob
		bpmEJob    *ejv1.ExtendedJob
	)

	BeforeEach(func() {
		controllers.AddToScheme(scheme.Scheme)
		recorder = record.NewFakeRecorder(20)
		manager = &cfakes.FakeManager{}
		manager.GetSchemeReturns(scheme.Scheme)
		manager.GetEventRecorderForReturns(recorder)
		resolver = fakes.FakeResolver{}
		jobFactory = fakes.FakeJobFactory{}

		request = reconcile.Request{NamespacedName: types.NamespacedName{Name: "foo", Namespace: "default"}}
		manifest = &bdm.Manifest{
			Name: "foo",
			Releases: []*bdm.Release{
				{
					Name:    "bar",
					URL:     "docker.io/cfcontainerization",
					Version: "1.0",
					Stemcell: &bdm.ReleaseStemcell{
						OS:      "opensuse",
						Version: "42.3",
					},
				},
			},
			InstanceGroups: []*bdm.InstanceGroup{
				{
					Name: "fakepod",
					Jobs: []bdm.Job{
						{
							Name:    "foo",
							Release: "bar",
							Properties: bdm.JobProperties{
								Properties: map[string]interface{}{
									"password": "((foo_password))",
								},
								Quarks: bdm.Quarks{
									Ports: []bdm.Port{
										{
											Name:     "foo",
											Protocol: "TCP",
											Internal: 8080,
										},
									},
								},
							},
						},
					},
				},
			},
			Variables: []bdm.Variable{
				{
					Name: "foo_password",
					Type: "password",
				},
			},
		}
		dmEJob = &ejv1.ExtendedJob{
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("dm-%s", manifest.Name),
			},
		}
		igEJob = &ejv1.ExtendedJob{
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("ig-%s", manifest.Name),
			},
		}
		bpmEJob = &ejv1.ExtendedJob{
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("bpm-%s", manifest.Name),
			},
		}
		config = &cfcfg.Config{CtxTimeOut: 10 * time.Second}
		_, log = helper.NewTestLogger()
		ctx = ctxlog.NewParentContext(log)
		ctx = ctxlog.NewContextWithRecorder(ctx, "TestRecorder", recorder)

		instance = &bdv1.BOSHDeployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "default",
			},
			Spec: bdv1.BOSHDeploymentSpec{
				Manifest: bdv1.ResourceReference{
					Name: "dummy-manifest",
					Type: "configmap",
				},
				Ops: []bdv1.ResourceReference{
					{
						Name: "bar",
						Type: "configmap",
					},
					{
						Name: "baz",
						Type: "secret",
					},
				},
			},
		}

		client = &cfakes.FakeClient{}
		client.GetCalls(func(context context.Context, nn types.NamespacedName, object runtime.Object) error {
			switch object := object.(type) {
			case *bdv1.BOSHDeployment:
				instance.DeepCopyInto(object)
			case *ejv1.ExtendedJob:
				return apierrors.NewNotFound(schema.GroupResource{}, nn.Name)
			}

			return nil
		})

		manager.GetClientReturns(client)
	})

	JustBeforeEach(func() {
		resolver.WithOpsManifestReturns(manifest, []string{}, nil)
		reconciler = cfd.NewDeploymentReconciler(ctx, config, manager,
			&resolver, &jobFactory, controllerutil.SetControllerReference,
		)
	})

	Describe("Reconcile", func() {
		Context("when the manifest can not be resolved", func() {
			It("returns an empty result when the resource was not found", func() {
				client.GetReturns(apierrors.NewNotFound(schema.GroupResource{}, "not found is requeued"))

				reconciler.Reconcile(request)
				result, err := reconciler.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())
				Expect(reconcile.Result{}).To(Equal(result))
			})

			It("handles an error when the request failed", func() {
				client.GetReturns(apierrors.NewBadRequest("bad request returns error"))

				_, err := reconciler.Reconcile(request)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("bad request returns error"))

				// check for events
				Expect(<-recorder.Events).To(ContainSubstring("GetBOSHDeploymentError"))
			})

			It("handles an error when resolving the BOSHDeployment", func() {
				resolver.WithOpsManifestReturns(nil, []string{}, fmt.Errorf("resolver error"))

				_, err := reconciler.Reconcile(request)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("resolver error"))

				// check for events
				Expect(<-recorder.Events).To(ContainSubstring("WithOpsManifestError"))
			})
		})

		Context("when the manifest can be resolved", func() {
			It("handles an error when resolving manifest", func() {
				manifest = &bdm.Manifest{}
				resolver.WithOpsManifestReturns(manifest, []string{}, errors.New("fake-error"))

				_, err := reconciler.Reconcile(request)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("error resolving the manifest foo: fake-error"))
			})

			It("handles an error when setting the owner reference on the object", func() {
				reconciler = cfd.NewDeploymentReconciler(ctx, config, manager, &resolver, &jobFactory,
					func(owner, object metav1.Object, scheme *runtime.Scheme) error {
						return fmt.Errorf("some error")
					},
				)

				_, err := reconciler.Reconcile(request)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to set ownerReference for Secret 'foo.with-ops': some error"))
			})

			It("handles an error when creating manifest secret with ops ", func() {
				client.GetCalls(func(context context.Context, nn types.NamespacedName, object runtime.Object) error {
					switch object := object.(type) {
					case *bdv1.BOSHDeployment:
						instance.DeepCopyInto(object)
					case *ejv1.ExtendedJob:
						return apierrors.NewNotFound(schema.GroupResource{}, nn.Name)
					case *corev1.Secret:
						if nn.Name == "foo.with-ops" {
							return apierrors.NewNotFound(schema.GroupResource{}, nn.Name)
						}
					}

					return nil
				})
				client.UpdateCalls(func(context context.Context, object runtime.Object, _ ...crc.UpdateOption) error {
					switch object := object.(type) {
					case *bdv1.BOSHDeployment:
						object.DeepCopyInto(instance)
					}
					return nil
				})
				client.CreateCalls(func(context context.Context, object runtime.Object, _ ...crc.CreateOption) error {
					switch object.(type) {
					case *corev1.Secret:
						return errors.New("fake-error")
					}
					return nil
				})

				By("From created state to ops applied state")
				_, err := reconciler.Reconcile(request)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to create with-ops manifest secret for BOSHDeployment 'default/foo': failed to apply Secret 'foo.with-ops': fake-error"))
			})

			It("handles an error when building desired manifest eJob", func() {
				jobFactory.VariableInterpolationJobReturns(dmEJob, errors.New("fake-error"))

				_, err := reconciler.Reconcile(request)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to build the desired manifest eJob"))
			})

			It("handles an error when building desired manifest eJob", func() {
				jobFactory.VariableInterpolationJobReturns(dmEJob, errors.New("fake-error"))

				_, err := reconciler.Reconcile(request)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to build the desired manifest eJob"))
			})

			It("handles an error when creating desired manifest eJob", func() {
				jobFactory.VariableInterpolationJobReturns(dmEJob, nil)

				client.CreateCalls(func(context context.Context, object runtime.Object, _ ...crc.CreateOption) error {
					switch object := object.(type) {
					case *ejv1.ExtendedJob:
						eJob := object
						if strings.HasPrefix(eJob.Name, "dm-") {
							return errors.New("fake-error")
						}
					}
					return nil
				})

				_, err := reconciler.Reconcile(request)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to create desired manifest ExtendedJob for BOSHDeployment 'default/foo': creating or updating ExtendedJob 'dm-foo': fake-error"))
			})

			It("handles an error when building instance group manifest eJob", func() {
				jobFactory.VariableInterpolationJobReturns(dmEJob, nil)
				jobFactory.InstanceGroupManifestJobReturns(dmEJob, errors.New("fake-error"))

				_, err := reconciler.Reconcile(request)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to build instance group manifest eJob"))
			})

			It("handles an error when creating instance group manifest eJob", func() {
				jobFactory.VariableInterpolationJobReturns(dmEJob, nil)
				jobFactory.InstanceGroupManifestJobReturns(igEJob, nil)

				client.CreateCalls(func(context context.Context, object runtime.Object, _ ...crc.CreateOption) error {
					switch object := object.(type) {
					case *ejv1.ExtendedJob:
						eJob := object
						if strings.HasPrefix(eJob.Name, "ig-") {
							return errors.New("fake-error")
						}
					}
					return nil
				})

				_, err := reconciler.Reconcile(request)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to create instance group manifest ExtendedJob for BOSHDeployment 'default/foo': creating or updating ExtendedJob 'ig-foo': fake-error"))
			})

			It("handles an error when building BPM configs eJob", func() {
				jobFactory.VariableInterpolationJobReturns(dmEJob, nil)
				jobFactory.InstanceGroupManifestJobReturns(dmEJob, nil)
				jobFactory.BPMConfigsJobReturns(dmEJob, errors.New("fake-error"))

				_, err := reconciler.Reconcile(request)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to build BPM configs eJob"))
			})

			It("handles an error when creating BPM configs eJob", func() {
				jobFactory.VariableInterpolationJobReturns(dmEJob, nil)
				jobFactory.InstanceGroupManifestJobReturns(igEJob, nil)
				jobFactory.BPMConfigsJobReturns(bpmEJob, nil)

				client.CreateCalls(func(context context.Context, object runtime.Object, _ ...crc.CreateOption) error {
					switch object := object.(type) {
					case *ejv1.ExtendedJob:
						eJob := object
						if strings.HasPrefix(eJob.Name, "bpm-") {
							return errors.New("fake-error")
						}
					}
					return nil
				})

				_, err := reconciler.Reconcile(request)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to create BPM configs ExtendedJob for BOSHDeployment 'default/foo': creating or updating ExtendedJob 'bpm-foo': fake-error"))
			})
		})
	})
})
