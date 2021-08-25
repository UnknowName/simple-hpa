package metrics

import "time"

const (
    avgCount = 5
    Count = avgCount
    expireSecond = 2
    // countExpire = time.Minute * 1
    scaleExpire = time.Minute * 2
)
