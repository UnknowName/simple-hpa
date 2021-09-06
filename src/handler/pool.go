package handler

import (
	"log"
	"simple-hpa/src/ingress"
	"simple-hpa/src/metrics"
	"simple-hpa/src/scale"
	"simple-hpa/src/utils"
	"time"
)

type IngressType uint8

const (
	defaultQueueSize             = 1024
	defaultPoolSize              = 10
	nginx            IngressType = iota
	traefik
)

type handler interface {
	parseData([]byte) <-chan ingress.Access
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
			avgTimeTick := time.Tick(time.Second * time.Duration(60 / ph.config.AvgTime))
			for {
				select {
				case byteData := <-ph.queue[i]:
					accessChan := worker.parseData(byteData)
					accessChan = utils.FilterService(accessChan, ph.config.AutoScale.Services)
					qpsChan := utils.CalculateQPS(accessChan, avgTimeTick, ph.qpsRecord)
					go utils.RecordQps(qpsChan, ph.config.AutoScale.MaxQPS, ph.config.AutoScale.SafeQPS, ph.scaleRecord)
				}
			}
		}(i, worker)
	}
	sleepTime := time.Millisecond * 121
	echoIntervalTime := time.Second * time.Duration(60 / ph.config.AvgTime)
	go utils.DisplayQPS(ph.qpsRecord, echoIntervalTime, sleepTime + 100)
	log.Println("start echo worker success")
	go utils.AutoScaleByQPS(ph.scaleRecord, sleepTime - 200, ph.k8sClient, ph.config)
	log.Println("start auto scale worker success")
	ph.isStart = true
}
