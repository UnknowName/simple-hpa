package ingress

import (
	"log"
	"testing"
	"time"
)

func TestEvent_Echo(t *testing.T) {
	var time_msec float64
	time_msec = 1627007390.044
	time_d := time.Duration(time_msec)
	log.Println(time.Unix(time_d.Milliseconds(), 0))
}
