package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"simple-hpa/src/ingress"
	"simple-hpa/src/metrics"
	"simple-hpa/src/utils"
	"time"
)

const bufSize = 1024

var (
	config *utils.Config
	calcuRecord map[string]*metrics.Calculate
	scaleRecord map[string]*metrics.ScaleRecord
)


func init() {
	log.SetFlags(log.Ldate | log.Lmicroseconds | log.Llongfile)
	if len(os.Args) != 2 {
		log.Println("use", os.Args[0], "configFile")
		time.Sleep(time.Second * 5)
		return
	}
	if config == nil {
		config = utils.NewConfig("config.yml")
	}
	if config.AutoScale.Services == nil || len(config.AutoScale.Services) == 0 {
		log.Println("WARNING, Auto scale dest service not defined")
	}
	calcuRecord = make(map[string]*metrics.Calculate)
	scaleRecord = make(map[string]*metrics.ScaleRecord)
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
	// 对QPS每隔固定时间进行取样
	go func() {
		for {
			select {
			case <-time.Tick(time.Second * (60 / metrics.Count)):
				for svc, qps := range calcuRecord {
					scaleRecord[svc].RecordQps(qps.AvgQps())
					log.Printf("%s current qps: %f", svc, qps.AvgQps())
				}
			}
			time.Sleep(time.Millisecond * 100)
		}
	}()
	// 计算一分钟的平均QPS是否在安全范围之内
	go func() {
		for {
			select {
			case <-time.Tick(time.Minute + time.Millisecond * 100):
				for svc, scRecord := range scaleRecord {
					log.Printf("%s is safe: %t, is wasteful: %t", svc, scRecord.IsSafe(), scRecord.IsWasteful())
					if (!scRecord.IsSafe() || scRecord.IsWasteful()) && scRecord.Interval() {
						// 说明过量或者过少，都要调整，但是这里记录下上次调整的时间节点，防止频繁的改动
						newCount := scRecord.GetSafeCount(calcuRecord[svc].AvgQps())
						if newCount != scRecord.GetCount() {
							scRecord.ChangeCount(newCount)
							scRecord.ChangeScaleState(true)
							log.Printf("service %s %d -> %d", svc, scRecord.GetCount(), newCount)
						}
					}
				}
			}
			time.Sleep(time.Millisecond * 100)
		}
	}()
	for {
		buf := make([]byte, bufSize)
		n, err := conn.Read(buf)
		if err != nil || n == 0 {
			log.Println("read error ", err)
			continue
		}
		go parseJson(buf[:n], config)
	}
}

func parseJson(buf []byte, config *utils.Config) {
	byteStrings := bytes.Split(buf, []byte("nginx: "))
	if len(byteStrings) != 2 {
		log.Println("Not NGINX Ingress data, origin string is:", string(buf))
		return
	}
	jsonByte := byteStrings[1]
	if !bytes.HasPrefix(jsonByte, []byte("{")) {
		return
	}
	accessItem := new(ingress.NGINXAccess)
	// JSON化之前，去掉URL里面的中文\x
	if err := json.Unmarshal(bytes.ReplaceAll(jsonByte, []byte("\\x"), []byte("")), accessItem); err != nil {
		log.Println("To json failed ", err, "origin string:", string(jsonByte))
		return
	}
	if accessItem.ServiceName() == "." {
		return
	}
	for _, service := range config.AutoScale.Services {
		if service == accessItem.ServiceName() {
			go calculateQPS(accessItem, config)
			break
		}
	}
}

func calculateQPS(access ingress.Access, config *utils.Config) {
	if serviceMetric, exist := calcuRecord[access.ServiceName()]; exist {
		serviceMetric.Update(access.Upstream(), access.AccessTime())
	} else {
		calcuRecord[access.ServiceName()] = metrics.NewCalculate(access.Upstream(), access.AccessTime())
	}
	if _, exit := scaleRecord[access.ServiceName()]; !exit {
		scaleRecord[access.ServiceName()] = metrics.NewScaleRecord(config.AutoScale.Max, config.AutoScale.Safe)
	}
}