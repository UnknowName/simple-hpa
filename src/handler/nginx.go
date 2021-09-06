package handler

import (
    "bytes"
    "encoding/json"
    "log"
    "simple-hpa/src/ingress"
)


type nginxDataHandler struct {
    // 日志切割关键字
    logKey      []byte
    ingressType IngressType
}

func (ndh *nginxDataHandler) parseData(data []byte) <-chan ingress.Access {
    channel := make(chan ingress.Access)
    go func() {
        defer close(channel)
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