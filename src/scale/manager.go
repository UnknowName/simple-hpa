package scale

import (
	"log"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/net/context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

var client *k8SClient

type Scaler interface {
	GetServicePod(namespace, service string) (*int32, error)
	ChangeServicePod(namespace, service string, newCount *int32) error
}

func getClient() (*kubernetes.Clientset, error) {
	client, err := getClientOutCluster()
	if err == nil {
		log.Println("there is a kube file,guess outside the cluster")
		return client, nil
	}
	log.Println("guess inside the cluster")
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

func newK8SClient() *k8SClient {
	if client != nil {
		return client
	}
	clientset, err := getClient()
	if err != nil {
		log.Fatalln("init client failed")
	}
	return &k8SClient{clientset: clientset}
}

type k8SClient struct {
	clientset *kubernetes.Clientset
}

func (kc *k8SClient) GetServicePod(namespace, service string) (*int32, error) {
	dep, err := kc.clientset.AppsV1().Deployments(namespace).Get(context.TODO(), service, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return dep.Spec.Replicas, nil
}

func (kc *k8SClient) ChangeServicePod(namespace, service string, newCount *int32) error {
	dep, err := kc.clientset.AppsV1().Deployments(namespace).Get(context.TODO(), service, metav1.GetOptions{})
	if err != nil {
		return err
	}
	dep.Spec.Replicas = newCount
	_, err = kc.clientset.AppsV1().Deployments(namespace).Update(context.TODO(), dep, metav1.UpdateOptions{})
	return err
}

func newOks(c int) *oks {
	return &oks{data: make([]bool, c, c), i: 0}
}

type oks struct {
	data []bool
	i    int
}

func (o *oks) insert(r bool) {
	o.data[o.i] = r
	o.i = (o.i + 1) % len(o.data)
}

func (o *oks) allFalse() bool {
	for _, v := range o.data {
		if v == true {
			return false
		}
	}
	return true
}

func (o *oks) allTrue() bool {
	for _, v := range o.data {
		if v == false {
			return false
		}
	}
	return true
}

func NewScaler(cnt, internal int) *ScalerManage {
	r := &ScalerManage{
		cnt:       cnt,
		interval:  time.Second * time.Duration(internal),
		histories: make(map[string]time.Time),
		client:    newK8SClient(),
		safes:     make(map[string]*oks),
		wastes:    make(map[string]*oks),
	}
	return r
}

type ScalerManage struct {
	cnt       int
	interval  time.Duration
	histories map[string]time.Time // 历史操作记录
	safes     map[string]*oks
	wastes    map[string]*oks
	client    *k8SClient
}

func (sm *ScalerManage) Update(k string, isSafe, isWaste bool) {
	if val, ok := sm.safes[k]; !ok {
		sm.safes[k] = newOks(sm.cnt)
		sm.safes[k].insert(isSafe)
	} else {
		val.insert(isSafe)
	}
	if val, ok := sm.wastes[k]; !ok {
		sm.wastes[k] = newOks(sm.cnt)
		sm.wastes[k].insert(isWaste)
	} else {
		val.insert(isWaste)
	}
}

func (sm *ScalerManage) NeedChange(serviceName string) bool {
	latest, ok := sm.histories[serviceName]
	if !ok {
		sm.histories[serviceName] = time.Now().Add(sm.interval)
		return false
	}
	if latest.After(time.Now()) {
		return false
	}
	return sm.isWaste(serviceName) || sm.isDanger(serviceName)
}

func (sm *ScalerManage) isDanger(serviceName string) bool {
	// qps < conf.MaxQPS 全为假
	return sm.safes[serviceName].allFalse()
}

func (sm *ScalerManage) isWaste(serviceName string) bool {
	// qps < conf.SafeQPS全为真
	return sm.wastes[serviceName].allTrue()
}

func (sm *ScalerManage) ChangeServicePod(serviceName string, newCnt *int32) *int32 {
	namespaces := strings.Split(serviceName, ".")
	if len(namespaces) != 2 {
		log.Fatalln(serviceName, "no valid serviceName, use format like svc.namespace")
	}
	namespace, service := namespaces[1], namespaces[0]
	oldCnt, err := sm.client.GetServicePod(namespace, service)
	if err != nil {
		log.Println("get ", serviceName, "pod error", err)
		return nil
	}
	if *oldCnt == *newCnt {
		return nil
	}
	log.Printf("change %s from %d to %d", serviceName, *oldCnt, *newCnt)
	err = sm.client.ChangeServicePod(namespace, service, newCnt)
	sm.histories[serviceName] = time.Now().Add(sm.interval)
	if err != nil {
		log.Println("change service pod error", err)
	}
	return oldCnt
}
