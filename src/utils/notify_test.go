package utils

import (
    "testing"
    "time"
)

func TestNewSender(t *testing.T) {
    sender := NewSender("dding", "d9631ffb98254665dc5033fbd73d5cfee06558fc522f80bfd83cdfeb9da168df")
    sender.Send("test msg")
    time.Sleep(time.Second * 10)
}
