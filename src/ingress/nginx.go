package ingress

import (
	"encoding/json"
	"fmt"
	"time"
)

// 不需要New函数，因为来源于JSON化，不会主动创建

type NGINXAccess struct {
	Meta
	Time                 time.Time `json:"time_msec"`
	UpstreamAddr         string    `json:"upstream_addr"`
	UpstreamResponseTime string    `json:"upstream_response_time"`
	Status               int       `json:"status"`
}

func (na *NGINXAccess) ServiceName() string {
	return fmt.Sprintf("%s.%s", na.Service, na.Namespace)
}

func (na *NGINXAccess) AccessTime() time.Time {
	return na.Time
}

func (na *NGINXAccess) Upstream() string {
	return na.UpstreamAddr
}

func (na *NGINXAccess) UnmarshalJSON(data []byte) error {
	tmp := struct {
		Meta
		TimeFloat    float64 `json:"time_msec"`
		UpstreamAddr string  `json:"upstream_addr"`
		ResponseTime string  `json:"upstream_response_time"`
		Status       int     `json:"status"`
	}{}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	na.Namespace = tmp.Namespace
	na.Service = tmp.Service
	na.UpstreamAddr = tmp.UpstreamAddr
	na.Status = tmp.Status
	na.UpstreamResponseTime = tmp.ResponseTime
	na.Time = time.Unix(int64(tmp.TimeFloat), 0)
	return nil
}
