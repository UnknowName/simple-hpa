package metrics

import (
	"log"
	"testing"
	"time"
)

func TestNewRecord(t *testing.T) {
	var max int
	max = 10
	r := NewScaleRecord(float32(max), 6, 1, 5, 120)
	log.Println(r.Interval())
	log.Println(r.Interval())
	time.Sleep(time.Minute * 1)
	log.Println(r.Interval())
}
