// Copyright 2014 by Thorsten von Eicken, see LICENSE in top-level directory

// Booter for widuino - JeeBoot compatible boot loader
// Listens to MQTT messages on topics /rf/<group_id>/<node_id>/rb and responds using the
// corresponding .../tx topics.

// TODO: this has lots of code overlap with udpgw.go, need to factor stuff out or use JeeBus
// if that stabilizes...

package main

import (
	"flag"
	"log"
	"net"
	"os"
	"sync"

	"github.com/golang/glog"
	"github.com/tve/widuino/gears"
	"github.com/tve/widuino/hub/database"
)

const dbPath = "_data"

var bootConfig = flag.String("bootConfig", "sketches.json", "config file for boot server")

// handle to (global) levelDB database
var db *database.DB

// received messages are broadcast to a set of receivers, each attached to a channel
var recvProcessors []chan gears.RFMessage
var processorsLock sync.Mutex // guard changes to recvProcessors array
// to transmit a message anyone can push into the xmit channel
var xmitChan chan gears.RFMessage

func RegisterRecvProcessor(f func(chan gears.RFMessage)) {
	if recvProcessors == nil {
		recvProcessors = make([]chan gears.RFMessage, 0)
	}
	processorsLock.Lock()
	defer processorsLock.Unlock()
	ch := make(chan gears.RFMessage, 10)
	recvProcessors = append(recvProcessors, ch)
	go f(ch)
}

//===== Main

func main() {
	flag.Parse()

	// open database
	var err error
	db, err = database.Open(dbPath)
	if err != nil {
		glog.Fatalf("Cannot open database %s: %s", dbPath, err.Error())
	}

	// register processors
	RegisterRecvProcessor(LogProcessor)
	RegisterRecvProcessor(db.NewProcessor())

	// start receiver mux - forwards to all recvProcessors
	recv := make(chan gears.RFMessage, 10)
	go func() {
		for m := range recv {
			for _, ch := range recvProcessors {
				ch <- m
			}
		}
	}()

	// allocate xmit channel with buffering to allow for retransmit delays
	xmitChan = make(chan gears.RFMessage, 100)

	listener, err := net.Listen("tcp", "localhost:9323")
	if err != nil {
		log.Fatal(err)
	}
	glog.Infof("Listening for libchan connections on port 9323")
	go ServeChan(listener)

	booter := NewBooter(*bootConfig)
	if booter == nil {
		os.Exit(1)
	}
	udpGw := &UDPGateway{Port: 9999, Recv: recv, Xmit: xmitChan, Boot: booter}
	udpGw.Run()

}
