package ingress

import "time"

type Meta struct {
	Namespace string `json:"namespace"`
	Service   string `json:"service"`
}

type Access interface {
	AccessTime() time.Time
	Upstream() string
	ServiceName() string
}
