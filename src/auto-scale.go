package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"path"
	"syscall"

	"auto-scale/src/handler"
	"auto-scale/src/utils"
)

const (
	netType = "udp"
	bufSize = 1024
)

var (
	buf       [bufSize]byte
	bufByte   bytes.Buffer
	config    *utils.Config
	configPath string
)

func init() {
	flag.StringVar(&configPath, "config", "config.yaml", "config path ...")
	flag.Parse()
	pwd, _ := os.Getwd()
	cfg := path.Join(pwd, configPath)
	// log.SetFlags(log.Ldate | log.Lmicroseconds | log.Llongfile)
	config = utils.NewConfig(cfg)
	if config.ScaleServices == nil || len(config.ScaleServices) == 0 {
		log.Fatalln("WARNING, Auto scale dest service not defined")
	}
	go func() {
		quitChan := make(chan os.Signal)
		signal.Notify(quitChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGILL)
		c := <-quitChan
		log.Println("received ", c)
		os.Exit(0)
	}()
}

func main() {
	listenAddr := fmt.Sprintf("%s:%d", config.Listen.ListenAddr, config.Listen.Port)
	addr, err := net.ResolveUDPAddr(netType, listenAddr)
	if err != nil {
		log.Fatalln("Resolve udp address error ", err)
		return
	}
	conn, err := net.ListenUDP(netType, addr)
	if err != nil {
		log.Fatalln("Listen on ", addr.IP, "failed ", err)
		return
	}
	defer conn.Close()
	log.Printf("App listen on %s/%s", listenAddr, netType)
	for _, conf := range config.ScaleServices {
		log.Printf("service %s.%s, safeQps=%.2f, maxQps=%.2f, minPod=%d, maxPod=%d factor=%.1f",
			conf.ServiceName, conf.Namespace, conf.SafeQps, conf.MaxQps, conf.MinPod, conf.MaxPod, conf.Factor)
	}
	log.Printf("forward origin message to %s", config.Forwards)
	poolHandler := handler.NewPoolHandler(config)
	forward := utils.NewForward(config.Forwards)
	for {
		n, err := conn.Read(buf[:])
		if err != nil && err == io.EOF {
			continue
		}
		if err != nil {
			log.Println(err)
			return
		}
		if n == bufSize {
			bufByte.Write(buf[:])
			continue
		}
		bufByte.Write(buf[:n])
		poolHandler.Execute(bufByte.Bytes())
		forward.Send(bufByte.Bytes())
		bufByte.Reset()
	}
}
