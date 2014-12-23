// Copyright 2014 by Thorsten von Eicken, see LICENSE in top-level directory
// Lots of stuff from https://github.com/jcw/jeebus/blob/master/gadgets/database/database.go

package database

import (
	"github.com/golang/glog"
	"github.com/tve/widuino/gears"
)

// Most of the logic here is a duplication of rf_subscribe. Sadly, Go doesn't have generics
// and factoring out the inner portions of the logic so the outer structure can be shared
// makes it all but unreadable.

// Subscribe to Sensor messages starting at the timestamp given by start in milliseconds
// since the epoch, returns a channel to read messages from.
func (db *DB) SensorSubscribe(name string, start int64) chan gears.SensorDataValue {
	c := make(chan gears.SensorDataValue, 100)
	go db.catchUpSensorSubscribe(name, start, c)
	return c
}

// Add a sensor subscriber assuming the subscriber mutex is already held and unlock it when done
func (db *DB) addSensorSubscriber(name string, start int64, c chan gears.SensorDataValue) {
	defer db.sensorSubscriberMutex.Unlock()
	// allocate subscriber arrays for this sensor if there are none
	if _, ok := db.sensorSubscribers[name]; !ok {
		s := make([]chan gears.SensorDataValue, 0)
		db.sensorSubscribers[name] = &s
		i := make([]int64, 0)
		db.sensorSubscriberStart[name] = &i
	}
	// append subscriber
	subs := db.sensorSubscribers[name]
	subStarts := db.sensorSubscriberStart[name]
	*subs = append(*subs, c)
	*subStarts = append(*subStarts, start)
	// sanity check length of subscriber arrays
	if len(*subs) != len(*subStarts) {
		glog.Fatalf("sensorSubscriber array mismatch %d != %d", len(*subs), len(*subStarts))
	}
}

// returns true if found
func (db *DB) removeSensorSubscriber(name string, c chan gears.SensorDataValue) (found bool) {
	found = false
	subs, _ := db.sensorSubscribers[name]
	if subs == nil {
		return
	}
	for i := 0; i < len(*subs); i += 1 {
		if (*subs)[i] != c {
			continue
		}
		close((*subs)[i])
		found = true
		if i+1 == len(*subs) {
			*subs = (*subs)[0:i]
			*db.sensorSubscriberStart[name] = (*db.sensorSubscriberStart[name])[0:i]
		} else {
			*subs = append((*subs)[0:i], (*subs)[i+1:]...)
			*db.sensorSubscriberStart[name] = append(
				(*db.sensorSubscriberStart[name])[0:i],
				(*db.sensorSubscriberStart[name])[i+1:]...)
		}
	}
	if len(*subs) != len(db.sensorSubscriberStart) {
		glog.Fatalf("sensorSubscriber array mismatch %d != %d",
			len(*subs), len(db.sensorSubscriberStart))
	}
	return
}

// Unsubscribe the given channel from sensor messages. This will (asynchronously) cause the
// channel to be closed.
func (db *DB) SensorUnsubscribe(c chan gears.SensorDataValue) {
	db.sensorSubscriberMutex.Lock()
	defer db.sensorSubscriberMutex.Unlock()
	for n := range db.sensorSubscribers {
		if db.removeSensorSubscriber(n, c) {
			break
		}
	}
}

// Push a sensor message, which stores it in the database and forwards it to all sensorSubscribers.
func (db *DB) SensorPublish(name string, m gears.SensorDataValue) {
	db.sensorSubscriberMutex.Lock()
	defer db.sensorSubscriberMutex.Unlock()
	for i := range *db.sensorSubscribers[name] {
		if m.At >= (*db.sensorSubscriberStart[name])[i] {
			(*db.sensorSubscribers[name])[i] <- m
		}
	}
	glog.V(2).Infof("Published: %d to %d sensorSubscribers", m.At, len(db.sensorSubscribers))
}

// catch-up on old messages from the database and then switch atomically into
// a subscription
func (db *DB) catchUpSensorSubscribe(name string, start int64, c chan gears.SensorDataValue) {

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
		db.SensorIterate(name, start, 0, func(m gears.SensorDataValue) error {
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
					db.sensorSubscriberMutex.Unlock()
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
		db.sensorSubscriberMutex.Lock()
		lastAt, locked = doCatchUp(lastAt)
		//fmt.Printf("doCatchup -> %d %t\n", lastAt, locked)
		if locked {
			db.addSensorSubscriber(name, lastAt+1, c)
			glog.V(2).Infof("Subscriber %v now caught up", c)
			return
		}
		lastAt += 1
	}
}
