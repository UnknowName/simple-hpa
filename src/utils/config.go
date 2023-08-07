package utils

import (
	"fmt"
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
	MaxQps            float32 `yaml:"maxQps"`
	SafeQps           float32 `yaml:"safeQps"`
	Factor            float32 `yaml:"factor"`
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
		float32(_maxQps),
		float32(_safeQps),
		float32(_factor),
	}
}

type scaleServiceConfig struct {
	Namespace   string  `yaml:"namespace"`
	ServiceName string  `yaml:"serviceName"`
	MaxPod      int32   `yaml:"maxPod"`
	MinPod      int32   `yaml:"minPod"`
	MaxQps      float32 `yaml:"maxQps"`
	SafeQps     float32 `yaml:"safeQps"`
	Factor      float32 `yaml:"factor"`
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
	Notifies      []notifyConfig        `yaml:"notifies"`
	ScaleServices []*scaleServiceConfig `yaml:"scaleServices"`
}

func (c *Config) String() string {
	return fmt.Sprintf("Config{Listen=%s:%d}", c.Listen.ListenAddr, c.Listen.Port)
}

func (c *Config) GetServiceConfig(service string) *scaleServiceConfig {
	for _, _conf := range c.ScaleServices {
		svcName := fmt.Sprintf("%s.%s", _conf.ServiceName, _conf.Namespace)
		if svcName == service {
			return _conf
		}
	}
	return nil
}

func (c *Config) valid() {
	if c.Default.MaxQps < c.Default.SafeQps {
		log.Fatalln("config error, default.maxQPS < default.safeQPS")
	}
	if c.Default.MaxPod < c.Default.MinPod {
		log.Fatalln("config error, default.maxPod < default.minPod")
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
			log.Fatalln(fmt.Sprintf("%s config err, MaxPod < MinPod", scaleConfig.ServiceName))
		}
		if scaleConfig.MaxQps <= 0 {
			scaleConfig.MaxQps = c.Default.MaxQps
		}
		if scaleConfig.SafeQps <= 0 {
			scaleConfig.SafeQps = c.Default.SafeQps
		}
		if scaleConfig.MaxQps < scaleConfig.SafeQps {
			log.Fatalln(fmt.Sprintf("%s config err, MaxQps < SafeQps", scaleConfig.ServiceName))
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
	forwardConfigs := make([]ForwardConfig, 0)
	for _, _forward := range strings.Split(_forwards, ",") {
		_forward = strings.TrimSpace(_forward)
		if _forward == "" {
			continue
		}
		values := strings.Split(_forward, "=")
		if len(values) != 2 {
			log.Println("WARN ", _forward, "env forward no valid")
			continue
		}
		typeName, addr := values[0], values[1]
		forwardConfigs = append(forwardConfigs, ForwardConfig{typeName, addr})
	}
	if len(forwardConfigs) > 0 {
		c.Forwards = forwardConfigs
	}
	notify := os.Getenv("NOTIFIES")
	notifies := strings.Split(notify, ",")
	notifyConfigs := make([]notifyConfig, 0)
	for _, _notify := range notifies {
		_notify = strings.TrimSpace(_notify)
		if _notify == "" {
			continue
		}
		values := strings.Split(_notify, ":")
		if len(values) != 3 {
			log.Println("WARN env notify", _notify, "no valid")
			continue
		}
		typeName, token, keyword := values[0], values[1], values[2]
		notifyConfig := notifyConfig{typeName, token, keyword}
		notifyConfigs = append(notifyConfigs, notifyConfig)
	}
	if len(notifyConfigs) > 0 {
		c.Notifies = notifyConfigs
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
	bytes, err := os.ReadFile(filename)
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

type notifyConfig struct {
	Type    string `json:"type"`
	Token   string `json:"token"`
	Keyword string `json:"keyword"`
}