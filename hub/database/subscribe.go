// Copyright 2014 by Thorsten von Eicken, see LICENSE in top-level directory
// Lots of stuff from https://github.com/jcw/jeebus/blob/master/gadgets/database/database.go

package database

import "github.com/golang/glog"

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

func (db *DB) Subscribe(start int64) chan RFMessage {
	c := make(chan RFMessage, 100)
	go db.catchUpSubscribe(start, c)
	return c
}

// catch-up on old messages from the database and then switch atomically into
// a subscription
func (db *DB) catchUpSubscribe(start int64, c chan RFMessage) {

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
		db.RFIterate(start, 0, func(m RFMessage) error {
			lastAt = m.At
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
			return
		}
		lastAt += 1
	}
}

func (db *DB) Unsubscribe(c chan RFMessage) {
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

func (db *DB) Publish(m RFMessage) {
	db.subscriberMutex.Lock()
	defer db.subscriberMutex.Unlock()
	for i := range db.subscribers {
		if m.At >= db.subscriberStart[i] {
			db.subscribers[i] <- m
		}
	}
}
