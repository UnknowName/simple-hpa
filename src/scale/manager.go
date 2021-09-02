package scale

import (
	"golang.org/x/net/context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"log"
	"path/filepath"
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

func NewK8SClient() *K8SClient {
	clientset,err := getClient()
	if err != nil {
		log.Fatalln("init client failed")
	}
	return &K8SClient{clientset: clientset}
}

type K8SClient struct {
	clientset *kubernetes.Clientset
}

func (kc *K8SClient) GetServicePod(namespace, service string) (*int32, error) {
	dep, err := kc.clientset.AppsV1().Deployments(namespace).Get(context.TODO(), service, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return dep.Spec.Replicas, nil
}

func (kc *K8SClient) ChangeServicePod(namespace, service string, newCount *int32) error {
	dep, err := kc.clientset.AppsV1().Deployments(namespace).Get(context.TODO(), service, metav1.GetOptions{})
	if err != nil {
		return err
	}
	dep.Spec.Replicas = newCount
	_, err = kc.clientset.AppsV1().Deployments(namespace).Update(context.TODO(), dep, metav1.UpdateOptions{})
	return err
}