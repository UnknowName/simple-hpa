package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log"
	"sync"
	"sync/atomic"

	"github.com/opentracing/opentracing-go"

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

func (ndh *nginxDataHandler) parseDataWithFilter(data []byte, services []string) ingress.Access {
	byteStrings := bytes.Split(data, ndh.logKey)
	if len(byteStrings) != 2 {
		log.Println("Not NGINX Ingress data, origin string is:", string(data))
		return nil
	}
	// JSON化之前，去掉URL里面的中文\x
	jsonByte := bytes.ReplaceAll(byteStrings[1], []byte("\\x"), []byte(""))
	accessItem := new(ingress.NGINXAccess)
	err := Unmarshal(jsonByte, accessItem)
	if err != nil {
		log.Println("json failed", err)
		return nil
	}
	// 原始数据有问题，非json化数据
	for _, service := range services {
		if service == accessItem.ServiceName() {
			return accessItem
		}
	}
	return nil
}

func Unmarshal(data []byte, o ingress.Access) error {
	wg := sync.WaitGroup{}
	wg.Add(3)
	var cnt uint32
	go func() {
		defer wg.Done()
		err := json.Unmarshal(data, o)
		if err != nil {
			atomic.AddUint32(&cnt, 1)
		}
	}()

	go func() {
		defer wg.Done()
		err := json.Unmarshal(append(data, '}'), o)
		if err != nil {
			atomic.AddUint32(&cnt, 1)
		}
	}()

	go func() {
		defer wg.Done()
		err := json.Unmarshal(data[:len(data)-1], o)
		if err != nil {
			atomic.AddUint32(&cnt, 1)
		}
	}()
	wg.Wait()
	if cnt == 3 {
		return errors.New(string(data))
	}
	return nil
}