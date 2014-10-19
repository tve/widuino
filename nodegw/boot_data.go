// Copyright 2014 by Thorsten von Eicken, see LICENSE in top-level directory

// Boot file manager
package main

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"

	//"gopkg.in/fsnotify.v1"
	"github.com/golang/glog"
)

type software []byte

type booter struct {
	dir      string                 // directory with config and hex files
	nodeType map[string]pairingInfo // map HwId -> nodeType, groupId, nodeId
	sketch   map[uint16]string      // map NodeType -> .hex file
	software map[uint16]software    // map NodeType -> sketch hex data
}

var commentRe = regexp.MustCompile(`(?m)#.*$`)

func NewBooter(configFile string) *booter {
	b := booter{
		nodeType: make(map[string]pairingInfo),
		sketch:   make(map[uint16]string),
		software: make(map[uint16]software),
	}

	config, err := ioutil.ReadFile(configFile)
	if err != nil {
		glog.Errorf("Error reading config file %s: %s", configFile, err.Error())
		return nil
	}
	b.dir = path.Dir(configFile)

	// remove comments
	config = commentRe.ReplaceAllLiteral(config, []byte{})

	// parse json
	d := json.NewDecoder(bytes.NewReader(config))

	err = d.Decode(&b.nodeType)
	if err != nil {
		glog.Errorf("Error reading pairing from config: %s", err.Error())
		return nil
	}
	for i, v := range b.nodeType {
		glog.Infof("  node %s -> nodeType=%d RF%di%d", i, v[0], v[1], v[2])
	}

	var sketches map[string]string
	err = d.Decode(&sketches)
	if err != nil {
		glog.Errorf("Error reading sketches from config: %s", err.Error())
		return nil
	}
	b.sketch = make(map[uint16]string)
	for k, v := range sketches {
		if i, err := strconv.Atoi(k); err == nil {
			b.sketch[uint16(i)] = v
		}
	}
	for i, v := range b.sketch {
		glog.Infof("  nodeType=%d -> %s", i, v)
	}

	return &b
}

// ===== Filesystem change notifications

// ===== Read boot configuration

// ===== Read Intel hex files

func (b *booter) findSoftware(id uint16) (software, error) {
	sw, ok := b.software[id]
	glog.V(2).Infof("findSoftware for nodeType=%d", id)
	if ok {
		glog.V(2).Infof("  retrieving cached software for nodeType=%d", id)
		return sw, nil
	}

	// load it
	file, ok := b.sketch[id]
	if !ok {
		return []byte{}, fmt.Errorf("no sketch is configured")
	}
	fName := path.Join(b.dir, file)
	glog.V(2).Infof("  reading %s", fName)
	f, err := os.Open(fName)
	if err != nil {
		return []byte{}, fmt.Errorf("opening %s: %s", fName, err.Error())
	}
	sw, err = readHex(f)
	// add to cache
	if err == nil {
		glog.V(2).Infof("  saving software for nodeType=%d in cache (%d bytes)",
			id, len(sw))
		b.software[id] = sw
	}
	// return the info
	return sw, err
}

func readHex(rd io.Reader) ([]byte, error) {
	scanner := bufio.NewScanner(rd)
	i := 0
	sw := make([]byte, 0, 64*1024)
	for scanner.Scan() {
		l := scanner.Text()
		i += 1
		if strings.HasPrefix(l, ":") {
			b, err := hex.DecodeString(l[1:])
			if err != nil {
				return []byte{}, fmt.Errorf("line %d: %s", i, err.Error())
			}
			if len(b) < 5 {
				return []byte{}, fmt.Errorf("line %d: short line (%d<5)", i, len(b))
			}

			// verify checksum
			sum := int8(0)
			for _, v := range b {
				sum += int8(v)
			}
			if sum != 0 {
				return []byte{}, fmt.Errorf("line %d: bad checksum %d", i, sum)
			}

			// get & check length
			length := int(b[0])
			if len(b) < 5+length {
				return []byte{}, fmt.Errorf("line %d: short line (%d<%d)", len(b),
					5+length)
			}

			// get address, note 64KB limit...
			addr := int(b[2]) + int(b[1])<<8

			// TODO: doesn't handle hex files over 64 KB
			switch b[3] { // switch on record type byte
			case 0x00: // data record
				if addr+length-1 > len(sw) {
					sw = sw[:addr+length]
				}
				copy(sw[addr:addr+length], b[4:4+length])
			}
		}
	}

	// round length up to BOOT_SIZE_ROUND multiple
	len := len(sw)
	if len%BOOT_SIZE_ROUND != 0 {
		len += BOOT_SIZE_ROUND - (len % BOOT_SIZE_ROUND)
	}

	sw2 := make([]byte, len) // save some space by reallocating
	copy(sw2, sw)
	return sw2, nil
}
