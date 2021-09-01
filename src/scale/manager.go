package scale

import (
	"errors"
	"golang.org/x/net/context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"log"
	"path/filepath"
	"strings"
)

func getClient() (*kubernetes.Clientset, error) {
	client, err := getClientOutCluster()
	if err == nil {
		log.Println("fond the config file,guess out cluster")
		return client, nil
	}
	log.Println("guess in cluster")
	return getClientInCluster()
}

func getClientOutCluster() (*kubernetes.Clientset, error) {
	var kubeConfigFile string
	homePath := homedir.HomeDir()
	if homePath != "" {
		kubeConfigFile = filepath.Join(homePath, ".kube", "config")
	}
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfigFile)
	if err != nil {
		return nil, err
	}
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalln(err)
		return nil, err
	}
	return clientSet, nil
}

func getClientInCluster() (*kubernetes.Clientset, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}

func ChangeServicePod(service string, count *int32) error {
	names := strings.Split(service, ".")
	if len(names) != 2 {
		return errors.New("service name wrong,use full name like namespace.serviceName")
	}
	namespace, service := names[0], names[1]
	client, err := getClient()
	if err != nil {
		return err
	}
	deployment, err := client.AppsV1().Deployments(namespace).Get(context.TODO(), service, metav1.GetOptions{})
	if err != nil {
		return err
	}
	deployment.Spec.Replicas = count
	_, err = client.AppsV1().Deployments(namespace).Update(context.TODO(), deployment, metav1.UpdateOptions{})
	if err != nil {
		log.Println("change service pod error ", err)
	}
	return err
}