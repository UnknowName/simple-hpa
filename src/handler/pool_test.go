package handler

import (
    "simple-hpa/src/scale"
    "simple-hpa/src/utils"
    "testing"
    "time"
)

func TestNewPoolHandler(t *testing.T) {
    config := utils.NewConfig("F:\\GoCodes\\simple-hpa\\config.yml")
    client := scale.NewK8SClient()
    pool := NewPoolHandler(config, client)
    pool.Execute([]byte("hello,world"))
    time.Sleep(time.Second * 5)
}
