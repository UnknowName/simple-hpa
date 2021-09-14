package handler

import (
	"golang.org/x/net/context"
	"log"
	"time"

	"simple-hpa/src/ingress"
	"simple-hpa/src/metrics"
	"simple-hpa/src/scale"
	"simple-hpa/src/utils"
)

type IngressType uint8

type FilterFunc func(itemChan ingress.Access, services []string, parent context.Context) ingress.Access

const (
	defaultQueueSize             = 1024
	defaultPoolSize              = 10
	nginx            IngressType = iota
	traefik
)

type handler interface {
	parseData([]byte, []string, FilterFunc, context.Context) <-chan ingress.Access
	parseDataWithFilter([]byte, []string) ingress.Access
}

func newDataHandler(ingressType IngressType) handler {
	switch ingressType {
	case nginx:
		return &nginxDataHandler{ingressType: ingressType, logKey: []byte("nginx: ")}
	default:
		panic("un support ingress type")
	}
	return nil
}

func NewPoolHandler(config *utils.Config, client *scale.K8SClient) *PoolHandler {
	var ingressType IngressType
	switch config.IngressType {
	case "nginx":
		ingressType = nginx
	case "traefik":
		ingressType = traefik
	default:
		panic("Not support Ingress type")
	}
	workers := make([]handler, defaultPoolSize, defaultPoolSize)
	queues := make([]chan []byte, defaultPoolSize, defaultPoolSize)
	for i := 0; i < defaultPoolSize; i++ {
		workers[i] = newDataHandler(ingressType)
		queues[i] = make(chan []byte, defaultQueueSize)
	}
	poolHandler := &PoolHandler{
		k8sClient:   client,
		config:      config,
		ingressType: ingressType,
		workers:     workers,
		poolSize:    defaultPoolSize,
		queue:       queues,
		qpsRecord:   make(map[string]*metrics.Calculate),
		scaleRecord: make(map[string]*metrics.ScaleRecord),
	}
	poolHandler.startWorkers()
	return poolHandler
}

type PoolHandler struct {
	k8sClient   *scale.K8SClient
	config      *utils.Config
	ingressType IngressType
	workers     []handler
	poolSize    uint8
	queue       []chan []byte
	isStart     bool
	qpsRecord   map[string]*metrics.Calculate
	scaleRecord map[string]*metrics.ScaleRecord
}

func (ph *PoolHandler) startRecord() {
	avgTimeTick := time.Tick(time.Second * time.Duration(60/ph.config.AvgTime))
	for {
		select {
		case <-avgTimeTick:
			for service, calculate := range ph.qpsRecord {
				if v, exist := ph.scaleRecord[service]; exist {
					v.RecordQps(calculate.AvgQps(), metrics.QPSRecordExpire)
				} else {
					ph.scaleRecord[service] = metrics.NewScaleRecord(ph.config.AutoScale.MaxQPS, ph.config.AutoScale.SafeQPS)
				}
			}
		}
	}
}

func (ph *PoolHandler) startEcho(echoTime time.Duration) {
	utils.DisplayQPS(ph.qpsRecord, echoTime)
}

func (ph *PoolHandler) Execute(data []byte) {
	index := time.Now().UnixMilli() % defaultPoolSize
	ph.queue[index] <- data
}

func (ph *PoolHandler) startWorkers() {
	if ph.isStart {
		return
	}
	for i, worker := range ph.workers {
		go func(i int, worker handler) {
			for {
				byteData := <-ph.queue[i]
				accessItem := worker.parseDataWithFilter(byteData, ph.config.AutoScale.Services)
				if accessItem == nil {
					continue
				}
				calculateQPS(accessItem, &ph.qpsRecord)
			}
		}(i, worker)
	}
	echoIntervalTime := time.Second * time.Duration(60/ph.config.AvgTime)
	go ph.startRecord()
	log.Println("start record qps worker success")
	go ph.startEcho(echoIntervalTime)
	log.Println("start echo worker success")
	go utils.AutoScaleByQPS(&ph.scaleRecord, ph.k8sClient, ph.config)
	log.Println("start auto scale worker success")
	ph.isStart = true
}
