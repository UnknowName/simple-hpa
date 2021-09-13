package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/opentracing/opentracing-go"
	"log"
	"simple-hpa/src/ingress"
	"simple-hpa/src/metrics"
	"sync"
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

func FilterService(itemChan ingress.Access, services []string, parent context.Context) ingress.Access {
	span, ctx := opentracing.StartSpanFromContext(parent, "filterService")
	defer ctx.Done()
	span.LogKV("filterService", "start")
	span.LogKV("filterService", "go func")
	for _, service := range services {
		if itemChan.ServiceName() == service {
			span.LogKV("filterService", "data.ServiceName() == service")
			return itemChan
			span.LogKV("filterService", "complete ...")
		}
	}
	return nil
}

type serviceInfo struct {
	Name     string
	PodCount int32
	AvgQps   float64
}

func (si *serviceInfo) String() string {
	return fmt.Sprintf("serviceInfo{Name=%s,qps=%f,pod=%d)", si.Name, si.AvgQps, si.PodCount)
}

var a int

var mutex sync.Mutex

func CalculateQPS(data <-chan ingress.Access, timeTick <-chan time.Time,
	qpsRecord map[string]*metrics.Calculate, parent context.Context) <-chan *serviceInfo {
	span, ctx := opentracing.StartSpanFromContext(parent, "CalculateQPS")
	span.LogKV("CalculateQPS", "start")
	channel := make(chan *serviceInfo)
	go func() {
		defer func() {
			ctx.Done()
			span.Finish()
		}()
		defer close(channel)
		select {
		case item := <-data:
			span.LogKV("CalculateQPS", "get data success")
			if item == nil {
				return
			}
			if record, exist := qpsRecord[item.ServiceName()]; exist {
				record.Update(item.Upstream(), item.AccessTime())
			} else {
				qpsRecord[item.ServiceName()] = metrics.NewCalculate(item.Upstream(), item.AccessTime())
			}
			/*case <-timeTick:
			span.LogKV("CalculateQPS", "time tick")
			span1, ctx1 := opentracing.StartSpanFromContext(parent, "CalculateQPS")
			defer func() {
				ctx1.Done()
				span1.Finish()
			}()
			mutex.Lock()
			a = a + 1
			span1.LogKV("tick", fmt.Sprintf("number --> %d", a))
			mutex.Unlock()
			for service, calculate := range qpsRecord {
				channel <- &serviceInfo{Name: service, AvgQps: calculate.AvgQps(), PodCount: calculate.GetPodCount()}
			}
			span1.LogKV("tick", "complete")*/
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
