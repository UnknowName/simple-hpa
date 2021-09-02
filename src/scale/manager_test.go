package scale

import (
	"log"
	"testing"
)

func TestK8SClient_ChangeServicePod(t *testing.T) {
	client := NewK8SClient()
	newCount := int32(1)
	err := client.ChangeServicePod("demo-dev", "daohao", &newCount)
	log.Println(err)
}