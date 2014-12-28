// Copyright 2014 by Thorsten von Eicken, see LICENSE in top-level directory
// Lots of stuff from https://github.com/jcw/jeebus/blob/master/gadgets/database/database.go

package database

import (
	"fmt"
	"sync"

	"github.com/dmcgowan/go/codec"
	"github.com/golang/glog"
	"github.com/syndtr/goleveldb/leveldb"
	dbutil "github.com/syndtr/goleveldb/leveldb/util"
	"github.com/tve/widuino/gears"
)

type DB struct {
	ldb  *leveldb.DB
	path string
	// rfmessages can have a list of subscribers
	rfSubscriberMutex sync.Mutex
	rfSubscribers     []chan gears.RFMessage
	rfSubscriberStart []int64
	// each sensor can have a list of subscribers
	sensorSubscriberMutex sync.Mutex
	sensorSubscribers     map[string]*[]chan gears.SensorDataValue
	sensorSubscriberStart map[string]*[]int64
}

func Open(path string) (*DB, error) {
	glog.Infof("Opening LevelDB in %s", path)
	ldb, err := leveldb.OpenFile(path, nil)
	if err != nil {
		return nil, fmt.Errorf("database.Open %s: %s", path, err.Error())
	}
	return &DB{
			ldb, path,
			sync.Mutex{}, make([]chan gears.RFMessage, 0), make([]int64, 0),
			sync.Mutex{}, make(map[string]*[]chan gears.SensorDataValue),
			make(map[string]*[]int64)},
		nil
}

func (db *DB) Close() {
	db.ldb.Close()
}

var ErrNotFound = fmt.Errorf("key not found")

var mh = codec.MsgpackHandle{}

// Get performs a key lookup in the store and returns the value. It handles decoding
// the value into the interface provided using msgPack.
func (db *DB) Get(key string, value interface{}) error {
	glog.V(3).Infoln("get", key)
	data, err := db.ldb.Get([]byte(key), nil)
	if err == leveldb.ErrNotFound {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("database.Get key %s: %s", key, err.Error())
	}
	d := codec.NewDecoderBytes(data, &mh)
	err = d.Decode(value)
	if err != nil {
		return fmt.Errorf("database.Get decoding for key %s: %s", key, err.Error())
	}
	return nil
}

// Puts a new value into the database. Handles encoding using msgPack. Putting nil deletes
// an existing key-value pair
func (db *DB) Put(key string, value interface{}) error {
	glog.V(2).Infoln("put", key, value)
	if value != nil {
		var data []byte
		e := codec.NewEncoderBytes(&data, &mh)
		err := e.Encode(value)
		if err != nil {
			return fmt.Errorf("database.Put encoding for key %s: %s", key, err.Error())
		}
		err = db.ldb.Put([]byte(key), data, nil)
		if err != nil {
			return fmt.Errorf("database.Put key %s: %s", key, err.Error())
		}
	} else {
		db.ldb.Delete([]byte(key), nil)
	}
	return nil
}

// Iterate over a key range, start inclusive, end exclusive. The value is always placed into
// the 'value' interface in order to allow typing. If fun returns an error the iteration is
// aborted.
func (db *DB) Iterate(from, to string, value interface{}, fun func(key string) error) error {
	slice := &dbutil.Range{[]byte(from), []byte(to)}
	if len(to) == 0 {
		slice.Limit = append(slice.Start, 0xFF)
	}

	glog.V(4).Infof("Iterating from %s to %s", string(slice.Start), string(slice.Limit))
	iter := db.ldb.NewIterator(slice, nil)
	defer iter.Release()

	for iter.Next() {
		glog.V(4).Infof("  Iter: %s=>%#v", string(iter.Key()), iter.Value())
		d := codec.NewDecoderBytes(iter.Value(), &mh)
		err := d.Decode(value)
		if err != nil {
			return err
		}
		err = fun(string(iter.Key()))
		if err != nil {
			return err
		}
	}
	glog.V(4).Infof("Done iterating from %s to %s", string(slice.Start), string(slice.Limit))
	return nil
}

/*
func dbKeys(prefix string) (results []string) {
	glog.V(3).Infoln("keys", prefix)
	// TODO: decide whether this key logic is the most useful & least confusing
	// TODO: should use skips and reverse iterators once the db gets larger!
	skip := len(prefix)
	prev := "/" // impossible value, this never matches actual results

	dbIterateOverKeys(prefix, "", func(k string, v []byte) {
		i := strings.IndexRune(k[skip:], '/') + skip
		if i < skip {
			i = len(k)
		}
		if prev != k[skip:i] {
			// need to make a copy of the key, since it's owned by iter
			prev = k[skip:i]
			results = append(results, string(prev))
		}
	})
	return
}
*/
