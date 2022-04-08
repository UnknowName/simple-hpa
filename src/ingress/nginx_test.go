package ingress

import (
	"log"
	"testing"
	"time"
)

func TestEventEcho(t *testing.T) {
	var timeMsec float64
	timeMsec = 1627007390.044
	timeDur := time.Duration(timeMsec)
	log.Println(time.Unix(timeDur.Milliseconds(), 0))
}
