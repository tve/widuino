// Copyright 2014 by Thorsten von Eicken, see LICENSE in top-level directory

// Boot protocol handler
package main

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/golang/glog"
)

//===== Boot protocol messages

const PairingRequestLen = 22

type PairingRequest struct {
	NodeType uint16    // type of this remote node, 100..999 freely available
	GroupId  uint8     // current network group, 1..250 or 0 if unpaired
	NodeId   uint8     // current node ID, 1..30 or 0 if unpaired
	Check    uint16    // crc checksum over the current shared key (NOT USED)
	HwId     [16]uint8 // unique hardware ID or 0's if not available
}

type PairingReply struct {
	NodeType uint16   // type of this remote node, 100..999 freely available
	GroupId  uint8    // current network group, 1..250 or 0 if unpaired
	NodeId   uint8    // current node ID, 1..30 or 0 if unpaired
	ShKey    [16]byte // shared key or 0's if not used
}

const UpgradeRequestLen = 8

type UpgradeRequest struct {
	NodeType uint16 // type of this remote node, 100..999 freely available
	SwId     uint16 // current software ID or 0 if unknown
	SwSize   uint16 // current software download size, in units of 16 bytes
	SwCheck  uint16 // current crc checksum over entire download
}

type UpgradeReply struct {
	NodeType uint16 // type of this remote node, 100..999 freely available
	SwId     uint16 // current software ID or 0 if unknown
	SwSize   uint16 // software download size, in units of 16 bytes
	SwCheck  uint16 // current crc checksum over entire download
}

const DownloadRequestLen = 4

type DownloadRequest struct {
	SwId    uint16 // current software ID
	SwIndex uint16 // current download index, as multiple of payload size
}

const BOOT_DATA_MAX = 64
const BOOT_SIZE_ROUND = 16

type DownloadReply struct {
	SwIdXorIx uint16              // current software ID xor current download index
	Data      [BOOT_DATA_MAX]byte // download payload
}

// Booter interface

type Booter interface {
	Pair(PairingRequest) *PairingReply
	Upgrade(UpgradeRequest) *UpgradeReply
	Download(DownloadRequest) *DownloadReply
}

type pairingInfo [3]uint16 // reading json: nodeType, groupId, nodeId

// hwIdIsZero returns true if the 16-byte hardware ID is all zeroes, meaning, it's unset
func hwIdIsZero(hwId [16]uint8) bool {
	for _, id := range hwId {
		if id != 0 {
			return false
		}
	}
	return true
}

func (b *booter) Pair(req PairingRequest) *PairingReply {
	repl := PairingReply{}
	var hwId string

	// handle hwId
	if hwIdIsZero(req.HwId) {
		// We need to assign a random hwId...
		_, err := rand.Read(repl.ShKey[0:8])
		if err != nil {
			glog.Error("Can't generate random hwId: %s", err.Error())
		}
		glog.Infof("New HwID: %x", repl.ShKey)
		// convert hwId to a string
		hwId = hex.EncodeToString(repl.ShKey[:])
	} else {
		// convert hwId to a string
		hwId = hex.EncodeToString(req.HwId[:])
	}

	// see what we should reply with...
	info, ok := b.nodeType[hwId]
	if !ok {
		info = b.nodeType["00000000000000000000000000000000"]
	}

	// construct reply
	repl.NodeType = info[0]
	repl.GroupId = uint8(info[1])
	repl.NodeId = uint8(info[2])

	glog.Infof("  Reply: RF%di%d new NodeType=%d hwId=%x",
		repl.GroupId, repl.NodeId, repl.NodeType, repl.ShKey)
	return &repl
}

func (b *booter) Upgrade(req UpgradeRequest) *UpgradeReply {
	sw, err := b.findSoftware(req.NodeType)
	if err != nil {
		glog.Warningf("Cannot load sketch for nodeType %d: %s", req.NodeType, err.Error())
		return nil
	}

	glog.Infof("  Reply: NodeType=%d sw=%d (0x%x)", req.NodeType, req.SwId, sw.Crc16())
	return &UpgradeReply{
		NodeType: req.NodeType,
		SwId:     req.NodeType,
		SwSize:   uint16(len(sw) / 16), // sw size in units of 16 bytes
		SwCheck:  sw.Crc16(),
	}
}

func (b *booter) Download(req DownloadRequest) *DownloadReply {
	sw, err := b.findSoftware(req.SwId)
	if err != nil {
		glog.Warningf("Cannot load sketch for NodeType %d: %s", req.SwId, err.Error())
		return nil
	}
	offset := req.SwIndex * BOOT_DATA_MAX
	if offset >= uint16(len(sw)) {
		glog.Warningf("Request beyond end of sw: ix=%d -> off=%d, SwSize=%d",
			req.SwIndex, offset, len(sw))
		return nil
	}
	// Extract the chunk of data
	data := sw[offset:]
	if len(data) > BOOT_DATA_MAX {
		data = data[:BOOT_DATA_MAX]
	}
	// Compose reply
	repl := DownloadReply{SwIdXorIx: req.SwId ^ req.SwIndex}
	glog.Infof("  Reply: sw=%d %d bytes", req.SwId, len(data))
	deWhiten(data, repl.Data[:])
	return &repl
}

// ===== De-whitening

// de-whitening prevents simple runs of all-0 or all-1 bits
func deWhiten(inBuf, outBuf []byte) {
	for i, _ := range inBuf {
		//fmt.Printf("[%d]: %2x ^ %2x -> %2x\n", i, inBuf[i], byte(211*i), inBuf[i]^byte(211*i))
		outBuf[i] = inBuf[i] ^ byte(211*i)
	}
}

// ===== CRC-16

// Table for software.Crc16()
var crcTable = []uint16{
	0x0000, 0xCC01, 0xD801, 0x1400, 0xF001, 0x3C00, 0x2800, 0xE401,
	0xA001, 0x6C00, 0x7800, 0xB401, 0x5000, 0x9C01, 0x8801, 0x4400,
}

// Calculate Arduino-style CRC-16
func (sw software) Crc16() uint16 {
	var crc uint16 = 0xFFFF
	for _, b := range sw {
		crc = crc>>4 ^ crcTable[crc&0x0F] ^ crcTable[b&0x0F]
		crc = crc>>4 ^ crcTable[crc&0x0F] ^ crcTable[b>>4]
	}
	return crc
}

/* Reference implementation
func (sw software) Crc16b() uint16 {
	var crc uint16 = 0xFFFF
	for _, b := range sw {
		crc ^= uint16(b)
		for i := 0; i < 8; i++ {
			if crc&1 != 0 {
				crc = (crc >> 1) ^ 0xA001
			} else {
				crc = (crc >> 1)
			}
		}
	}
	return crc
}
*/
