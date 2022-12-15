package utils

import (
	"log"
	"os"
	"testing"
)

func TestNewConfig(t *testing.T) {
	os.Setenv("SCALE_SERVICES", "daohao.demo-dev:1:2:10:20:1")
	os.Setenv("NOTIFIES", "dding:token:keyword")
	config := NewConfig("F:\\GoCodes\\simple-hpa\\config.yaml")
	log.Println(config.ScaleServices, config.Default.MaxPod, config.Default.AvgTime)
	log.Println(config.Notifies)
}
