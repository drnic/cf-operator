package extendedjob

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// ConvertOutputToSecret converts the output files of each container
// in the pod related to an ejob into a kubernetes secret.
func ConvertOutputToSecret(namespace string, namePrefix string) error {

	// hostname of the container is the pod name in kubernetes
	podName, err := os.Hostname()
	if err != nil {
		return errors.Wrapf(err, "Failed to fetch pod name.")
	}
	if podName == "" {
		return errors.Wrapf(err, "Pod name is empty.")
	}

	pod := &corev1.Pod{}

	// Authenticate with the cluster
	clientSet, err := authenticateInCluster()
	if err != nil {
		return err
	}

	// Fetch the pod
	pod, err = clientSet.CoreV1().Pods(namespace).Get(podName, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed to fetch pod %s", podName)
	}

	//clientSet.RESTClient().Get().Resource("extendedjob").Name("")

	fileNotifyChannel := make(chan string)
	errorChannel := make(chan error)

	// Loop over containers and create secrets for each output file for container
	for _, container := range pod.Spec.Containers {

		if container.Name == "output-persist" {
			continue
		}

		filePath := fmt.Sprintf("%s%s%s", "/mnt/quarks", container.Name, "/output.json")

		// Go routine to wait for the file to be created
		go waitForFile(container.Name, filePath, fileNotifyChannel, errorChannel)

		// wait for all the go routines
		for i := 0; i < len(pod.Spec.Containers)-1; i++ {
			select {
			case containerName := <-fileNotifyChannel:
				err := createOutputSecret(containerName, namePrefix, namespace, podName, clientSet)
				if err != nil {
					fmt.Println("Failure creating secret", err)
				}
			case failure := <-errorChannel:
				fmt.Println("Failure in some container", failure)
			}
		}
	}
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

// authenticateInCluster authenticates with the in cluster and returns the client
func authenticateInCluster() (*kubernetes.Clientset, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to authenticate with incluster config")
	}

	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to fetch clientset with incluster config")
	}
	return clientSet, nil
}

func createOutputSecret(containerName string, namePrefix string, namespace string, podName string, clientSet *kubernetes.Clientset) error {

	// Create secret
	secretName := namePrefix + containerName
	filePath := fmt.Sprintf("%s%s%s", "/mnt/quarks", containerName, "/output.json")

	// Fetch json from file
	file, _ := ioutil.ReadFile(filePath)
	var data map[string]string
	err := json.Unmarshal([]byte(file), &data)
	if err != nil {
		return errors.Wrapf(err, "failed to convert output file %s into json for creating secret %s in pod %s",
			filePath, secretName, podName)
	}

	// Create secret for the outputfile to persist
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
		},
	}
	secret.StringData = data

	//secret.Labels = secretLabels
	secret, err = clientSet.CoreV1().Secrets(namespace).Create(secret)
	if err != nil {
		return errors.Wrapf(err, "Failed to create secret %s for container %s in pod %s.", secretName, containerName, podName)
	}
	return nil
}
