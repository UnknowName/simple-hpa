package handler

import (
	"simple-hpa/src/ingress"
	"simple-hpa/src/metrics"
	"sync"
)

var mutex sync.Mutex

func calculateQPS(item ingress.Access, qpsRecord *map[string]*metrics.Calculate) {
	mutex.Lock()
	defer mutex.Unlock()
	dict := *qpsRecord
	if record, exist := dict[item.ServiceName()]; exist {
		record.Update(item.Upstream(), item.AccessTime())
	} else {
		dict[item.ServiceName()] = metrics.NewCalculate(item.Upstream(), item.AccessTime(), 5)
	}
}
