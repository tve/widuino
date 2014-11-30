// Copyright 2014 by Thorsten von Eicken, see LICENSE in top-level directory
// Lots of stuff from https://github.com/jcw/jeebus/blob/master/gadgets/database/database.go

package database

import (
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/golang/glog"
)

type RFMessage struct {
	At    int64  // milliseconds since unix epoch
	Group byte   // RF network group
	Node  byte   // RF node ID (Node=0 -> bcast)
	DoAck bool   // Send with ack requested
	Kind  byte   // Message payload type ("module" numbers)
	Data  []byte // Message payload
	//Xtra map[string]interface{} // Extra info added during processing
}

func (m RFMessage) RfTag() string {
	return fmt.Sprintf("RFg%di%dk%d", m.Group, m.Node, m.Kind)
}

const prefix = "raw/"

func NewProcessor(db *DB) func(chan RFMessage) {
	return func(in chan RFMessage) {
		go func() {
			for m := range in {
				err := db.PutRFMessage(m)
				if err != nil {
					glog.Errorf("Error writing database: %s", err.Error())
				}
			}
		}()
	}
}

func (db *DB) PutRFMessage(m RFMessage) error {
	if m.At == 0 {
		// Add the time in milliseconds since the epoch
		m.At = time.Now().UnixNano() / 1000000
	}
	// Form the key
	key := genRFKey(m.At)
	// Write data
	err := db.Put(key, m)
	if err != nil {
		return err
	}
	// Publish to subscribers
	db.Publish(m)
	return nil
}

func genRFKey(at int64) string {
	return fmt.Sprintf("%s%013d", prefix, at)
}

func parseRFKey(str string) (int64, error) {
	if len(str) <= len(prefix) {
		return 0, fmt.Errorf("bad RFMessage key: %s", str)
	}
	return strconv.ParseInt(str[len(prefix):], 10, 64)
}

func (db *DB) RFIterate(start, end int64, handle func(m RFMessage) error) error {
	startKey := genRFKey(start)
	endKey := genRFKey(math.MaxInt64)
	if end > 0 {
		endKey = genRFKey(end)
	}
	var m RFMessage
	return db.Iterate(startKey, endKey, &m, func(key string) error {
		return handle(m)
	})
}
