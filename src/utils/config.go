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
	defaultFact         = 1
)

type DefaultConfig struct {
	AvgTime           int     `yaml:"avgTime"`
	ScaleIntervalTime int     `yaml:"scaleIntervalTime"`
	MaxPod            int32   `yaml:"maxPod"`
	MinPod            int32   `yaml:"minPod"`
	MaxQps            float64 `yaml:"maxQps"`
	SafeQps           float64 `yaml:"safeQps"`
	Factor            int32   `yaml:"factor"`
}

func newScaleConfig(namespace, svc, minPod, maxPod, safeQps, maxQps, factor string) *scaleServiceConfig {
	_minPod, err := strconv.Atoi(minPod)
	if err != nil {
		log.Fatalln(err)
	}
	_maxPod, err := strconv.Atoi(maxPod)
	if err != nil {
		log.Fatalln(err)
	}
	_factor, err := strconv.Atoi(factor)
	if err != nil {
		log.Fatalln(err)
	}
	_safeQps, err := strconv.Atoi(safeQps)
	if err != nil {
		log.Fatalln(err)
	}
	_maxQps, err := strconv.Atoi(maxQps)
	if err != nil {
		log.Fatalln(err)
	}
	return &scaleServiceConfig{
		namespace,
		svc,
		int32(_maxPod),
		int32(_minPod),
		float64(_maxQps),
		float64(_safeQps),
		int32(_factor),
	}
}

type scaleServiceConfig struct {
	Namespace   string  `yaml:"namespace"`
	ServiceName string  `yaml:"serviceName"`
	MaxPod      int32   `yaml:"maxPod"`
	MinPod      int32   `yaml:"minPod"`
	MaxQps      float64 `yaml:"maxQps"`
	SafeQps     float64 `yaml:"safeQps"`
	Factor      int32   `yaml:"factor"`
}

func (ssc *scaleServiceConfig) String() string {
	return fmt.Sprintf("{%s.%s, %d, %.2f}", ssc.ServiceName, ssc.Namespace, ssc.MaxPod, ssc.MaxQps)
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
	IngressType   string                `yaml:"ingressType"`
	Default       *DefaultConfig        `yaml:"default"`
	Listen        listenConfig          `yaml:"listen"`
	Forwards      []ForwardConfig       `yaml:"forwards"`
	ScaleServices []*scaleServiceConfig `yaml:"scaleServices"`
}

func (c *Config) String() string {
	return fmt.Sprintf("Config{Listen=%s:%d}", c.Listen.ListenAddr, c.Listen.Port)
}

func (c *Config) valid() {
	if c.Default.MaxQps < c.Default.SafeQps {
		panic("config error, default.maxQPS < default.safeQPS")
	}
	if c.Default.MaxPod < c.Default.MinPod {
		panic("config error, default.maxPod < default.minPod")
	}
	if c.Default.MinPod < defaultMinPod {
		c.Default.MinPod = defaultMinPod
		log.Println("WARN, default.minPod < 1, use ", defaultMinPod)
	}
	if c.Default.AvgTime <= 0 {
		c.Default.AvgTime = defaultAvgTime
		log.Println("INFO default.avgTime use default ", defaultAvgTime)
	}
	if c.Default.ScaleIntervalTime <= 0 {
		c.Default.ScaleIntervalTime = defaultIntervalTime
		log.Println("INFO, default.scaleIntervalTime use default ", defaultIntervalTime)
	}
	if c.IngressType == "" {
		c.IngressType = defaultIngressType
		log.Println("INFO config ingressType use default ", defaultIngressType)
	}
	if c.Default.Factor <= 0 {
		c.Default.Factor = defaultFact
	}
	if c.ScaleServices == nil {
		c.ScaleServices = make([]*scaleServiceConfig, 0)
		log.Println("WARN config scaleServices not present,this mean nothing to do")
	}
	for _, scaleConfig := range c.ScaleServices {
		if scaleConfig.MinPod <= 0 {
			scaleConfig.MinPod = c.Default.MinPod
		}
		if scaleConfig.MaxPod <= 0 {
			scaleConfig.MaxPod = c.Default.MaxPod
		}
		if scaleConfig.MinPod <= 0 {
			scaleConfig.MinPod = c.Default.MinPod
		}
		if scaleConfig.MaxPod < scaleConfig.MinPod {
			panic(fmt.Sprintf("%s config err, MaxPod < MinPod", scaleConfig.ServiceName))
		}
		if scaleConfig.MaxQps <= 0 {
			scaleConfig.MaxQps = c.Default.MaxQps
		}
		if scaleConfig.SafeQps <= 0 {
			scaleConfig.SafeQps = c.Default.SafeQps
		}
		if scaleConfig.MaxQps < scaleConfig.SafeQps {
			panic(fmt.Sprintf("%s config err, MaxQps < SafeQps", scaleConfig.ServiceName))
		}
		if scaleConfig.Factor <= 0 {
			scaleConfig.Factor = c.Default.Factor
		}
	}
	if c.Forwards == nil {
		c.Forwards = make([]ForwardConfig, 0)
	}
}

func (c *Config) getEnvConfig() {
	envService := os.Getenv("SCALE_SERVICES")
	if envService != "" {
		scaleConfigs := make([]*scaleServiceConfig, 0)
		log.Println("Fond env SCALE_SERVICES:", envService)
		for _, _service := range strings.Split(envService, ",") {
			if _service == "" {
				continue
			}
			items := strings.Split(_service, ":")
			_serviceNamespace := strings.Split(items[0], ".")
			if len(items) != 6 || len(_serviceNamespace) != 2 {
				log.Printf("ERROR env SCALE_SERVICES item %s format error ", _serviceNamespace)
				log.Fatalln("use service.namespace:minPod:maxPod:safeQps:MaxQps:factor format")
			}
			serviceName, namespace := _serviceNamespace[0], _serviceNamespace[1]
			minPod, maxPod, safeQps, maxQps, factor := items[1], items[2], items[3], items[4], items[5]
			serviceConfig := newScaleConfig(namespace, serviceName, minPod, maxPod, safeQps, maxQps, factor)
			scaleConfigs = append(scaleConfigs, serviceConfig)
		}
		c.ScaleServices = scaleConfigs
	}
	sliceTime, err := strconv.Atoi(os.Getenv("SCALE_INTERVAL_TIME"))
	if err == nil {
		c.Default.ScaleIntervalTime = sliceTime
	}
	avgTime, err := strconv.Atoi(os.Getenv("AVG_TIME"))
	if err == nil && avgTime > 0 && avgTime <= maxAvgTime {
		c.Default.AvgTime = avgTime
	} else if avgTime != 0 && avgTime > maxAvgTime {
		// if env AVG_TIME not set, default is 0
		c.Default.AvgTime = maxAvgTime
		log.Printf("WARN AVG_TIME env is %d great than %d, reset use %d", avgTime, maxAvgTime, maxAvgTime)
	} else {
		log.Printf("WARN AVG_TIME env is %d it's not valid, use config.yaml value %d", avgTime, c.Default.AvgTime)
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
