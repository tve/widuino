// Copyright 2014 by Thorsten von Eicken, see LICENSE in top-level directory

// Log writer - write all received messages to log files to be able to analyze and reprocess
// them at will. The logs have the format:
// YYYY-MM-DD HH:MM:SS GG II UU LL: 00 11 22 ..
// Where the first two "words" are the local time, GGG is the network group in hex,
// II is the node id in decimal, LL is the packet payload length excl. module id in decimal,
// UU is the module id (first payload byte) in hex, 00 11 etc are payload bytes after
// the module id (there should be LL-1 bytes).

package main

import (
	"./database"
	"fmt"
	"os"
	"time"

	"github.com/golang/glog"
)

const datalog = "./_log"

func init() {
	RegisterRecvProcessor(LogProcessor)
}

func LogProcessor(in chan database.RFMessage) {
	go func() {
		var log_fd *os.File
		log_name := ""
		for m := range in {
			// Format the string
			t := time.Now().Format("2006-01-02 15:04:05")
			str := fmt.Sprintf("%s %02x %02d %02x %02d: % x\n",
				t, m.Group, m.Node, m.Kind, len(m.Data)+1, m.Data)
			// Open log file if necessary
			name := fmt.Sprintf("%s/%s.wd", datalog, time.Now().Format("2006-01-02"))
			if name != log_name {
				if log_fd != nil {
					log_fd.Close()
				}
				log_name = name
				var err error
				log_fd, err = os.OpenFile(log_name,
					os.O_WRONLY+os.O_APPEND+os.O_CREATE, 0664)
				if err != nil {
					glog.Errorf("Cannot open %s: %s", log_name, err.Error())
					log_name = ""
				}
			}
			// Write data
			_, err := log_fd.WriteString(str)
			if err != nil {
				glog.Errorf("Error writing %s: %s", log_name, err.Error())
				log_name = ""
			}
		}
	}()
}
