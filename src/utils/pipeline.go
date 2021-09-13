package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/opentracing/opentracing-go"
	"log"
	"simple-hpa/src/ingress"
)

func ParseUDPData(data []byte) <-chan ingress.Access {
	channel := make(chan ingress.Access)
	go func() {
		defer close(channel)
		byteStrings := bytes.Split(data, []byte("nginx: "))
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
		if err := json.Unmarshal(bytes.ReplaceAll(jsonByte, []byte("\\x"), []byte("")), accessItem); err != nil {
			log.Println("To json failed ", err, "origin string:", string(jsonByte))
			return
		}
		if accessItem.ServiceName() == "." {
			return
		}
		channel <- accessItem
	}()
	return channel
}

func FilterService(itemChan ingress.Access, services []string, parent context.Context) ingress.Access {
	span, ctx := opentracing.StartSpanFromContext(parent, "filterService")
	defer ctx.Done()
	span.LogKV("filterService", "start")
	span.LogKV("filterService", "go func")
	for _, service := range services {
		if itemChan.ServiceName() == service {
			span.LogKV("filterService", "data.ServiceName() == service")
			return itemChan
			span.LogKV("filterService", "complete ...")
		}
	}
	return nil
}
