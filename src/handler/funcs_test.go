package handler

import (
    "fmt"
    "testing"

    ingress2 "auto-scale/src/ingress"
)

func TestUnmarshal(t *testing.T) {
    str := "{\"time_str\": \"2022-10-08T06:49:58+00:00\", \"time_msec\": 1665211798.921, " +
        "\"upstream_addr\": \"172.16.0.181:5000\", \"namespace\": \"wxd\", \"status\": 200, " +
        "\"service\": \"sixunmall-web-host\""
    byteStr := []byte(str)
    accessItem := new(ingress2.NGINXAccess)
    err := Unmarshal(byteStr, accessItem)
    fmt.Println(err)
    fmt.Println(accessItem)
}
