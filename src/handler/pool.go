package handler

import (
	"fmt"
	"log"
	"math"
	"time"

	"auto-scale/src/ingress"
	"auto-scale/src/scale"
	"auto-scale/src/utils"
)

type IngressType uint8

const (
	// 一个随机常数，用于防止时间间隔重叠
	randTime                     = time.Millisecond * 211
	defaultQueueSize             = 1024
	defaultPoolSize              = 10
	// timeFmt                      = "2006-01-02 15:04:05"
	nginx            IngressType = iota
	traefik
)

type handler interface {
	ParseData(data []byte) ingress.Access
	SetScaleService(services []string)
}

func newDataHandler(ingressType IngressType) handler {
	switch ingressType {
	case nginx:
		return &nginxDataHandler{
			ingressType: ingressType,
			logKey:      []byte("nginx: "),
			autoService: make(map[string]struct{}),
		}
	default:
		log.Fatalln("un support ingress type")
	}
	return nil
}

func NewPoolHandler(config *utils.Config) *PoolHandler {
	var ingressType IngressType
	switch config.IngressType {
	case "nginx":
		ingressType = nginx
	case "traefik":
		ingressType = traefik
	default:
		log.Fatalln("Not support Ingress type")
	}
	workers := make([]handler, defaultPoolSize, defaultPoolSize)
	queues := make([]chan []byte, defaultPoolSize, defaultPoolSize)
	for i := 0; i < defaultPoolSize; i++ {
		workers[i] = newDataHandler(ingressType)
		queues[i] = make(chan []byte, defaultQueueSize)
	}
	senders := make([]utils.Sender, 0)
	for _, _conf := range config.Notifies {
		sender := utils.NewSender(_conf.Type, _conf.Token)
		if sender == nil {
			continue
		}
		senders = append(senders, sender)
	}
	poolHandler := &PoolHandler{
		config:     config,
		workers:    workers,
		senders:    senders,
		calculator: NewCalculator(config.Default.AvgTime),
		adjuster:   scale.NewScaler(minuteCount/config.Default.AvgTime, config.Default.ScaleIntervalTime),
		poolSize:   defaultPoolSize,
		queue:      queues,
	}
	poolHandler.startWorkers()
	return poolHandler
}

type PoolHandler struct {
	config     *utils.Config
	senders    []utils.Sender
	workers    []handler
	calculator *Calculator
	adjuster   *scale.ScalerManage
	poolSize   uint8
	queue      []chan []byte
	isStart    bool
}

func (ph *PoolHandler) Execute(data []byte) {
	index := time.Now().UnixMilli() % defaultPoolSize
	ph.queue[index] <- data
}

func (ph *PoolHandler) autoScale() {
	log.Println("start auto scale worker success")
	for {
		record := <-ph.calculator.Pipeline()
		if record == nil {
			continue
		}
		conf := ph.config.GetServiceConfig(record.ServiceName)
		if conf == nil {
			continue
		}
		qps := record.AvgQps() * conf.Factor / float32(ph.config.Default.AvgTime)
		log.Printf("latest %d seconds %s qps(*%.1f)=%.1f active upstreams=%d",
			ph.config.Default.AvgTime,
			record.ServiceName,
			conf.Factor,
			qps,
			record.TotalUpstreams,
		)
		ph.adjuster.Update(record.ServiceName, qps < conf.MaxQps, qps < conf.SafeQps)
		if ph.adjuster.NeedChange(record.ServiceName) {
			cnt := int32(math.Ceil(float64(qps / conf.MaxQps)))
			if cnt == 0 {
				continue
			}
			if cnt > conf.MaxPod {
				log.Printf("%s wants %d, but max is %d", record.ServiceName, cnt, conf.MaxPod)
				cnt = conf.MaxPod
			}
			if cnt < conf.MinPod {
				log.Printf("%s wants %d, but min is %d", record.ServiceName, cnt, conf.MinPod)
				cnt = conf.MinPod
			}
			go func() {
				for _, sender := range ph.senders {
					sender.Send(fmt.Sprintf("%s new count %d", record.ServiceName, cnt))
				}
			}()
			ph.adjuster.ChangeServicePod(record.ServiceName, &cnt)
		}
	}
}

func (ph *PoolHandler) startWorkers() {
	if ph.isStart {
		return
	}
	for i, worker := range ph.workers {
		services := make([]string, len(ph.config.ScaleServices))
		for i, config := range ph.config.ScaleServices {
			services[i] = fmt.Sprintf("%s.%s", config.ServiceName, config.Namespace)
		}
		worker.SetScaleService(services)
		go func(i int, worker handler) {
			for {
				byteData := <-ph.queue[i]
				accessItem := worker.ParseData(byteData)
				if accessItem == nil {
					continue
				}
				ph.calculator.Update(accessItem)
			}
		}(i, worker)
	}
	go ph.autoScale()
	ph.isStart = true
}
