# Development

- [Development](#development)
  - [Requirements](#requirements)
  - [Dependencies](#dependencies)
  - [Creating a new Resource and Controller](#creating-a-new-resource-and-controller)
    - [Reconcile Results](#reconcile-results)
    - [Testing](#testing)
  - [Create-Or-Update pattern](#create-or-update-pattern)
  - [Logging and Events](#logging-and-events)
  - [Versioning](#versioning)
  - [Releasing](#releasing)
    - [How to Create a New Release](#how-to-create-a-new-release)

## Requirements

- A working Kubernetes cluster
- Helm binary
- Go 1.12.2 and install the tool chain: `make tools`

## Dependencies

Run with libraries fetched via go modules:

```bash
export GO111MODULE=on
```

## Creating a new Resource and Controller

- create a new directory: `./pkg/kube/apis/<group_name>/<version>`
- in that directory, create the following files:
  - `types.go`
  - `register.go`
  - `doc.go`

  > You can safely use the implementation from another controller as inspiration.
  > You can also copy the files and modify them.

  The `types.go` file contains the definition of your resource. This is the file you care about. Make sure to run `make generate` _every time you make a change_. You can also check to see what changes would be done by running `make verify-gen-kube`.

  The `register.go` file contains some code that registers your new types.
  This file looks almost the same for all API resources.

  The `doc.go` (deep object copy) is required to make the `deepcopy` generator work.
  It's safe to copy this file from another controller.

- in `bin/gen-kube`, add your resource to the `GROUP_VERSIONS` variable (separated by a space `" "`):

  ```bash
  # ...
  GROUP_VERSIONS="boshdeployment:v1alpha1 <controller_name>:<version>"
  # ...
  ```

- regenerate code

  ```bash
  # int the root of the project
  make generate
  ```

- create a directory structure like this for your actual controller code:

  ```
  .
  +-- pkg
     +-- kube
         +-- controllers
             +-- <controller_name>
             ¦   +-- controller.go
             ¦   +-- reconciler.go
             +-- controller.go
  ```

  - `controller.go` is your controller implementation; this is where you should implement an `Add` function where register the controller with the `Manager`, and you watch for changes for resources that you care about.
  - `reconciler.go` contains the code that takes action and reconciles actual state with desired state.

  Simple implementation to get you started below.
  As always, use the other implementations to get you started.

  **Controller:**

  ```go
  package mycontroller

  import (
    "go.uber.org/zap"

    "sigs.k8s.io/controller-runtime/pkg/controller"
    "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
    "sigs.k8s.io/controller-runtime/pkg/handler"
    "sigs.k8s.io/controller-runtime/pkg/manager"
    "sigs.k8s.io/controller-runtime/pkg/source"

    mrcv1 "code.cloudfoundry.org/cf-operator/pkg/kube/apis/myresourcecontroller/v1"
  )

  func Add(log *zap.SugaredLogger, mgr manager.Manager) error {
    r := NewReconciler(log, mgr, controllerutil.SetControllerReference)

    // Create a new controller
    c, err := controller.New("myresource-controller", mgr, controller.Options{Reconciler: r})
    if err != nil {
      return err
    }

    // Watch for changes to primary resource
    err = c.Watch(&source.Kind{Type: &mrcv1.MyResource{}}, &handler.EnqueueRequestForObject{})
    if err != nil {
      return err
    }

    return nil
  }
  ```

  **Reconciler:**
  ```go
  package myresource

  import (
    "go.uber.org/zap"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/runtime"
    "sigs.k8s.io/controller-runtime/pkg/client"
    "sigs.k8s.io/controller-runtime/pkg/manager"
    "sigs.k8s.io/controller-runtime/pkg/reconcile"
  )

  type setReferenceFunc func(owner, object metav1.Object, scheme *runtime.Scheme) error

  func NewReconciler(log *zap.SugaredLogger, mgr manager.Manager, srf setReferenceFunc) reconcile.Reconciler {
    return &ReconcileMyResource{
      log:          log,
      client:       mgr.GetClient(),
      scheme:       mgr.GetScheme(),
      setReference: srf,
    }
  }

  type ReconcileMyResource struct {
    client       client.Client
    scheme       *runtime.Scheme
    setReference setReferenceFunc
    log          *zap.SugaredLogger
  }

  func (r *ReconcileMyResource) Reconcile(request reconcile.Request) (reconcile.Result, error) {
    r.log.Infof("Reconciling MyResource %s\n", request.NamespacedName)
    return reconcile.Result{}, nil
  }
  ```

- add the new resource to `addToSchemes` in `pkg/controllers/controller.go`.
- add the new controller to `addToManagerFuncs` in the same file.
- create a custom resource definition in `deploy/helm/cf-operator/templates/`
- add the custom resource definition to `bin/apply-crds`

### Reconcile Results

	// RequeueOnError will requeue if reconcile also returns an error
	RequeueOnError = reconcile.Result{}
	// Requeue will requeue the request, behaviour is different than returning an error
	Requeue = reconcile.Result{Requeue: true}
	// RequeueAfterDefault requeues after the default, unless reconcile also returns an error
	RequeueAfterDefault = reconcile.Result{RequeueAfter: config.RequeueAfter}
	// NoRequeue does not requeue, unless reconcile also returns an error
	NoRequeue = reconcile.Result{Requeue: false}

### Testing

- create functions in `env/machine`
- create functions in `env/catalog`

## Create-Or-Update pattern

A pattern that comes up quite often is that an object needs to be updated if it already exists or created if it doesn't. `controller-runtime` provides the `controller-util` package which has a `CreateOrUpdate` function that can help with that. The object's desired state must be reconciled with the existing state inside the passed in callback MutateFn - `type MutateFn func() error`. The MutateFn is called regardless of creating or updating an object.

```go
_, err = controllerutil.CreateOrUpdate(ctx, r.client, someSecret, secretMutateFn(someSecret, someSecret.StringData, someSecret.Labels, someSecret.Annotations))

func secretMutateFn(s *corev1.Secret, secretData map[string]string, labels map[string]string, annotations map[string]string) controllerutil.MutateFn {
	return func() error {
		s.Labels = labels
		s.Annotations = annotations
		s.StringData = secretData
		return nil
	}
}
```

- Care must be taken when persisting objects that are already in their final state because they will be overwritten with the existing state if there already is such an object in the system.
- `CreateOrUpdate`s should not use blindly `DeepCopyInto` or `DeepCopy` all the time, but make more precise changes.

## Logging and Events

We start with a single context and pass that down via controllers into
reconcilers. Reconcilers will create a context with timeout from the inherited
context and linting will check if the `cancel()` function of that context is
being handled.

The `ctxlog` module provides a context with a named zap logger and an event recorder to the reconcilers.
This is how it's set up for reconcilers:

```
// after logger is available
ctx := ctxlog.NewParentContext(log)
// adding named log and event recorder in controllers
ctx = ctxlog.NewContextWithRecorder(ctx, "example-reconciler", mgr.GetEventRecorderFor("example-recorder"))
// adding timeout in reconcilers
ctx, cancel := context.WithTimeout(ctx, timeout)
defer cancel()
```

The `ctxlog` package provides several logging functions. `Infof`, `Errorf`, `Error` and such wrap the corresponding zap log methods.

The logging functions are also implemented on struct, to add event generation to the logging:

```
ctxlog.WithEvent(instance, "Reason").Infof("message: %s", v)
err := ctxlog.WithEvent(instance, "Reason").Errorf("message: %s", v)
err := ctxlog.WithEvent(instance, "Reason").Error("part", "part", "part")
```
The reason should be camel-case, so switch statements could match it.

Error funcs like `WithEvent().Errorf()` also return an error, with the same message as the log message and event that were generated.

Calling `WarningEvent` just creates a warning event, without logging.

## Versioning

APIs and types follow the upstream versioning scheme described at: https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning
