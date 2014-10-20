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

	"github.com/golang/glog"
	"gopkg.in/fsnotify.v1"
)

type software []byte

type booter struct {
	configFile string                 // path to config file
	dir        string                 // directory with config and hex files
	watcher    *fsnotify.Watcher      // watcher for all the files
	nodeType   map[string]pairingInfo // map HwId -> nodeType, groupId, nodeId
	sketch     map[uint16]string      // map NodeType -> .hex file
	software   map[string]software    // map .hex file -> sketch hex data
}

var commentRe = regexp.MustCompile(`(?m)#.*$`)

func NewBooter(configFile string) *booter {
	b := booter{
		configFile: configFile,
		nodeType:   make(map[string]pairingInfo),
		sketch:     make(map[uint16]string),
		software:   make(map[string]software),
	}

	var err error
	b.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		glog.Warningf("Cannot watch config files: %s", err.Error())
	}
	b.watchHandler(b.watcher)
	b.watcher.Add(configFile)
	b.watcher.Events <- fsnotify.Event{configFile, fsnotify.Create}

	return &b
}

// ===== Filesystem change notifications

// eventHandler waits for filesystem event notifications and causes the config to be reloaded
func (b *booter) watchHandler(watcher *fsnotify.Watcher) {
	go func() {
		for event := range watcher.Events {
			glog.Info("config watcher event: ", event)
			if event.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Rename) == 0 {
				continue
			}
			if event.Name == b.configFile {
				// reload the config, start by disposing of existing info
				b.nodeType = make(map[string]pairingInfo)
				b.sketch = make(map[uint16]string)
				for k, _ := range b.software {
					if k != b.configFile {
						b.watcher.Remove(k)
					}
				}
				b.software = make(map[string]software)
				// now load fresh
				err := b.readBootConfig()
				if err != nil {
					glog.Errorf("Config error in %s: %s",
						b.configFile, err.Error())
				}
			} else {
				if _, ok := b.software[event.Name]; ok {
					b.readHexFile(event.Name)
				} else {
					b.watcher.Remove(event.Name)
				}
			}
		}
	}()
	go func() {
		for err := range watcher.Errors {
			glog.Warningf("boot info watch error:", err.Error())
		}
	}()
}

// ===== Read boot configuration

// read the boot config file
func (b *booter) readBootConfig() error {
	b.dir = path.Dir(b.configFile)

	config, err := ioutil.ReadFile(b.configFile)
	if err != nil {
		return err
	}

	// remove comments
	config = commentRe.ReplaceAllLiteral(config, []byte{})

	// parse json
	d := json.NewDecoder(bytes.NewReader(config))

	err = d.Decode(&b.nodeType)
	if err != nil {
		return fmt.Errorf("error reading pairing: %s", err.Error())
	}
	for i, v := range b.nodeType {
		glog.Infof("  node %s -> nodeType=%d RF%di%d", i, v[0], v[1], v[2])
	}

	var sketches map[string]string
	err = d.Decode(&sketches)
	if err != nil {
		return fmt.Errorf("error reading sketches: %s", err.Error())
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
	return nil
}

// ===== Read Intel hex files

func (b *booter) findSoftware(id uint16) (software, error) {
	file, ok := b.sketch[id]
	if !ok {
		return []byte{}, fmt.Errorf("no sketch is configured")
	}
	if file[0] != '/' {
		file = path.Join(b.dir, file)
	}

	sw, ok := b.software[file]
	glog.V(2).Infof("findSoftware for nodeType=%d -> %s", id, file)
	if !ok {
		err := b.readHexFile(file)
		if err != nil {
			return []byte{}, err
		}
		sw = b.software[file]
	} else {
		glog.V(2).Infof("  retrieving cached software for nodeType=%d", id)
	}
	return sw, nil
}

// read software
func (b *booter) readHexFile(file string) error {
	b.watcher.Add(file)
	// load it
	glog.V(2).Infof("  reading %s", file)
	f, err := os.Open(file)
	if err != nil {
		return fmt.Errorf("opening %s: %s", file, err.Error())
	}
	sw, err := readHex(f)
	// add to cache
	if err != nil {
		return err
	}
	glog.V(2).Infof("  saving software %s in cache (%d bytes)", file, len(sw))
	b.software[file] = sw
	return nil
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
