package utils

import (
	"fmt"
	"log"
	"net"
)

type ForwardType uint8

const (
	udp    = "udp"
	// tcp    = "tcp"
	tryMax = 5
)

type Forwarder interface {
	Send(data []byte, cnt int)
}

func NewForward(configs []ForwardConfig) *Forward {
	forwards := make([]Forwarder, 0)
	for _, config := range configs {
		forward := newForwarder(config.TypeName, config.Address)
		if forward == nil {
			continue
		}
		forwards = append(forwards, forward)
	}
	return &Forward{forwards: forwards}
}

func newForwarder(typeName, ipAddr string) Forwarder {
	switch typeName {
	case "rsyslog":
		addr, err := net.ResolveUDPAddr(udp, ipAddr)
		if err != nil {
			log.Fatalln(err)
		}
		conn, err := net.DialUDP(udp, nil, addr)
		if err != nil {
			log.Fatalln(err)
		}
		return &RsyslogForward{conn: conn}
	}
	log.Println("WARN un supported forward type ", typeName, "Skip it")
	return nil
}

type Forward struct {
	forwards []Forwarder
}

func (f *Forward) Send(data []byte) {
	for _, forward := range f.forwards {
		go forward.Send(data, 1)
	}
}

type RsyslogForward struct {
	conn *net.UDPConn
}

func (rf *RsyslogForward) String() string {
	return fmt.Sprintf("RsyslogForward{}")
}

func (rf *RsyslogForward) Send(data []byte, cnt int) {
	n, err := rf.conn.Write(data)
	if len(data) != n || err != nil {
		log.Printf("%s send failed %d, try again", string(data), cnt)
		if cnt < tryMax {
			rf.Send(data, cnt+1)
		}
	}
}
