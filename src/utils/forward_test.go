package utils

import (
    "fmt"
    "strings"
    "testing"
)

func TestNewForward(t *testing.T) {
    name := "rsyslog=128.0.255.10:514, es=128.0.255.10:9200"
    for _, forward := range strings.Split(name, ",") {
        typeName, addr := strings.Split(forward, "=")[0], strings.Split(forward, "=")[1]
        fmt.Println(typeName, "=   ", addr)
    }
}
