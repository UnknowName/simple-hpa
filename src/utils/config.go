package utils

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v2"
)

const (
	maxAvgTime          = 60
	defaultAvgTime      = 5
	defaultIntervalTime = 120
	defaultIngressType  = "nginx"
	defaultMinPod       = 1
)

type autoScaleConfig struct {
	MaxPod   int32    `yaml:"maxPod"`
	MinPod   int32    `yaml:"minPod"`
	MaxQPS   float64  `yaml:"maxQPS"`
	SafeQPS  float64  `yaml:"safeQPS"`
	Services []string `yaml:"services"`
}

type listenConfig struct {
	ListenAddr string `yaml:"address"`
	Port       int    `yaml:"port"`
}

type ForwardConfig struct {
	TypeName string `yaml:"type"`
	Address  string `yaml:"address"`
}

type Config struct {
	AvgTime           int             `yaml:"avgTime"`
	ScaleIntervalTime int             `yaml:"scaleIntervalTime"`
	IngressType       string          `yaml:"ingressType"`
	Listen            listenConfig    `yaml:"listen"`
	AutoScale         autoScaleConfig `yaml:"autoScale"`
	Forwards          []ForwardConfig `yaml:"forwards"`
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
	if c.AutoScale.MinPod < defaultMinPod {
		c.AutoScale.MinPod = defaultMinPod
		log.Println("WARN, config minPod < 1, use ", defaultMinPod)
	}
	if c.AvgTime <= 0 {
		c.AvgTime = defaultAvgTime
		log.Println("INFO config avgTime use default ", defaultAvgTime)
	}
	if c.ScaleIntervalTime <= 0 {
		c.ScaleIntervalTime = defaultIntervalTime
		log.Println("INFO, config scaleIntervalTime use default ", defaultIntervalTime)
	}
	if c.IngressType == "" {
		c.IngressType = defaultIngressType
		log.Println("INFO config ingressType use default ", defaultIngressType)
	}
	if c.AutoScale.Services == nil {
		c.AutoScale.Services = make([]string, 0)
		log.Println("WARN config autoServices not  present,this mean nothing todo")
	}
	if c.Forwards == nil {
		c.Forwards = make([]ForwardConfig, 0)
	}
}

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
	sliceTime, err := strconv.Atoi(os.Getenv("SCALE_INTERVAL_TIME"))
	if err == nil {
		c.ScaleIntervalTime = sliceTime
	}
	avgTime, err := strconv.Atoi(os.Getenv("AVG_TIME"))
	if err == nil && avgTime > 0 && avgTime <= maxAvgTime {
		c.AvgTime = avgTime
	} else if avgTime != 0 && avgTime > maxAvgTime {
		// if env AVG_TIME not set, default is 0
		c.AvgTime = maxAvgTime
		log.Printf("WARN AVG_TIME env is %d great than %d, reset use %d", avgTime, maxAvgTime, maxAvgTime)
	} else {
		log.Printf("WARN AVG_TIME env is %d it's not valid, use config.yaml value %d", avgTime, c.AvgTime)
	}
	ingressType := os.Getenv("INGRESS_TYPE")
	if ingressType != "" {
		c.IngressType = ingressType
	}
	_forwards := os.Getenv("FORWARDS")
	if len(_forwards) <= 0 {
		return
	}
	forwardConfigs := make([]ForwardConfig, 0)
	for _, _forward := range strings.Split(_forwards, ",") {
		typeName, addr := strings.Split(_forward, "=")[0], strings.Split(_forward, "=")[1]
		forwardConfigs = append(forwardConfigs, ForwardConfig{typeName, addr})
	}
	if len(forwardConfigs) > 0 {
		c.Forwards = forwardConfigs
	}
}

func NewConfig(filename string) *Config {
	config := new(Config)
	file, err := os.Open(filename)
	if err != nil {
		log.Fatalln("Open file error ", err)
		return nil
	}
	defer file.Close()
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatalln("Read yaml file error ", err)
		return nil
	}
	if err := yaml.Unmarshal(bytes, config); err != nil {
		log.Fatalln("Convert config file error ", err)
		return nil
	}
	config.getEnvConfig()
	config.valid()
	return config
}
