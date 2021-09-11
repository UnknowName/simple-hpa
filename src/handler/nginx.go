package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/opentracing/opentracing-go"
	"log"
	"simple-hpa/src/ingress"
	"time"
)

type nginxDataHandler struct {
	// 日志切割关键字
	logKey      []byte
	ingressType IngressType
}

func (ndh *nginxDataHandler) parseData(data []byte, context context.Context) <-chan ingress.Access {
	span, ctx := opentracing.StartSpanFromContext(context, "parseData")
	span.LogKV("parseData", "start")
	channel := make(chan ingress.Access)
	go func() {
		defer func() {
			ctx.Done()
			span.Finish()
		}()
		defer close(channel)
		span.LogKV("parseData", "go func")
		byteStrings := bytes.Split(data, ndh.logKey)
		if len(byteStrings) != 2 {
			log.Println("Not NGINX Ingress data, origin string is:", string(data))
			return
		}
		jsonByte := byteStrings[1]
		if !bytes.HasPrefix(jsonByte, []byte("{")) {
			return
		}
		accessItem := new(ingress.NGINXAccess)
		// JSON化之前，去掉URL里面的中文\x
		span.LogKV("parseData", "json unmarshal start")
		time1 := time.Now().Unix()
		if err := json.Unmarshal(bytes.ReplaceAll(jsonByte, []byte("\\x"), []byte("")), accessItem); err != nil {
			log.Println("To json failed ", err, "origin string:", string(jsonByte))
			return
		}
		span.LogKV("parseData", "json unmarshal complete")
		time2 := time.Now().Unix()
		log.Printf("time: %d \n", time2-time1)
		if accessItem.ServiceName() == "." {
			return
		}
		channel <- accessItem
	}()
	return channel
}
