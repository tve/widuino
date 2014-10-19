// Copyright 2014 by Thorsten von Eicken, see LICENSE in top-level directory
// Based on code Copyright 2014 by Jean-Claude Wippler: github.com/jcw/jeeboot

// wiboot: booter for widuino - JeeBoot compatible boot loader

package main

import (
	"flag"
	"fmt"
	"io/ioutil"

	"github.com/jcw/flow"
	_ "github.com/jcw/flow/gadgets"
	_ "github.com/jcw/housemon/gadgets/rfdata"
	_ "github.com/jcw/jeebus/gadgets/serial"
	_ "github.com/tve/widuino/gadgets"
)

const Version = "0.1.0"

var (
	showInfo = flag.Bool("i", false,
		"display some information about this tool")
	udpPort = flag.String("port", "9999",
		"UDP port to listen to UDP/RF gateway")
	freqBand = flag.Int("band", 868,
		"frequency band used to listen for incoming JeeBoot requests")
	netGroup = flag.Int("group", 212,
		"net group used to listen for incoming JeeBoot requests")
	configFile = flag.String("config", "config.json",
		"configuration file containing the swid/hwid details")
)

func main() {
	flag.Parse()

	if *showInfo {
		fmt.Println("Wiboot", Version, "+ Flow", flow.Version)
		return
	}

	// load configuration from file
	config, err := ioutil.ReadFile(*configFile)
	if err != nil {
		panic(err)
	}

	// main processing pipeline: serial, rf12demo, jeeboot, serial
	// firmware: jeeboot, readtext, intelhex, binaryfill, calccrc, bootdata
	// other valid packets are routed to the bootServer gadget

	c := flow.NewCircuit()
	c.Add("udp", "UdpGW")
	c.Add("sk", "Sink")
	c.Add("jb", "JeeBoot")
	c.Add("rd", "ReadTextFile")
	c.Add("hx", "IntelHexToBin")
	c.Add("bf", "BinaryFill")
	c.Add("cs", "CalcCrc16")
	c.Add("bd", "BootData")
        // Nodes to connect nodes to jeeboot
	c.Connect("udp.Recv", "sk.In", 0)   // throw away data messages coming in
	c.Connect("udp.Oob", "jb.In", 0)    // out-of band messages -> boot protocol
	c.Connect("udp.Rej", "sk.In", 0)    // throw away rejected serial port msgs
	c.Connect("jb.Out", "udp.Xmit", 0)  // path back out from JeepBoot to nodes
        // Pipeline to read-in sketches and get them in a format for sending
	c.Connect("jb.Files", "rd.In", 0)
	c.Connect("rd.Out", "hx.In", 0)
	c.Connect("hx.Out", "bf.In", 0)
	c.Connect("bf.Out", "cs.In", 0)
	c.Connect("cs.Out", "bd.In", 0)
	c.Feed("udp.Port", *udpPort)
	c.Feed("jb.Cfg", config)
	c.Feed("bf.Len", 64)
	c.Run()
}
