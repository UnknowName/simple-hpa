package metrics

import (
	"fmt"
	"math"
	"time"
)

type svcState int

const (
	safe svcState = iota
	wasteful
)

func NewScaleRecord(maxQps, safeQps float64) *ScaleRecord {
	return &ScaleRecord{
		latestQps:   make([]map[time.Time]float64, avgCount, avgCount),
		maxQps:      maxQps,
		safeQps:     safeQps,
		isScaled:    make(map[bool]time.Time),
		latestCount: 0,
	}
}

type ScaleRecord struct {
	latestQps   []map[time.Time]float64
	maxQps      float64
	safeQps     float64
	isScaled    map[bool]time.Time
	latestCount int32
}

func (r *ScaleRecord) String() string {
	return fmt.Sprintf("Record{max=%f, safe=%f}", r.maxQps, r.safeQps)
}

func (r *ScaleRecord) isState(state svcState) bool {
	var value float64
	switch {
	case state == 0:
		value = r.maxQps
	case state == 1:
		value = r.safeQps / 2
	default:
		panic(fmt.Sprintf("un know state, %v", state))
	}
	// TODO 队列里面的数据还是历史的，如果没有更新请求进来，会一直保留着，要加个时间，清理掉过期的数据
	for _, qpsDict := range r.latestQps {
		for qpsTime, qps := range qpsDict {
			if qpsTime.After(time.Now()) && qps > value {
				return false
			}
		}
	}
	return true
}

func (r *ScaleRecord) IsSafe() bool {
	// avgQPS < config.max
	var state svcState
	state = safe
	return r.isState(state)
}

func (r *ScaleRecord) GetSafeCount() (count *int32) {
	count = new(int32)
	sum := float64(0)
	length := float64(0)
	for _, qpsDict := range r.latestQps {
		for qpsTime, qps := range qpsDict {
			if qpsTime.After(time.Now()) {
				sum += qps
				length++
			}
		}
	}
	if sum == 0 {
		*count = 1
		return count
	}
	*count = int32(math.Ceil(sum / length / r.safeQps))
	if *count < int32(1) {
		*count = int32(1)
	}
	return count
}

func (r *ScaleRecord) IsWasteful() bool {
	var state svcState
	state = wasteful
	return r.isState(state)
}

func (r *ScaleRecord) Interval() bool {
	for _, v := range r.isScaled {
		if v.After(time.Now()) {
			return false
		}
	}
	return true
}

func (r *ScaleRecord) ChangeScaleState(state bool) {
	r.isScaled[state] = time.Now().Add(scaleExpire)
}

func (r *ScaleRecord) ChangeCount(count int32) {
	r.latestCount = count
}

func (r *ScaleRecord) GetCount() int32 {
	if r.latestCount < 1 {
		return 1
	}
	return r.latestCount
}

func (r *ScaleRecord) RecordQps(qps float64, expireTime time.Duration) {
	qpsDict := map[time.Time]float64{time.Now().Add(expireTime): qps}
	r.latestQps = append(r.latestQps[1:], qpsDict)
}

func (r *ScaleRecord) RecordPodCount(v int32) {
	r.latestCount = v
}
