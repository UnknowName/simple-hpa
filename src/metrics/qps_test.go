package metrics

import (
	"log"
	"testing"
	"time"
)

func TestNewCalculate(t *testing.T) {
	cal := NewCalculate("172.18.0.0:8080", time.Now(), 5)
	log.Println(cal)
	queue := make([]int, 5, 5)
	log.Println(queue)
	time.Sleep(time.Duration(5) * time.Second)
	queue = append(queue[1:], 20)
	queue = append(queue[1:], 30)
	queue = append(queue[1:], 40)
	queue = append(queue[1:], 50)
	queue = append(queue[1:], 60)
	queue = append(queue[1:], 70)
	log.Println(queue)
}
