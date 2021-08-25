package utils

import (
	"log"
	"testing"
)

func TestNewConfig(t *testing.T) {
	config := NewConfig("F:\\GoCodes\\simple-hpa\\config.yml")
	log.Println(config, config.AutoScale.Max > 19.8)
	log.Println(config.AutoScale.Services)
	for _, svc := range config.AutoScale.Services {
		log.Println(svc)
	}
}
