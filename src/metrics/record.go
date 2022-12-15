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
	minute = 60
)

func NewScaleRecord(maxQps, safeQps, factor float32, avgTime, scaleIntervalTime int) *ScaleRecord {
	avgCount := minute / avgTime
	initTime := time.Now().Add(time.Second * time.Duration(scaleIntervalTime))
	return &ScaleRecord{
		latestQps:   make([]map[time.Time]float32, avgCount, avgCount),
		maxQps:      maxQps,
		safeQps:     safeQps,
		isScaled:    map[bool]time.Time{true: initTime},
		factor:      factor,
		latestCount: nil,
	}
}

type ScaleRecord struct {
	latestQps   []map[time.Time]float32
	maxQps      float32
	safeQps     float32
	isScaled    map[bool]time.Time
	factor      float32
	latestCount *int32
}

func (r *ScaleRecord) String() string {
	return fmt.Sprintf("Record{max=%f, safe=%f}", r.maxQps, r.safeQps)
}

func (r *ScaleRecord) isState(state svcState) bool {
	var value float32
	switch {
	case state == 0:
		value = r.maxQps
	case state == 1:
		value = r.safeQps / 2
	default:
		panic(fmt.Sprintf("un know state, %v", state))
	}
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

func (r *ScaleRecord) GetSafeCount() *int32 {
	count := new(int32)
	sum := float32(0)
	length := float32(0)
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
	*count = int32(math.Ceil(float64(sum / length / r.safeQps)))
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

func (r *ScaleRecord) ChangeScaleState(state bool, scaleIntervalTime int) {
	r.isScaled[state] = time.Now().Add(time.Duration(scaleIntervalTime) * time.Second)
}

func (r *ScaleRecord) ChangeCount(count *int32) {
	r.latestCount = count
}

func (r *ScaleRecord) GetCount() *int32 {
	if r.latestCount != nil && *r.latestCount < 1 {
		*r.latestCount = 1
	}
	return r.latestCount
}

func (r *ScaleRecord) RecordQps(qps float32, expireTime time.Duration) {
	qpsDict := map[time.Time]float32{time.Now().Add(expireTime): qps * r.factor}
	r.latestQps = append(r.latestQps[1:], qpsDict)
}
