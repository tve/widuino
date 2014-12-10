// Copyright 2014 by Thorsten von Eicken, see LICENSE in top-level directory
// Lots of stuff from https://github.com/jcw/jeebus/blob/master/gadgets/database/database.go

package database

import (
	"github.com/golang/glog"
	"github.com/tve/widuino/gears"
)

/*
func (db *DB) subscribeAt(start int64) chan RFMessage {
	db.subscriberMutex.Lock()
	defer db.subscriberMutex.Unlock()
	c := make(chan RFMessage, 100)
	db.subscribers = append(db.subscribers, c)
	db.subscriberStart[len(db.subscribers)-1] = start
	return c
}
*/

func (db *DB) Subscribe(start int64) chan gears.RFMessage {
	c := make(chan gears.RFMessage, 100)
	go db.catchUpSubscribe(start, c)
	return c
}

// catch-up on old messages from the database and then switch atomically into
// a subscription
func (db *DB) catchUpSubscribe(start int64, c chan gears.RFMessage) {

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
					db.subscriberMutex.Unlock()
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
		db.subscriberMutex.Lock()
		lastAt, locked = doCatchUp(lastAt)
		//fmt.Printf("doCatchup -> %d %t\n", lastAt, locked)
		if locked {
			defer db.subscriberMutex.Unlock()
			db.subscribers = append(db.subscribers, c)
			db.subscriberStart = append(db.subscriberStart, lastAt+1)
			if len(db.subscribers) != len(db.subscriberStart) {
				glog.Fatalf("subscriber array mismatch %d != %d",
					len(db.subscribers), len(db.subscriberStart))
			}
			glog.V(2).Infof("Subscriber %v now caught up", c)
			return
		}
		lastAt += 1
	}
}

func (db *DB) Unsubscribe(c chan gears.RFMessage) {
	db.subscriberMutex.Lock()
	defer db.subscriberMutex.Unlock()
	for i := 0; i < len(db.subscribers); i += 1 {
		if db.subscribers[i] != c {
			continue
		}
		close(db.subscribers[i])
		if i+1 == len(db.subscribers) {
			db.subscribers = db.subscribers[0:i]
			db.subscriberStart = db.subscriberStart[0:i]
		} else {
			db.subscribers = append(db.subscribers[0:i], db.subscribers[i+1:]...)
			db.subscriberStart = append(db.subscriberStart[0:i], db.subscriberStart[i+1:]...)
		}
	}
	if len(db.subscribers) != len(db.subscriberStart) {
		glog.Fatalf("subscriber array mismatch %d != %d",
			len(db.subscribers), len(db.subscriberStart))
	}
}

func (db *DB) Publish(m gears.RFMessage) {
	db.subscriberMutex.Lock()
	defer db.subscriberMutex.Unlock()
	for i := range db.subscribers {
		if m.At >= db.subscriberStart[i] {
			db.subscribers[i] <- m
		}
	}
	glog.V(2).Infof("Published: %d to %d subscribers", m.At, len(db.subscribers))
}
