package utils

import (
	"log"
	"os"
	"testing"
)

func TestNewConfig(t *testing.T) {
	os.Setenv("SCALE_SERVICES", "daohao.demo-dev:1:2:10:20:1")
	config := NewConfig("F:\\GoCodes\\simple-hpa\\config.yaml")
	// log.Println(config, config.AutoScale.MaxQPS > 19.8)
	log.Println(config.ScaleServices, config.Default.MaxPod, config.Default.AvgTime)
}
