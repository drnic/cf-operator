package extendedjob

import (
	"fmt"
	"os"
	"time"
	"io/ioutil"
	"encoding/json"


	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// ConvertOutputToSecret converts the output files of each container
// in the pod into a kubernetes secret.
func ConvertOutputToSecret(namespace string, namePrefix string) error {

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

	// Loop over containers and create secrets
	for _, container := range pod.Spec.Containers {

		if container.Name == "output-persist" {
			continue
		}

		// Go routine to wait for the file to be created
		go waitForFile(container.Name, "/mnt/quarks/"+container.Name+"/output.json", fileNotifyChannel, errorChannel)

		// wait for all the go routines
		for i := 0; i < len(pod.Spec.Containers)-1; i++ {
			select {
				case notified := <-fileNotifyChannel:
					fmt.Println("Created file in container ", notified)

					// Create secret
					secretName := namePrefix + container.Name

					// Get json from file
					file, _ := ioutil.ReadFile("/mnt/quarks/"+container.Name+"/output.json")

					var data map[string]string
					_ = json.Unmarshal([]byte(file), &data)

					secret := &corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      secretName,
							Namespace: namespace,
						},
					}

					secret.StringData = data
					//secret.Labels = secretLabels
					secret, err := clientset.CoreV1().Secrets(namespace).Create(secret)
					if err != nil {
						fmt.Println("Failed to create secret", secret.Name)
					}				
				case failure := <-errorChannel:
					fmt.Println("Failure in some container", failure)
			}
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

	for {
		time.Sleep(1 * time.Second)
		_, err := os.Stat(fileName)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			} else {
				errorChannel <- err
				break
			}
		}
		fileNotifyChannel <- containerName
		break
	}
}
