package extendedjob

import (
	"fmt"
	"os"
	"time"

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

	fileNotifyChannel := make(chan string)
	errorChannel := make(chan error)
	//	var wg sync.WaitGroup

	//	wg.Add(len(pod.Spec.Containers) - 1)

	// Loop over containers and create secrets
	for _, container := range pod.Spec.Containers {

		// Go routine to wait for the file to be created
		fmt.Println("Came into the for loops of containers")
		go waitForFile(container.Name, "/mnt/quarks/"+container.Name+"/output.json", fileNotifyChannel, errorChannel)

		// For loop over select since I know the number of messages ahead
		select {
		case notified := <-fileNotifyChannel:
			fmt.Println("Created file in container ", notified)
		case failure := <-errorChannel:
			fmt.Println("Failure in some container", failure)
		}
	}

	/*if err != nil {
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

// waitForFile waits for the file to be created
func waitForFile(containerName string, fileName string, fileNotifyChannel chan<- string, errorChannel chan<- error) {
	fmt.Println("Started a go routine for a continaer", containerName)

	for {
		time.Sleep(1 * time.Second)
		fmt.Println("Checking for file at ", fileName)
		_, err := os.Stat(fileName)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Println("Not found ", fileName)
				continue
			} else {
				fmt.Println("Some shitty error", err)
				errorChannel <- err
				break
			}
		}
		fmt.Println("Found file sending some output to", fileName)
		fileNotifyChannel <- containerName
		break
	}
}
