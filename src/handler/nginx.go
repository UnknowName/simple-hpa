package handler

import (
	"bytes"
	"log"

	"simple-hpa/src/ingress"
)


type nginxDataHandler struct {
	// 日志切割关键字
	logKey      []byte
	ingressType IngressType
	autoService map[string]struct{}
}

func (ndh *nginxDataHandler) SetAutoService(services []string) {
	for _, service := range services {
		ndh.autoService[service] = struct{}{}
	}
}

func (ndh *nginxDataHandler) ParseData(data []byte) ingress.Access {
	byteStrings := bytes.Split(data, ndh.logKey)
	if len(byteStrings) != 2 {
		log.Println("Not NGINX Ingress data, origin string is:", string(data))
		return nil
	}
	// JSON化之前，去掉URL里面的中文\x
	jsonByte := bytes.ReplaceAll(byteStrings[1], []byte("\\x"), []byte(""))
	accessItem := new(ingress.NGINXAccess)
	// err := Unmarshal(jsonByte, accessItem)
	err := ConcurUnmarshal(jsonByte, accessItem)
	if err != nil {
		log.Println("json failed", err)
		return nil
	}
	// 原始数据有问题，非json化数据
	if _, ok := ndh.autoService[accessItem.ServiceName()]; ok {
		return accessItem
	}
	return nil
}