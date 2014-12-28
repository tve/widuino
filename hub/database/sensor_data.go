// Copyright 2014 by Thorsten von Eicken, see LICENSE in top-level directory
// Lots of stuff from https://github.com/jcw/jeebus/blob/master/gadgets/database/database.go

package database

import (
	"fmt"
	"math"
	"time"

	"github.com/golang/glog"
	"github.com/tve/widuino/gears"
)

const sensorPrefix = "sens/"

func (db *DB) PutSensorValue(name string, m gears.SensorDataValue) error {
	if m.At == 0 {
		// Add the time in milliseconds since the epoch
		m.At = time.Now().UnixNano() / 1000000
	}
	glog.V(2).Infof("Put: %d %+v", m.At, m)
	// Form the key
	key := genSensorKey(name, m.At)
	// Write data
	err := db.Put(key, m)
	if err != nil {
		return err
	}
	// Publish to subscribers
	db.SensorPublish(name, m)
	return nil
}

func genSensorKey(name string, at int64) string {
	return fmt.Sprintf("%s%s/%013d", sensorPrefix, name, at)
}

// Parse a sensor key into name, timestamp
func parseSensorKey(str string) (name string, at int64, err error) {
	_, err = fmt.Scanf(str, sensorPrefix+"%s/%d", &name, &at)
	return
}

func (db *DB) SensorIterate(name string, start, end int64,
	handle func(m gears.SensorDataValue) error) error {
	startKey := genSensorKey(name, start)
	endKey := genSensorKey(name, math.MaxInt64)
	if end > 0 {
		endKey = genSensorKey(name, end)
	}
	var m gears.SensorDataValue
	return db.Iterate(startKey, endKey, &m, func(key string) error {
		err := handle(m)
		//m = gears.SensorDataValue{} // not necessary, doesn't have any pointers
		return err
	})
}
