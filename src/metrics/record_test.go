package metrics

import (
	"log"
	"testing"
	"time"
)

func TestNewRecord(t *testing.T) {
	var max int
	max = 10
	r := NewScaleRecord(float64(max), 6)
	log.Println(r.Interval())
	r.ChangeScaleState(true)
	log.Println(r.Interval())
	time.Sleep(time.Minute * 1)
	log.Println(r.Interval())
}
