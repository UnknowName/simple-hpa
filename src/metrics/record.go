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
        latestQps: make([]float64, avgCount, avgCount),
        maxQps: maxQps,
        safeQps: safeQps,
        isScaled: make(map[bool]time.Time),
        latestCount: 0,
    }
}

type ScaleRecord struct {
    latestQps []float64
    maxQps    float64
    safeQps   float64
    isScaled  map[bool]time.Time
    latestCount int
}

func (r *ScaleRecord) String() string {
    return fmt.Sprintf("Record{max=%f, safe=%f}", r.maxQps, r.safeQps)
}

// 检查数据组中是否全为false

func (r *ScaleRecord) isState(state svcState) bool {
    var value float64
    switch  {
    case state == 0:
        value = r.maxQps
    case state == 1:
        value = r.safeQps / 2
    default:
        panic(fmt.Sprintf("un know state, %v", state))
    }
    for _, qps := range r.latestQps {
        if qps > value {
            return false
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

func (r *ScaleRecord) GetSafeCount() int {
    sum := float64(0)
    length := float64(0)
    for _, v := range r.latestQps {
        sum += v
        length++
    }
    newCount := int(math.Ceil(sum / length / r.safeQps))
    if newCount < 1 {
        return 1
    }
    return newCount
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

func (r *ScaleRecord) ChangeCount(count int) {
    r.latestCount = count
}

func (r *ScaleRecord) GetCount() int {
    if r.latestCount < 1 {
        return 1
    }
    return r.latestCount
}

func (r *ScaleRecord) RecordQps(qps float64) {
    r.latestQps = append(r.latestQps[1:], qps)
}

func (r *ScaleRecord) RecordPodCount(v int) {
    r.latestCount = v
}