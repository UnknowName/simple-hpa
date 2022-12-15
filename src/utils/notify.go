package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	msgtype     = "text"
	Timeout     = 5
	contentType = "application/json"
	dBase       = "https://oapi.dingtalk.com/robot/send?access_token="
)

type Sender interface {
	Send(msg string)
}

func NewSender(name, token string) Sender {
	switch strings.ToLower(name) {
	case "dding":
		return &dDingSender{url: fmt.Sprintf("%s%s", dBase, token)}
	}
	return nil
}

func newTMessage(msg string, atMobiles []string, atAll bool) *TMessage {
	if atMobiles == nil {
		atMobiles = make([]string, 0)
	}
	atUsers := At{AtMobiles: atMobiles}
	text := Content{Content: msg}
	return &TMessage{
		MsgType: msgtype,
		Text:    text,
		At:      atUsers,
		IsAtAll: atAll,
	}
}

type TMessage struct {
	MsgType string  `json:"msgtype"`
	Text    Content `json:"text"`
	At      At      `json:"at"`
	IsAtAll bool    `json:"isAtAll"`
}

type Content struct {
	Content string `json:"content"`
}

type At struct {
	AtMobiles []string `json:"atMobiles"`
}

type dDingSender struct {
	url string
}

func (dd *dDingSender) Send(msg string) {
	_msg := newTMessage(msg, nil, false)
	_bytes, err := json.Marshal(_msg)
	if err != nil {
		log.Println("msg transition json error: ", err)
	}
	client := http.Client{Timeout: time.Second * Timeout}
	resp, err := client.Post(dd.url, contentType, bytes.NewBuffer(_bytes))
	if err != nil {
		log.Println(err)
		return
	}
	buf := make([]byte, 1024)
	n, _ := resp.Body.Read(buf)
	log.Println(string(buf[:n]))
}
