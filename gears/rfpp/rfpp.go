// Copyright 2014 by Thorsten von Eicken, see LICENSE in top-level directory

package main

import (
	"fmt"
	"log"
	"time"

	"github.com/tve/widuino/gears"
)

func main() {
	log.Printf("Opening libchan connection to localhost:9323")
	gc, err := gears.Dial("localhost:9323")
	if err != nil {
		log.Fatal(err)
	}

	start := (time.Now().Unix() - 10*60) * 1000
	rfChan := gc.RFSubscribe(start)

	for m := range rfChan {
		ts := time.Unix(m.At/1000, (m.At%1000)*1000000).Format("2006-01-02 15:04:05.999")
		fmt.Printf("%-23s %-12s: %s\n", ts, m.RfTag(), RFFormat(m))
	}
}
