package metrics

import (
	"fmt"
	"sync"
	"time"
)

func NewCalculate(upstreamAddr string, accessTime time.Time) *Calculate {
	return &Calculate{
		secondTick:     time.Tick(time.Second),
		resetTick:      time.Tick(time.Second * avgCount),
		currentCount:   1,
		durationCounts: make([]map[int]time.Time, avgCount, avgCount),
		avg:            0,
		mutex:          sync.Mutex{},
		upstreams:      map[string]time.Time{upstreamAddr: accessTime},
	}
}

type Calculate struct {
	secondTick <-chan time.Time
	resetTick      <-chan time.Time
	currentCount   int
	durationCounts []map[int]time.Time
	avg            float64
	mutex          sync.Mutex
	upstreams      map[string]time.Time
}

func (c *Calculate) String() string {
	return fmt.Sprintf("Calculate{avg=%f}", c.avg)
}

func (c *Calculate) Update(upstream string, accessTime time.Time) {
	c.upstreams[upstream] = accessTime
	c.mutex.Lock()
	defer c.mutex.Unlock()
	select {
	case <-c.secondTick:
		data := map[int]time.Time{c.currentCount: time.Now().Add(scaleExpire)}
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
		c.avg = float64(sum) / avgCount
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

// TODO QPS的历史值也得取掉
func (c *Calculate) clean() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	for upstream, accessTime := range c.upstreams {
		if accessTime.Add(time.Second * expireSecond).Before(time.Now()) {
			delete(c.upstreams, upstream)
		}
	}
	for index, dict := range c.durationCounts {
		for cnt,addTime := range dict {
			if cnt > 0 && addTime.Before(time.Now()) {
				c.durationCounts[index] = map[int]time.Time{0: time.Now()}
			}
		}
	}
}