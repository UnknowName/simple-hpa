package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"simple-hpa/src/scale"
	"strings"
	"time"

	"simple-hpa/src/metrics"
	"simple-hpa/src/utils"
)

const (
	bufSize   = 1024
	echoTime  = time.Second * 12
	sleepTime = time.Millisecond * 113
	checkTime = time.Minute + time.Millisecond*211
)

var (
	config      *utils.Config
	calcuRecord map[string]*metrics.Calculate
	scaleRecord map[string]*metrics.ScaleRecord
	k8sClient   *scale.K8SClient
)

func init() {
	log.SetFlags(log.Ldate | log.Lmicroseconds | log.Llongfile)
	if len(os.Args) != 2 {
		log.Println("use", os.Args[0], "configFile")
		time.Sleep(time.Second * 5)
		return
	}
	if config == nil {
		config = utils.NewConfig(os.Args[1])
	}
	if config.AutoScale.Services == nil || len(config.AutoScale.Services) == 0 {
		log.Println("WARNING, Auto scale dest service not defined")
	}
	calcuRecord = make(map[string]*metrics.Calculate)
	scaleRecord = make(map[string]*metrics.ScaleRecord)
	k8sClient = scale.NewK8SClient()
}

func main() {
	listenAddr := fmt.Sprintf("%s:%d", config.Listen.ListenAddr, config.Listen.Port)
	addr, err := net.ResolveUDPAddr(config.Listen.NetType, listenAddr)
	if err != nil {
		log.Fatalln("Resolve udp address error ", err)
		return
	}
	conn, err := net.ListenUDP(config.Listen.NetType, addr)
	if err != nil {
		log.Fatalln("Listen on ", addr.IP, "failed ", err)
		return
	}
	log.Printf("App listen on %s/%s", listenAddr, config.Listen.NetType)
	log.Printf("Auto scale services: %s", config.AutoScale.Services)
	defer conn.Close()
	// 对QPS每隔固定时间进行打印
	go func() {
		for {
			select {
			case <-time.Tick(echoTime):
				for svc, qps := range calcuRecord {
					if qps == nil {
						continue
					}
					log.Printf("%s current qps=%.2f calculate count=%d", svc, qps.AvgQps(), qps.GetPodCount())
				}
			}
			time.Sleep(sleepTime)
		}
	}()
	// 计算一分钟的平均QPS是否在安全范围之内
	go func() {
		for {
			select {
			case <-time.Tick(checkTime):
				for svc, scRecord := range scaleRecord {
					log.Printf("%s is safe: %t, is wasteful: %t, scaleRecordCnt: %d", svc, scRecord.IsSafe(),
						scRecord.IsWasteful(), scRecord.GetCount())
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
								log.Println("WARN service name", svc, "error")
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
								log.Printf("%s scale from %d to %d", svc, scRecord.GetCount(), *newCount)
								scRecord.ChangeCount(*newCount)
								scRecord.ChangeScaleState(true)
							}
						}()
					}
					time.Sleep(sleepTime)
				}
			}
			time.Sleep(sleepTime)
		}
	}()
	avgTimeTick := time.Tick(time.Second * (60 / metrics.Count))
	for {
		buf := make([]byte, bufSize)
		n, err := conn.Read(buf)
		if err != nil || n == 0 {
			log.Println("read error ", err)
			continue
		}
		go func() {
			accessChan := utils.ParseUDPData(buf[:n])
			accessChan = utils.FilterService(accessChan, config.AutoScale.Services)
			qpsChan := utils.CalculateQPS(accessChan, avgTimeTick, calcuRecord)
			utils.RecordQps(qpsChan, config.AutoScale.MaxQPS, config.AutoScale.SafeQPS, scaleRecord)
		}()
	}
}
