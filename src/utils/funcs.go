package utils

import (
	"log"
	"simple-hpa/src/metrics"
	"simple-hpa/src/scale"
	"strings"
	"time"
)

func DisplayQPS(calcuRecord map[string]*metrics.Calculate, echoTime time.Duration) {
	for {
		select {
		case <-time.Tick(echoTime):
			for svc, qps := range calcuRecord {
				if qps == nil {
					continue
				}
				log.Printf("%s latest 5 second avg qps=%.2f 2 second active pod=%d", svc, qps.AvgQps(), qps.GetPodCount())
			}
		}
	}
}

func AutoScaleByQPS(scaleRecord *map[string]*metrics.ScaleRecord, k8sClient *scale.K8SClient, config *Config) {
	checkTime := time.Second*time.Duration(config.AutoScale.SliceSecond) + time.Millisecond*211
	dict := *scaleRecord
	for {
		select {
		case <-time.Tick(checkTime):
			for svc, scRecord := range dict {
				log.Printf("%s is safe=%t is wasteful=%t", svc, scRecord.IsSafe(), scRecord.IsWasteful())
				if (!scRecord.IsSafe() || scRecord.IsWasteful()) && scRecord.Interval() {
					// 说明过量或者过少，都要调整，但是这里记录下上次调整的时间节点，防止频繁的改动
					newCount := scRecord.GetSafeCount()
					if *newCount > config.AutoScale.MaxPod {
						*newCount = config.AutoScale.MaxPod
					} else if *newCount < config.AutoScale.MinPod {
						*newCount = config.AutoScale.MinPod
					}
					go func() {
						svcStrs := strings.Split(svc, ".")
						if len(svcStrs) != 2 {
							log.Println("WARN service name", svc, "please use namespace.service format")
							return
						}
						namespace, service := svcStrs[0], svcStrs[1]
						currCnt, err := k8sClient.GetServicePod(namespace, service)
						if err != nil {
							log.Println("WARN get kubernetes client error ", err)
							return
						}
						if *currCnt == *newCount {
							return
						}
						err = k8sClient.ChangeServicePod(namespace, service, newCount)
						if err != nil {
							log.Println(svc, "scale failed, ", err)
						} else {
							log.Printf("%s scale from %d to %d", svc, *currCnt, *newCount)
							scRecord.ChangeCount(newCount)
							scRecord.ChangeScaleState(true)
						}
					}()
				}
			}
		}
	}
}
