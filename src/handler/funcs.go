package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"auto-scale/src/ingress"
)

const (
	jsonTry      = 2
	bracesSymbol = '}'
	endSymbol    = '"'
	minuteCount  = 60
)

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

func newRingBuffer(max uint, duration time.Duration) *RingBuffer {
	r := &RingBuffer{
		cap:   max,
		data:  make([]int, max),
		mutex: sync.RWMutex{},
	}
	go r.expire(duration)
	return r
}

type RingBuffer struct {
	cap   uint // 容量
	size  uint // 当前大小
	mutex sync.RWMutex
	in    uint  // 写入索引位置
	out   uint  // 读取索引位置
	data  []int // 内部最终数据
}

func (rb *RingBuffer) expire(duration time.Duration) {
	ticker := time.NewTicker(duration)
	for {
		select {
		case <-ticker.C:
			rb.mutex.RLock()
			if rb.size > 0 {
				rb.out = (rb.out + 1) % rb.cap
				rb.size--
			}
			rb.mutex.RUnlock()
		}
	}
}

func (rb *RingBuffer) isFull() bool {
	return rb.cap == rb.size
}

func (rb *RingBuffer) Insert(data int) {
	if rb.isFull() {
		log.Printf("buffer is full, %v can add", data)
		return
	}
	rb.mutex.Lock()
	defer rb.mutex.Unlock()
	rb.data[rb.in] = data
	rb.size++
	rb.in = (rb.in + 1) % rb.cap
}

func (rb *RingBuffer) Total() int {
	var total int
	if rb.size == 0 {
		return total
	}
	rb.mutex.RLock()
	defer rb.mutex.RUnlock()
	var i uint
	for i = 0; i < rb.size; i++ {
		total += rb.data[rb.out]
	}
	rb.size = 0
	return total
}

func newUpstream(expire time.Duration) *UpStream {
	v := &UpStream{mutex: sync.Mutex{}, duration: expire, backends: make(map[string]time.Time)}
	go v.expire()
	return v
}

type UpStream struct {
	mutex    sync.Mutex
	duration time.Duration
	backends map[string]time.Time
}

func (us *UpStream) expire() {
	ticker := time.NewTicker(us.duration)
	for {
		select {
		case <- ticker.C:
			us.mutex.Lock()
			for backend, inTime := range us.backends {
				if inTime.Add(us.duration).Before(time.Now()) {
					delete(us.backends, backend)
				}
			}
			us.mutex.Unlock()
		}
	}
}

func (us *UpStream) Total() int {
	return len(us.backends)
}

func (us *UpStream) Update(upstream string, accessTime time.Time) {
	us.mutex.Lock()
	defer us.mutex.Unlock()
	us.backends[upstream] = accessTime
}

// 每avgTime的一个记录

type Record struct {
	ServiceName    string
	TotalQps       int
	TotalUpstreams int
}

func (r *Record) AvgQps() float32 {
	if r.TotalQps == 0 || r.TotalUpstreams == 0 {
		return 0
	}
	return float32(r.TotalQps) / float32(r.TotalUpstreams)
}

func NewCalculator(svcName string, frequency int) *Calculator {
	duration := time.Duration(frequency) * time.Second
	r := &Calculator{
		mutex:      sync.RWMutex{},
		duration:   duration,
		qpsCal:     newRingBuffer(minuteCount,duration),
		podCal:     newUpstream(duration),
		currentCnt: 0,
		secTicker:  time.NewTicker(time.Second),
		resultChan: make(chan *Record, frequency),
		serviceName: svcName,
	}
	go r.inPipe()
	return r
}

type Calculator struct {
	mutex      sync.RWMutex
	duration   time.Duration          // 允许接收的时间范围，防止很早之前的数据
	qpsCal     *RingBuffer            // qps计算器，
	podCal     *UpStream              // 服务Pod的计数,key为服务名
	currentCnt int                    // 一秒之内的数据，一秒后会加入到data里面
	secTicker  *time.Ticker           // 重置时钟
	resultChan chan *Record           // 计算出结果后的
	// inTicker    *time.Ticker
	serviceName string
}

func (c *Calculator) Update(v ingress.Access) {
	if v.AccessTime().Add(c.duration).Before(time.Now()) {
		return
	}
	c.mutex.Lock()
	c.currentCnt++
	c.mutex.Unlock()
	select {
	case <-c.secTicker.C:
		c.qpsCal.Insert(c.currentCnt)
		c.currentCnt = 1
	default:
	}
	c.podCal.Update(v.Upstream(), v.AccessTime())
}

func (c *Calculator) inPipe() {
	ticker := time.NewTicker(c.duration+randTime)
	for {
		select {
		case <-ticker.C:
			c.resultChan <- &Record{ServiceName: c.serviceName,
				TotalQps:       c.qpsCal.Total() + c.currentCnt,
				TotalUpstreams: c.podCal.Total(),
			}
		}
	}
}

func (c *Calculator) Pipeline() <-chan *Record {
	return c.resultChan
}