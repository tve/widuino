// Copyright 2014 by Thorsten von Eicken, see LICENSE in top-level directory
// Lots of stuff from https://github.com/jcw/jeebus/blob/master/gadgets/database/database.go

package database

import (
	"fmt"

	"github.com/tve/widuino/gears"
)

/*
// ===== Metrics =====

const metricsPrefix = "metric"

func (db *DB) PutMetric(name string, m gears.Metric) error {
	key := genMetricKey(name)
	return db.Put(key, m)
}

func (db *DB) GetMetric(name string) (gears.Metric, error) {
	key := genMetricKey(name)
	var m gears.Metric
	err := db.Get(key, &m)
	return m, err
}

func genMetricKey(name string) string {
	return fmt.Sprintf("%s/%s", metricsPrefix, name)
}

func parseMetricKey(key string) (name string) {
	fmt.Scanf(metricsPrefix+"/%s", &name)
	return
}
*/

// ===== SensorInfo =====

const sensInfoPrefix = "sensinfo"

func (db *DB) PutSensorInfo(name string, m gears.SensorInfo) error {
	key := genSensorInfoKey(name)
	return db.Put(key, m)
}

func (db *DB) GetSensorInfo(name string) (gears.SensorInfo, error) {
	key := genSensorInfoKey(name)
	var m gears.SensorInfo
	err := db.Get(key, &m)
	return m, err
}

func genSensorInfoKey(name string) string {
	return fmt.Sprintf("%s/%s", sensInfoPrefix, name)
}

func parseSensorInfoKey(key string) (name string) {
	fmt.Scanf(sensInfoPrefix+"/%s", &name)
	return
}
