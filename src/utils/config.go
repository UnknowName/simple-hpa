package utils

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
)

type autoScaleConfig struct {
	MaxPod      int32    `yaml:"maxPod"`
	MinPod      int32    `yaml:"minPod"`
	MaxQPS      float64  `yaml:"maxQPS"`
	SafeQPS     float64  `yaml:"safeQPS"`
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

func (c *Config) valid() {
	if c.AutoScale.MaxQPS < c.AutoScale.SafeQPS {
		panic("config error, autoScale maxQPS < autoScale.safeQPS")
	}
	if c.AutoScale.MaxPod < c.AutoScale.MinPod {
		panic("config error, autoScale maxPod < autoScale.minPod")
	}
	if c.AutoScale.MinPod < 1 {
		log.Println("warning, minPod < 1, use 1")
		c.AutoScale.MinPod = 1
	}
}

// env优先级大于config.yml，这样在容器环境中，只需要修改env而不要重新打包镜像

func (c *Config) getEnvConfig() {
	service := os.Getenv("SCALE_SERVICES")
	if service != "" {
		c.AutoScale.Services = strings.Split(service, ",")
	}
	safeQps, err := strconv.Atoi(os.Getenv("SAFE_QPS"))
	if err == nil {
		c.AutoScale.SafeQPS = float64(safeQps)
	}
	maxQps, err := strconv.Atoi(os.Getenv("MAX_QPS"))
	if err == nil {
		c.AutoScale.MaxQPS = float64(maxQps)
	}
	minPod, err := strconv.Atoi(os.Getenv("MIN_POD"))
	if err == nil {
		c.AutoScale.MinPod = int32(minPod)
	}
	maxPod, err := strconv.Atoi(os.Getenv("MAX_POD"))
	if err == nil {
		c.AutoScale.MaxPod = int32(maxPod)
	}
	sliceTime, err := strconv.Atoi(os.Getenv("SLICE_TIME"))
	if err == nil {
		c.AutoScale.SliceSecond = sliceTime
	}
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
	config.getEnvConfig()
	config.valid()
	return config
}
