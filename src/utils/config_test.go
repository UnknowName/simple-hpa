package utils

import (
	"log"
	"testing"
)

func TestNewConfig(t *testing.T) {
	config := NewConfig("F:\\GoCodes\\simple-hpa\\config.yaml")
	log.Println(config, config.AutoScale.MaxQPS > 19.8)
	log.Println(config.AutoScale.Services)
	for _, svc := range config.AutoScale.Services {
		log.Println(svc)
	}
	log.Printf(config.IngressType)
}
