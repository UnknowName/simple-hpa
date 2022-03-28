package metrics

import (
	"fmt"
	"sync"
	"time"
)

func NewCalculate(upstreamAddr string, accessTime time.Time, avgTime int) *Calculate {
	return &Calculate{
		secondTick:     time.Tick(time.Second),
		resetTick:      time.Tick(time.Duration(avgTime) * time.Second),
		currentCount:   1,
		durationCounts: make([]map[int]time.Time, avgTime, avgTime),
		avg:            0,
		avgTime:        avgTime,
		mutex:          sync.Mutex{},
		upstreams:      map[string]time.Time{upstreamAddr: accessTime},
	}
}

type Calculate struct {
	secondTick     <-chan time.Time
	resetTick      <-chan time.Time
	currentCount   int
	durationCounts []map[int]time.Time
	avg            float64  // 当前avgCnt秒内的平均值
	avgTime        int      // 一分钟内取样多少次
	mutex          sync.Mutex
	upstreams      map[string]time.Time
}

func (c *Calculate) String() string {
	return fmt.Sprintf("Calculate{avg=%f}", c.avg)
}

func (c *Calculate) Update(upstream string, accessTime time.Time) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.upstreams[upstream] = accessTime
	select {
	case <-c.secondTick:
		// 增加一个时间是防止前面有值，后面为空时，值不对的情况，这时就不要它了，显示为0
		data := map[int]time.Time{c.currentCount: time.Now().Add(time.Duration(c.avgTime) * time.Second)}
		c.durationCounts = append(c.durationCounts[1:], data)
		c.currentCount = 0
	case <-c.resetTick:
		sum := 0
		for _, dict := range c.durationCounts {
			for count, addTime := range dict {
				if addTime.After(time.Now()) {
					sum += count
				}
			}
		}
		c.avg = float64(sum) / float64(c.avgTime)
	default:
	}
	c.currentCount += 1
	go c.clean()
}

func (c *Calculate) AvgQps() float64 {
	go c.clean()
	if len(c.upstreams) == 0 || c.avg == 0 {
		return 0
	}
	return c.avg / float64(len(c.upstreams))
}

func (c *Calculate) GetPodCount() int32 {
	c.clean()
	length := int32(len(c.upstreams))
	return length
}

func (c *Calculate) clean() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	for upstream, accessTime := range c.upstreams {
		if accessTime.Add(time.Second * time.Duration(c.avgTime)).Before(time.Now()) {
			delete(c.upstreams, upstream)
		}
	}
	for index, dict := range c.durationCounts {
		for cnt, addTime := range dict {
			if cnt > 0 && addTime.Before(time.Now()) {
				c.durationCounts[index] = map[int]time.Time{0: time.Now()}
			}
		}
	}
}
