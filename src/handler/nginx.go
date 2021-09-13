package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/opentracing/opentracing-go"
	"log"
	"simple-hpa/src/ingress"
)

type nginxDataHandler struct {
	// 日志切割关键字
	logKey      []byte
	ingressType IngressType
}

func (ndh *nginxDataHandler) parseData(data []byte, services []string, filter FilterFunc, context context.Context) <-chan ingress.Access {
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
		if err := json.Unmarshal(bytes.ReplaceAll(jsonByte, []byte("\\x"), []byte("")), accessItem); err != nil {
			log.Println("To json failed ", err, "origin string:", string(jsonByte))
			return
		}
		span.LogKV("parseData", "json unmarshal complete")
		if accessItem.ServiceName() == "." {
			return
		}
		cha := filter(accessItem, services, ctx)
		if cha != nil {
			channel <- accessItem
		}
	}()
	return channel
}


func (ndh *nginxDataHandler) parseDataNew(data []byte, services []string) ingress.Access {
	byteStrings := bytes.Split(data, ndh.logKey)
	if len(byteStrings) != 2 {
		log.Println("Not NGINX Ingress data, origin string is:", string(data))
		return nil
	}
	jsonByte := byteStrings[1]
	if !bytes.HasPrefix(jsonByte, []byte("{")) {
		return nil
	}
	accessItem := new(ingress.NGINXAccess)
	// JSON化之前，去掉URL里面的中文\x
	if err := json.Unmarshal(bytes.ReplaceAll(jsonByte, []byte("\\x"), []byte("")), accessItem); err != nil {
		log.Println("To json failed ", err, "origin string:", string(jsonByte))
		return nil
	}
	for _, service := range services {
		if service == accessItem.ServiceName() {
			return accessItem
		}
	}
	return nil
}