package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"path"
	"simple-hpa/src/handler"
	"simple-hpa/src/scale"
	"simple-hpa/src/utils"
	"syscall"
)

const (
	netType = "udp"
	bufSize = 1024
)

var (
	config    *utils.Config
	server    *Server
	k8sClient *scale.K8SClient
)

type Server struct {
	configPath  string
	serviceName string
	tracerURL   string
}

func init() {
	server = new(Server)
	flag.StringVar(&server.configPath, "config", "./config.yaml", "config path ...")
	flag.StringVar(&server.serviceName, "svc", "simple-hpa", "simple service name")
	flag.StringVar(&server.tracerURL, "trace", "jaeger.jaeger-infra:5775", "trace url")
	flag.Parse()
	pwd, _ := os.Getwd()
	cfg := path.Join(pwd, server.configPath)
	log.SetFlags(log.Ldate | log.Lmicroseconds | log.Llongfile)
	config = utils.NewConfig(cfg)
	if config.AutoScale.Services == nil || len(config.AutoScale.Services) == 0 {
		log.Fatalln("WARNING, Auto scale dest service not defined")
	}
}

func main() {
	// tracerService, closer := tracer.New(server.serviceName, server.tracerURL)
	// defer closer.Close()
	// opentracing.SetGlobalTracer(tracerService)
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
	log.Printf("Auto scale services: %s, these service pod count min=%d max=%d",
		config.AutoScale.Services,
		config.AutoScale.MinPod,
		config.AutoScale.MaxPod,
	)
	log.Printf("forward origin message to %s", config.Forwards)
	k8sClient = scale.NewK8SClient()
	poolHandler := handler.NewPoolHandler(config, k8sClient)
	forward := utils.NewForward(config.Forwards)
	quitChan := make(chan os.Signal)
	signal.Notify(quitChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGILL)
	go quit(quitChan, conn)
	for {
		buf := make([]byte, bufSize)
		n, err := conn.Read(buf)
		if err != nil && err == io.EOF {
			continue
		}
		if err != nil {
			log.Println("receive quit signal,quit...")
			return
		}
		forward.Send(buf)
		poolHandler.Execute(buf[:n])
	}
}

func quit(c chan os.Signal, conn *net.UDPConn)  {
	select {
	case <- c:
		conn.Close()
		return
	}
}