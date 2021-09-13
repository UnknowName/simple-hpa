package utils

import (
	"context"
	"fmt"
	"github.com/opentracing/opentracing-go"
	"simple-hpa/src/ingress"
	"simple-hpa/src/metrics"
	"time"
)

type serviceInfo struct {
	Name     string
	PodCount int32
	AvgQps   float64
}

func (si *serviceInfo) String() string {
	return fmt.Sprintf("serviceInfo{Name=%s,qps=%f,pod=%d)", si.Name, si.AvgQps, si.PodCount)
}

func Provider(avgTime int, maxQps, safeQps float64, qpsRecord map[string]*metrics.Calculate, scaleRecord map[string]*metrics.ScaleRecord) {
	avgTimeTick := time.Tick(time.Second * time.Duration(60/avgTime))
	select {
	case <-avgTimeTick:
		sis := make([]*serviceInfo, 0)
		for service, calculate := range qpsRecord {
			si := &serviceInfo{Name: service, AvgQps: calculate.AvgQps(), PodCount: calculate.GetPodCount()}
			sis = append(sis, si)
		}
		RecordQps(sis, maxQps, safeQps, scaleRecord)
	}
}

func CalculateQPS(data <-chan ingress.Access,
	qpsRecord map[string]*metrics.Calculate, parent context.Context) {
	span, ctx := opentracing.StartSpanFromContext(parent, "CalculateQPS")
	span.LogKV("CalculateQPS", "start")
	go func() {
		defer func() {
			ctx.Done()
			span.Finish()
		}()
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
}

func RecordQps(sis []*serviceInfo, maxQps, safeQps float64, scaleRecord map[string]*metrics.ScaleRecord) {
	for _, i := range sis {
		if v, exist := scaleRecord[i.Name]; exist {
			v.RecordQps(i.AvgQps, metrics.QPSRecordExpire)
		} else {
			scaleRecord[i.Name] = metrics.NewScaleRecord(maxQps, safeQps)
		}
	}
}
