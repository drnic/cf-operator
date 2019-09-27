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

	// Authenticate with the cluster
	err, clientSet := authenticateInCluster()
	if err != nil {
		return err
	}

	// Fetch the pod
	pod, err := clientset.CoreV1().Pods(namespace).Get(podName, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed to fetch pod %s", podName)
	}


	fileNotifyChannel := make(chan string)
	errorChannel := make(chan error)

	// Loop over containers and create secrets for each output file for container
	for _, container := range pod.Spec.Containers {

		if container.Name == "output-persist" {
			continue
		}

		filePath := fmt.Sprintf("%s%s", "/mnt/quarks",container.Name,"/output.json")

		// Go routine to wait for the file to be created
		go waitForFile(container.Name, filePath, fileNotifyChannel, errorChannel)

		// wait for all the go routines
		for i := 0; i < len(pod.Spec.Containers)-1; i++ {
			select {
				case notified := <-fileNotifyChannel:
					createOutputSecret(namePrefix, filePath, podName)
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
func authenticateInCluster() (clientSet, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil,errors.Wrapf(err, "failed to authenticate with incluster config")
	}

	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil,errors.Wrapf(err, "failed to fetch clientset with incluster config")
	}
	return clientSet, nil
}

func createOutputSecret(namePrefix string, filePath string, podName string, containerName string) error {

	// Create secret
	secretName := namePrefix + container.Name

	// Fetch json from file
	file, _ := ioutil.ReadFile(filePath)
	var data map[string]string
	err = json.Unmarshal([]byte(file), &data)
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
	secret.StringData = data

	//secret.Labels = secretLabels
	secret, err := clientset.CoreV1().Secrets(namespace).Create(secret)
	if err != nil {
		return errors.Wrapf(err, "Failed to create secret %s for container %s in pod %s.", secretName, containerName, podName)
		if 
	}
}