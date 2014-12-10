// Copyright 2014 by Thorsten von Eicken, see LICENSE in top-level directory
// Lots of stuff from https://github.com/jcw/jeebus/blob/master/gadgets/database/database.go

package database

import (
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/golang/glog"
	"github.com/tve/widuino/gears"
)

const prefix = "raw/"

func (db *DB) NewProcessor() func(chan gears.RFMessage) {
	return func(in chan gears.RFMessage) {
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

func (db *DB) PutRFMessage(m gears.RFMessage) error {
	if m.At == 0 {
		// Add the time in milliseconds since the epoch
		m.At = time.Now().UnixNano() / 1000000
	}
	glog.V(2).Infof("Put: %d %+v", m.At, m)
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

func (db *DB) RFIterate(start, end int64, handle func(m gears.RFMessage) error) error {
	startKey := genRFKey(start)
	endKey := genRFKey(math.MaxInt64)
	if end > 0 {
		endKey = genRFKey(end)
	}
	var m gears.RFMessage
	return db.Iterate(startKey, endKey, &m, func(key string) error {
		err := handle(m)
		m = gears.RFMessage{} // we wipe out m.Data in particular so it doesn't get reused
		return err
	})
}
