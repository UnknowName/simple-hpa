package handler

import (
    "fmt"
    "log"
    "testing"
    "time"

    "auto-scale/src/ingress"
)

func TestUnmarshal(t *testing.T) {
    str := "{\"time_str\": \"2022-10-08T06:49:58+00:00\", \"time_msec\": 1665211798.921, " +
        "\"upstream_addr\": \"172.16.0.181:5000\", \"namespace\": \"wxd\", \"status\": 200, " +
        "\"service\": \"sixunmall-web-host\""
    byteStr := []byte(str)
    accessItem := new(ingress.NGINXAccess)
    err := ConcurUnmarshal(byteStr, accessItem)
    fmt.Println(err)
    fmt.Println(accessItem)
}

func TestNewRingBuffer(t *testing.T) {
    buf := newRingBuffer(2, time.Second)
    buf.Insert(12)
    buf.Insert(13)
    buf.Insert(16)
    log.Println(buf.Total())
    time.Sleep(time.Second * 1)
    log.Println("get new total")
    log.Println(buf.Total())
    maps := map[string]string{"a":"b","c":"d"}
    for key := range maps {
        log.Println(key)
    }
}
