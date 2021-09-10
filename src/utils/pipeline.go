package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"simple-hpa/src/ingress"
	"simple-hpa/src/metrics"
	"time"
)

func ParseUDPData(data []byte) <-chan ingress.Access {
	channel := make(chan ingress.Access)
	go func() {
		defer close(channel)
		byteStrings := bytes.Split(data, []byte("nginx: "))
		if len(byteStrings) != 2 {
			log.Println("Not NGINX Ingress data, origin string is:", string(data))
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
		channel <- accessItem
	}()
	return channel
}

func FilterService(itemChan <-chan ingress.Access, services []string) <-chan ingress.Access {
	channel := make(chan ingress.Access)
	go func() {
		defer close(channel)
		data := <-itemChan
		if data == nil {
			log.Println("data is nil")
			return
		}
		for _, service := range services {
			if data.ServiceName() == service {
				channel <- data
			}
		}
	}()
	return channel
}

type serviceInfo struct {
	Name     string
	PodCount int32
	AvgQps   float64
}

func (si *serviceInfo) String() string {
	return fmt.Sprintf("serviceInfo{Name=%s,qps=%f,pod=%d)", si.Name, si.AvgQps, si.PodCount)
}

func CalculateQPS(data <-chan ingress.Access, timeTick <-chan time.Time,
	qpsRecord map[string]*metrics.Calculate) <-chan *serviceInfo {
	channel := make(chan *serviceInfo)
	go func() {
		defer close(channel)
		select {
		case item := <-data:
			if item == nil {
				return
			}
			if record, exist := qpsRecord[item.ServiceName()]; exist {
				record.Update(item.Upstream(), item.AccessTime())
			} else {
				qpsRecord[item.ServiceName()] = metrics.NewCalculate(item.Upstream(), item.AccessTime())
			}
		case <-timeTick:
			for service, calculate := range qpsRecord {
				channel <- &serviceInfo{Name: service, AvgQps: calculate.AvgQps(), PodCount: calculate.GetPodCount()}
			}
		}
	}()
	return channel
}

func RecordQps(qpsChan <-chan *serviceInfo, maxQps, safeQps float64, scaleRecord map[string]*metrics.ScaleRecord) {
	select {
	case data := <-qpsChan:
		if data == nil {
			return
		}
		if v, exist := scaleRecord[data.Name]; exist {
			v.RecordQps(data.AvgQps, metrics.QPSRecordExpire)
		} else {
			scaleRecord[data.Name] = metrics.NewScaleRecord(maxQps, safeQps)
		}
	}
}
