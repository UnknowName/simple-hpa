package handler

import (
	"golang.org/x/net/context"
	"log"
	"strings"
	"time"

	"simple-hpa/src/ingress"
	"simple-hpa/src/metrics"
	"simple-hpa/src/scale"
	"simple-hpa/src/utils"
)

type IngressType uint8

type FilterFunc func(itemChan ingress.Access, services []string, parent context.Context) ingress.Access

const (
	// 一个随机常数，用于防止时间间隔重叠
	randTime                     = time.Millisecond * 211
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
	dict := &ph.qpsRecord
	for {
		select {
		case <-avgTimeTick:
			for service, calculate := range *dict {
				if v, exist := ph.scaleRecord[service]; exist {
					v.RecordQps(calculate.AvgQps(), time.Duration(ph.config.ScaleIntervalTime) * time.Second)
				} else {
					ph.scaleRecord[service] = metrics.NewScaleRecord(ph.config.AutoScale.MaxQPS,
						ph.config.AutoScale.SafeQPS,
						ph.config.AvgTime)
				}
			}
		}
	}
}

func (ph *PoolHandler) startEcho(echoTime time.Duration) {
	for {
		select {
		case <-time.Tick(echoTime):
			for svc, qps := range ph.qpsRecord {
				if qps == nil {
					continue
				}
				log.Printf("%s latest %d second signle pod avg qps=%.2f, %d second active backend pod=%d",
					svc,
					ph.config.AvgTime,
					qps.AvgQps(),
					ph.config.AvgTime,
					qps.GetPodCount())
			}
		}
	}
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
	echoIntervalTime := time.Second * time.Duration(ph.config.AvgTime)
	go ph.startRecord()
	log.Println("start record qps worker success")
	go ph.startEcho(echoIntervalTime)
	log.Println("start echo worker success")
	go ph.startAutoScale()
	log.Println("start auto scale worker success")
	ph.isStart = true
}

func (ph *PoolHandler) startAutoScale() {
	checkTime := time.Second*time.Duration(ph.config.ScaleIntervalTime) + randTime
	for {
		select {
		case <-time.Tick(checkTime):
			for namespaceSvc, scaleRecord := range ph.scaleRecord {
				log.Printf("%s is safe=%t is wasteful=%t", namespaceSvc, scaleRecord.IsSafe(), scaleRecord.IsWasteful())
				if (!scaleRecord.IsSafe() || scaleRecord.IsWasteful()) && scaleRecord.Interval() {
					// 说明过量或者过少，都要调整，但是这里记录下上次调整的时间节点，防止频繁的改动
					newCount := scaleRecord.GetSafeCount()
					if *newCount > ph.config.AutoScale.MaxPod {
						*newCount = ph.config.AutoScale.MaxPod
					} else if *newCount < ph.config.AutoScale.MinPod {
						*newCount = ph.config.AutoScale.MinPod
					}
					go func() {
						svcStrs := strings.Split(namespaceSvc, ".")
						if len(svcStrs) != 2 {
							log.Println("WARN service name error", namespaceSvc, "please use namespace.service format")
							return
						}
						namespace, service := svcStrs[0], svcStrs[1]
						currCnt, err := ph.k8sClient.GetServicePod(namespace, service)
						if err != nil {
							log.Printf("WARN get %s.%s replication count err %s", namespace, service, err )
							return
						}
						if *currCnt == *newCount {
							return
						}
						err = ph.k8sClient.ChangeServicePod(namespace, service, newCount)
						if err != nil {
							log.Printf("WARN %s scale failed %s", namespaceSvc, err)
							return
						} else {
							log.Printf("%s scale from %d to %d", namespaceSvc, *currCnt, *newCount)
							scaleRecord.ChangeCount(newCount)
							scaleRecord.ChangeScaleState(true, ph.config.ScaleIntervalTime)
						}
					}()
				}
			}
		}
	}
}
