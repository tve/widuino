// Copyright 2014 by Thorsten von Eicken, see LICENSE in top-level directory
// Lots of stuff from https://github.com/jcw/jeebus/blob/master/gadgets/database/database.go

package database

import (
	"github.com/golang/glog"
	"github.com/tve/widuino/gears"
)

// Most of the logic here is a duplication of sensor_subscribe. Sadly, Go doesn't have generics
// and factoring out the inner portions of the logic so the outer structure can be shared
// makes it all but unreadable.

func (db *DB) RFSubscribe(start int64) chan gears.RFMessage {
	c := make(chan gears.RFMessage, 100)
	go db.catchUpRFSubscribe(start, c)
	return c
}

func (db *DB) RFUnsubscribe(c chan gears.RFMessage) {
	db.rfSubscriberMutex.Lock()
	defer db.rfSubscriberMutex.Unlock()
	for i := 0; i < len(db.rfSubscribers); i += 1 {
		if db.rfSubscribers[i] != c {
			continue
		}
		close(db.rfSubscribers[i])
		if i+1 == len(db.rfSubscribers) {
			db.rfSubscribers = db.rfSubscribers[0:i]
			db.rfSubscriberStart = db.rfSubscriberStart[0:i]
		} else {
			db.rfSubscribers = append(db.rfSubscribers[0:i], db.rfSubscribers[i+1:]...)
			db.rfSubscriberStart = append(db.rfSubscriberStart[0:i], db.rfSubscriberStart[i+1:]...)
		}
	}
	if len(db.rfSubscribers) != len(db.rfSubscriberStart) {
		glog.Fatalf("rfSubscriber array mismatch %d != %d",
			len(db.rfSubscribers), len(db.rfSubscriberStart))
	}
}

func (db *DB) RFPublish(m gears.RFMessage) {
	db.rfSubscriberMutex.Lock()
	defer db.rfSubscriberMutex.Unlock()
	for i := range db.rfSubscribers {
		if m.At >= db.rfSubscriberStart[i] {
			db.rfSubscribers[i] <- m
		}
	}
	glog.V(2).Infof("Published: %d to %d rfSubscribers", m.At, len(db.rfSubscribers))
}

// catch-up on old messages from the database and then switch atomically into
// a subscription
func (db *DB) catchUpRFSubscribe(start int64, c chan gears.RFMessage) {

	// replay messages from the database while holding the subscribers lock to
	// prevent anything from being published. Use non-blocking channel send
	// to detect when the channel is full and release the subscribers lock in
	// that case so the whole system isn't blocked. Return the timestamp of the
	// last message sent and whether the lock was released or not.
	// This relies on the channel having a reasonable capacity so we have
	// some chance of catching up.
	doCatchUp := func(start int64) (int64, bool) {
		// replay old events and keep track of last one
		var lastAt int64
		locked := true
		count := 0
		db.RFIterate(start, 0, func(m gears.RFMessage) error {
			count += 1
			lastAt = m.At
			//glog.V(2).Infof("Sending m=%x d=%x", &m, &(m.Data))
			select {
			case c <- m:
				// sent, good...
			default:
				// we're gonna block, release lock
				if locked {
					locked = false
					db.rfSubscriberMutex.Unlock()
				}
				c <- m // blocking send...
			}
			return nil
		})
		glog.V(2).Infof("Sent %d catch-up messages", count)
		return lastAt, locked
	}

	// acquire the subscribers lock, forward messages from the database,
	// and then create a subscription if the lock was held the whole time,
	// otherwise repeat...
	lastAt := start
	for {
		var locked bool
		db.rfSubscriberMutex.Lock()
		lastAt, locked = doCatchUp(lastAt)
		//fmt.Printf("doCatchup -> %d %t\n", lastAt, locked)
		if locked {
			defer db.rfSubscriberMutex.Unlock()
			db.rfSubscribers = append(db.rfSubscribers, c)
			db.rfSubscriberStart = append(db.rfSubscriberStart, lastAt+1)
			if len(db.rfSubscribers) != len(db.rfSubscriberStart) {
				glog.Fatalf("rfSubscriber array mismatch %d != %d",
					len(db.rfSubscribers), len(db.rfSubscriberStart))
			}
			glog.V(2).Infof("rfSubscriber %v now caught up", c)
			return
		}
		lastAt += 1
	}
}
