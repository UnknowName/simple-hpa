package metrics

import "time"

const (
	avgCount        = 5
	Count           = avgCount
	expireSecond    = 2
	scaleExpire     = time.Minute * 2
	QPSRecordExpire = scaleExpire
)
