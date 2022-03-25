package metrics

import "time"

// TODO 取消这些常量，通过配置文件传入
const (
	expireSecond    = 2
	scaleExpire     = time.Minute * 5
	QPSRecordExpire = scaleExpire
)
