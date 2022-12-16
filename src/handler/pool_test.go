package handler

import (
    "auto-scale/src/scale"
    "auto-scale/src/utils"
    "testing"
    "time"
)

func TestNewPoolHandler(t *testing.T) {
    config := utils.NewConfig("F:\\GoCodes\\simple-hpa\\config.yaml")
    client := scale.NewK8SClient()
    pool := NewPoolHandler(config, client)
    pool.Execute([]byte("hello,world"))
    time.Sleep(time.Second * 5)
}
