package utils

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
)

type autoScaleConfig struct {
	Max         float64  `yaml:"max"`
	Safe        float64  `yaml:"safe"`
	SliceSecond int      `yaml:"sliceSecond"`
	Services    []string `yaml:"services"`
}

type listenConfig struct {
	ListenAddr string `yaml:"address"`
	Port       int    `yaml:"port"`
	NetType    string `yaml:"type"`
}

type Config struct {
	Listen    listenConfig    `yaml:"listen"`
	AutoScale autoScaleConfig `yaml:"autoScale"`
}

func (c *Config) String() string {
	return fmt.Sprintf("Config{Listen=%s:%d}", c.Listen.ListenAddr, c.Listen.Port)
}

func NewConfig(filename string) *Config {
	config := new(Config)
	file, err := os.Open(filename)
	if err != nil {
		log.Fatalln("Open file error ", err)
	}
	defer file.Close()
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatalln("Read yaml file error ", err)
	}
	if err := yaml.Unmarshal(bytes, config); err != nil {
		log.Fatalln("Convert config file error ", err)
	}
	return config
}
