package extendedjob

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// ConvertOutputToSecret converts the output files of each container
// in the pod into a kubernetes secret.
func ConvertOutputToSecret(namespace string) error {

	// Check for the file in /mnt using polling of inotigfu in go lang

	// hostname of the container is the pod name in kubernetes
	podName, err := os.Hostname()
	if err != nil {
		panic(err)
	}

	// Authenticate with the incluster kube config
	config, err := rest.InClusterConfig()
	if err != nil {
		return errors.Wrapf(err, "failed to authenticate with incluster config")
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return errors.Wrapf(err, "failed to fetch clientset with incluster config")
	}

	// Fetch pod
	pod, err := clientset.CoreV1().Pods(namespace).Get(podName, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed to fetch pod %s", podName)
	}

	fmt.Println(pod)

	/*// Loop over containers and create secrets
	for containerIndex, container := range pod.Spec.Containers {

		if err != nil {
			log.Fatal(err)
		}
		defer watcher.Close()

		done := make(chan bool)
		go func() {
			for {
				select {
				case event, ok := <-watcher.Events:
					if !ok {
						return
					}
					log.Println("event:", event)
					if event.Op&fsnotify.Write == fsnotify.Write {
						log.Println("modified file:", event.Name)
					}
				case err, ok := <-watcher.Errors:
					if !ok {
						return
					}
					log.Println("error:", err)
				}
			}
		}()

		err = watcher.Add("/tmp/")
		if err != nil {
			log.Fatal(err)
		}
		<-done
	}
	//
	// Create secrets*/
	return nil
}
