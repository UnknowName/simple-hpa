package scale

import (
    "log"
    "testing"
)

func TestChangeServicePod(t *testing.T) {
    cnt := int32(1)
    err := ChangeServicePod("demo-dev.daohao", &cnt)
    log.Println(err)
}