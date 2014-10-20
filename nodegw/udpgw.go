// Copyright 2014 by Thorsten von Eicken, see LICENSE in top-level directory

// UDP driver for Widuino, see README.md for details
// The message emitted for received packets has the form:
// PacketMap{
//   "rf12demo": g%db%di%d" (group/band/id)
//   "group": int,
//   "band" : int, // TODO: get the band from the GW sketch
//   "node" : int,
//   "type" : int, (3 packet type bits from header byte)
//   "raw"  : []byte,
// }

package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/golang/glog"
)

type Message map[string]interface{}

// message type codes used in UDP packets
const (
	RF_BcastPush = iota
	RF_BcastReq
	RF_DataPush
	RF_DataReq
	RF_AckData
	RF_BootReq
	RF_AckBcast
	RF_BootReply
	RF_Pairing
	RF_Debug
)

// UDP Gateway communicates with UDP/RF gateway nodes via UDP
// Registers as "UDP-Gateway"
type UDPGateway struct {
	Port     int          // UDP port number
	Recv     chan Message // channel for received messages
	Xmit     chan Message // channel to transmit messages
	Boot     Booter       // where to call to get boot data
	sock     *net.UDPConn
	groupMap *GroupMap // map between groups and GW IP addresses
}

func (u *UDPGateway) Run() {
	if u.groupMap == nil {
		u.groupMap = &GroupMap{group: make(map[byte]*net.UDPAddr)}
	}
	u.Listen(u.Port)
	go u.Transmitter()
	//go u.Booter()
	u.Receiver()
}

// send a packet (here the flags are 0..7)
func (u *UDPGateway) sendPacket(group, node, flags byte, data []byte) {
	// find UDP gateway's address
	addr := u.groupMap.mapGroupToAddr(group)
	if addr == nil {
		glog.Warningf("No GW known for RF group %d", group)
		return
	}
	// puts the UDP packet together
	buf := make([]byte, len(data)+3)
	buf[0] = flags // message type code
	buf[1] = group // RF group
	buf[2] = node  // node id
	copy(buf[3:], data)
	// logging and sending
	glog.Infof("Snd packet len=%d dst=%v node=%d", len(buf), addr, node)
	glog.V(2).Infof("  Send: %+v", buf)
	glog.V(4).Infof("  Pkt=%#v", buf)
	u.sock.WriteToUDP(buf, addr)
}

// Get packets to xmit, encode them, and ship them out
func (u *UDPGateway) Transmitter() {
	for m := range u.Xmit {
		if node, ok := m["node"].(byte); ok {
			group, gOk := m["group"].(byte)
			kind, kOk := m["kind"].(byte)
			data, dOk := m["data"].([]byte)
			if gOk && dOk {
				var flags byte = 0x3 // data_req
				if node == 0 {
					flags = 0x0 // bcast_push
				}
				if kOk {
					data = append([]byte{kind}, data...)
				}
				u.sendPacket(group, node, flags, data)
			} else {
				glog.Error("Message lacks group(%t) or data(%t): %+v",
					!gOk, !dOk, m)
			}
		} else {
			glog.Error("Unknown Message kind: %+v", m)
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	} else {
		return b
	}
}

func (u *UDPGateway) handlePairingRequest(pktSrc *net.UDPAddr, groupId, nodeId byte, data []byte) {
	pktLen := len(data)
	glog.Infof("UDP Recv boot pairing src=%v len=%d", pktSrc, pktLen)
	if pktLen != PairingRequestLen+3 {
		glog.Warningf("  Incorrect length=%d (!= %d)",
			pktLen, PairingRequestLen+3)
		return
	}
	pr := PairingRequest{}
	err := binary.Read(bytes.NewReader(data[3:]), binary.LittleEndian, &pr)
	if err != nil {
		glog.Warningf("  Cannot decode PairingRequest: %s", err.Error())
		return
	}
	glog.Infof("  PairingRequest: nodeType:%d RF%di%d HwID:%x", pr.NodeType,
		pr.GroupId, pr.NodeId, pr.HwId)
	reply := u.Boot.Pair(pr)
	if reply != nil {
		buf := bytes.Buffer{}
		_ = binary.Write(&buf, binary.LittleEndian, reply)
		u.sendPacket(groupId, nodeId, 0x8, buf.Bytes())
	}
}

func (u *UDPGateway) handleUpgradeRequest(pktSrc *net.UDPAddr, groupId, nodeId byte, data []byte) {
	pktLen := len(data)
	glog.Infof("UDP Recv boot upgrade src=%v len=%d", pktSrc, pktLen)
	ur := UpgradeRequest{}
	err := binary.Read(bytes.NewReader(data[3:]), binary.LittleEndian, &ur)
	if err != nil {
		glog.Warningf("  Cannot decode UpgradeRequest: %s", err.Error())
		return
	}
	glog.Infof("  UpgradeRequest: nodeType:%d RF%di%d Sw: [id=%d size=%d check=0x%04x]",
		ur.NodeType, groupId, nodeId, ur.SwId, ur.SwSize, ur.SwCheck)
	reply := u.Boot.Upgrade(ur)
	if reply != nil {
		buf := bytes.Buffer{}
		_ = binary.Write(&buf, binary.LittleEndian, reply)
		u.sendPacket(groupId, nodeId, 0x7, buf.Bytes())
	}
}

func (u *UDPGateway) handleDownloadRequest(pktSrc *net.UDPAddr, groupId, nodeId byte, data []byte) {
	pktLen := len(data)
	glog.Infof("UDP Recv boot download src=%v len=%d", pktSrc, pktLen)
	dr := DownloadRequest{}
	err := binary.Read(bytes.NewReader(data[3:]), binary.LittleEndian, &dr)
	if err != nil {
		glog.Warningf("  Cannot decode DownloadRequest: %s", err.Error())
		return
	}
	glog.Infof("  DownloadRequest: RF%di%d Sw: [id=%d ix=%d off=%d]",
		groupId, nodeId, dr.SwId, dr.SwIndex, dr.SwIndex*BOOT_DATA_MAX)
	reply := u.Boot.Download(dr)
	if reply != nil {
		buf := bytes.Buffer{}
		_ = binary.Write(&buf, binary.LittleEndian, reply)
		u.sendPacket(groupId, nodeId, 0x7, buf.Bytes())
	}
}

// Receive UDP packets, decode them, and output them
// The packet format is (by byte): flags, group, node_id, kind, data...
func (u *UDPGateway) Receiver() {
	pkt := make([]byte, 1600)
	for {
		glog.V(2).Infoln("******************************")
		pktLen, pktSrc, err := u.sock.ReadFromUDP(pkt)
		if err != nil {
			glog.Warning("UDP error: " + err.Error())
			continue
		}
		data := pkt[0:pktLen]
		if pktLen < 3 {
			glog.Infof("UDP: got too short a packet (%d) from %v", pktLen, pktSrc)
			continue
		}
		if pktLen > 66+3 {
			glog.Infof("UDP: got too long a packet (%d) from %v", pktLen, pktSrc)
			continue
		}
		// got a reasonable packet
		flags := data[0]
		groupId := data[1]
		nodeId := data[2]

		// Record the groupId -> addr mapping
		_ = u.groupMap.saveGroupToAddr(groupId, pktSrc)

		switch flags {
		// CTL + ACK + DST -> boot protocol pairing request
		// I.e: a node sends us its HW ID and we reply with groupId/nodeId/nodeType
		case 8:
			u.handlePairingRequest(pktSrc, groupId, nodeId, data)

		// CTL + ACK -> boot protocol upgrade or download request
		// I.e.: a node asks for the software Id & checksum or downloads a chunk
		case 5:
			switch pktLen {
			case UpgradeRequestLen + 3:
				u.handleUpgradeRequest(pktSrc, groupId, nodeId, data)
			case DownloadRequestLen + 3:
				u.handleDownloadRequest(pktSrc, groupId, nodeId, data)
			default:
				glog.Warningf("  Incorrect length=%d (!= %d)",
					pktLen, PairingRequestLen+3)
			}

		// Special packet to log from UDP GW itself
		case 9:
			glog.Infof("UDP-GW: %s", string(data[3:]))
		// Standard data packet, produce a Message
		case 0, 1:
			msg := Message{
				"rf":    fmt.Sprintf("RFg%di%d", groupId, nodeId),
				"group": groupId,
				"node":  nodeId,
				"at":    time.Now(),
			}
			var kind byte
			if pktLen > 3 {
				kind = data[3]
				msg["kind"] = kind
				msg["data"] = data[4:]
			} else {
				msg["data"] = data[3:]
			}
			glog.Infof("UDP Recv: src=%v RF%di%d kind=%d len=%d",
				pktSrc, groupId, nodeId, kind, pktLen)
			glog.V(4).Infof("  Pkt=%+v", data[0:min(len(data), 10)])
			// If an ACK is requested we should send that asap
			if flags&1 != 0 {
				u.sendPacket(groupId, nodeId, 0x6, []byte{})
			}
			// Now process what we got
			u.Recv <- msg
		}
	}
}

// Open a UDP port
func (u *UDPGateway) Listen(port int) {
	udpAddr := net.UDPAddr{Port: port}
	sock, err := net.ListenUDP("udp4", &udpAddr)
	if err != nil {
		glog.Fatalf("Can't listen to UDP :%d : %s", port, err.Error())
	}
	u.sock = sock
	glog.Infof("Listening on UDP port %d", port)
}

//===== Map of group_id to UDP ip:port

// When a JeeUDP gw comes up it sends us some hello-world packets so we can learn about
// the RF network group id that it handles. We keep track of that here so we can send
// packets in the reverse direction.

type GroupMap struct {
	group      map[byte]*net.UDPAddr // group id -> UDP address
	sync.Mutex                       // synchronize access to map
}

// lookup group_id -> UDP addr, returns nil if we don't have a mapping
func (gm *GroupMap) mapGroupToAddr(groupId byte) *net.UDPAddr {
	gm.Lock()
	defer gm.Unlock()
	if gm.group == nil {
		gm.group = make(map[byte]*net.UDPAddr)
	}
	return gm.group[groupId]
}

// save group_id -> UDP addr mapping, return true if this is new groupId
func (gm *GroupMap) saveGroupToAddr(groupId byte, addr *net.UDPAddr) bool {
	gm.Lock()
	defer gm.Unlock()
	if gm.group == nil {
		gm.group = make(map[byte]*net.UDPAddr)
	}
	newGroup := gm.group[groupId] == nil
	if newGroup || !gm.group[groupId].IP.Equal(addr.IP) || gm.group[groupId].Port != addr.Port {
		glog.Infof("RF group %d now reachable via %s:%d", groupId, addr.IP, addr.Port)
	}
	gm.group[groupId] = addr
	return newGroup
}
