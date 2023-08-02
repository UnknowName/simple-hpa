package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"

	"auto-scale/src/ingress"
	"auto-scale/src/metrics"
)

const (
	jsonTry      = 2
	bracesSymbol = '}'
	endSymbol    = '"'
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

func ConcurUnmarshal(data []byte, ing ingress.Access) error {
	wg := sync.WaitGroup{}
	wg.Add(jsonTry)
	var cnt uint32
	go func() {
		defer wg.Done()
		err := json.Unmarshal(data, ing)
		if err != nil {
			atomic.AddUint32(&cnt, 1)
		}
	}()

	go func() {
		defer wg.Done()
		n := bytes.Count(data, []byte{bracesSymbol})
		i := len(data)
		if n > 1 {
			// JSON字符串中多出一个}
			for n >= 1 {
				i--
				if data[i] == bracesSymbol {
					n--
				}
			}
			i++
		} else if n == 1 {
			// step2: 处理像{}xxx 合法外多出的字符串或正常结束的
			for data[i-1] != bracesSymbol {
				i--
			}
		} else {
			// JSON字符串中缺'}'的,因为最后会添加}，因此这里只检查最后一个是不是'"'
			if data[i-1] != endSymbol {
				data = append(data, endSymbol)
				i += 2
			} else {
				i++
			}
		}
		data = append(data, bracesSymbol)
		err := json.Unmarshal(data[:i], ing)
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
