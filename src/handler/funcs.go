package handler

import (
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"

	"simple-hpa/src/ingress"
	"simple-hpa/src/metrics"
)

const jsonTry = 3

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

func Unmarshal(data []byte, o ingress.Access) error {
	wg := sync.WaitGroup{}
	wg.Add(jsonTry)
	var cnt uint32
	go func() {
		defer wg.Done()
		err := json.Unmarshal(data, o)
		if err != nil {
			atomic.AddUint32(&cnt, 1)
		}
	}()

	go func() {
		defer wg.Done()
		err := json.Unmarshal(append(data, '}'), o)
		if err != nil {
			atomic.AddUint32(&cnt, 1)
		}
	}()

	go func() {
		defer wg.Done()
		err := json.Unmarshal(data[:len(data)-1], o)
		if err != nil {
			atomic.AddUint32(&cnt, 1)
		}
	}()
	wg.Wait()
	if cnt == jsonTry {
		return errors.New(string(data))
	}
	return nil
}
