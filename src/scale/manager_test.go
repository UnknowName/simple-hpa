package scale

import (
	"fmt"
	"log"
	"testing"
)

func TestK8SClient_ChangeServicePod(t *testing.T) {
	client := NewK8SClient()
	newCount := int32(1)
	err := client.ChangeServicePod("demo-dev", "daohao", &newCount)
	log.Println(err)
}

func TestScalerManage_Update(t *testing.T) {
	service := "daohao demo"
	var namespace, svc string
	_, err := fmt.Sscanf(service, "%s %s", &namespace, &svc)
	log.Println(err)
	log.Println("names", namespace)
	log.Println("svc", svc)
}