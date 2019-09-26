package extendedjob

import (
	"fmt"
	"os"
	"strings"
)

// ConvertOutputToSecret converts the output files of each container
// in the pod into a kubernetes secret.
func ConvertOutputToSecret() error {

	// Check for the file in /mnt using polling of inotigfu in go lang

	// hostname of the container is the pod name in kubernetes
	podName, err := os.Hostname()
	if err != nil {
		panic(err)
	}

	fmt.Println("Pod Name", podName)

	for _, e := range os.Environ() {
		pair := strings.Split(e, "=")
		fmt.Println(pair[0])
	}

	/*// Authenticate with the incluster kube config
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	for {
		// get pods in all the namespaces by omitting namespace
		// Or specify namespace to get pods in particular namespace
		pods, err := clientset.CoreV1().Pods("").List(metav1.ListOptions{})
		if err != nil {
			panic(err.Error())
		}
		fmt.Printf("There are %d pods in the cluster\n", len(pods.Items))

		// Examples for error handling:
		// - Use helper functions e.g. errors.IsNotFound()
		// - And/or cast to StatusError and use its properties like e.g. ErrStatus.Message
		_, err = clientset.CoreV1().Pods("default").Get("example-xxxxx", metav1.GetOptions{})
		if errors.IsNotFound(err) {
			fmt.Printf("Pod example-xxxxx not found in default namespace\n")
		} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
			fmt.Printf("Error getting pod %v\n", statusError.ErrStatus.Message)
		} else if err != nil {
			panic(err.Error())
		} else {
			fmt.Printf("Found example-xxxxx pod in default namespace\n")
		}

		time.Sleep(10 * time.Second)
	}

	// Fetch the extended job
	instance := ejv1.ExtendedJob{}

	err := client.ExtendedJobs(namespace)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			ctxlog.Info(ctx, "Skip reconcile: Job not found")
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		ctxlog.Info(ctx, "Error reading the object")
		return reconcile.Result{}, err
	}


	if !reflect.DeepEqual(ejv1.Output{}, ej.Spec.Output) && ej.Spec.Output != nil {

	*/

	return nil
}
