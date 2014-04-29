// Copyright 2014 by Thorsten von Eicken, see LICENSE in top-level directory

// UDP GW JeeBus flow gadget for Widuino, see README.md for details

package network

import (
        "fmt"
        "net"
        "sync"

        "github.com/golang/glog"
        "github.com/jcw/flow"
)

func init() {
        flow.Registry["UDP-Gateway"] = func() flow.Circuitry { return &UDPGateway{} }
}

//===== UDPGateway gadget

// UDP Gateway communicates with UDP/RF gateway nodes via UDP
// Registers as "UDP-Gateway"
type UDPGateway struct {
        flow.Gadget
        Port flow.Input         // port number as float64
        Recv flow.Output        // regular incoming packets in rf12demo look-alike format
        Oob  flow.Output        // out-of-band (boot) incoming packets
        Rej  flow.Output        // unrecognized packets
        Xmit flow.Input         // outgoing packets
        sock *net.UDPConn
        group byte              // we can only handle one group for now :-(
}

func (w *UDPGateway) Run() {
        if port, ok := <-w.Port; ok {
                p := int(port.(float64))
                fmt.Printf("UDP-Gateway listening on port %d\n", p)
                w.Listen(p)
                go w.Transmitter()
                w.Receiver()
        }
}

// send a packet (here the flags are 0..7)
func (w *UDPGateway) sendPacket(group, node, flags byte, data []byte) {
	// find UDP gateway's address
	addr := mapGroupToAddr(group)
	if addr == nil {
		glog.Warningf("No GW known for RF group %d", group)
		return
	}
	// puts the UDP packet together
	buf := make([]byte, len(data)+3)
	buf[0] = flags                // message type code
	buf[1] = group                // RF group
	buf[2] = node                 // node id
	copy(buf[3:], data)
	// logging and sending
	glog.V(2).Infof("Snd packet len=%d dst=%v node=%d", len(buf), addr, node);
	glog.V(4).Infof("  Pkt=%#v", buf);
	w.sock.WriteToUDP(buf, addr)
}

// Get packets to xmit, encode them, and ship them out
func (w *UDPGateway) Transmitter() {
        for m := range w.Xmit {
                switch v := m.(type) {
                case string: // writes the string verbatim
                case int: // pulses DTR on serial line to reset
                case bool: // sets/clears RTS
                case []byte:
			// TODO: fix w.group
			w.sendPacket(w.group, v[0] & 0x1f, v[0]>>5, v[1:])
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

// Receive UDP packets, decode them, and output them
// The packet format is (by byte): flags, group, node_id, data...
func (w *UDPGateway) Receiver() {
        pkt := make([]byte, 1600)
        for {
                pktLen, pktSrc, err := w.sock.ReadFromUDP(pkt)
                if err != nil {
                        glog.Warning("UDP error: " + err.Error())
                        continue
                }
                data := pkt[0:pktLen]
                if pktLen < 4 {
                        glog.Infof("UDP: got too short a packet (%d) from %v", pktLen, pktSrc)
                        w.Rej.Send(data)
                } else {
                        flags := data[0]
                        groupId := data[1]
                        nodeId := data[2]

                        // Record the groupId -> addr mapping
                        newGroup := saveGroupToAddr(groupId, pktSrc)
                        if newGroup {
                                w.group = groupId
                                hack := map[string]int{
                                        "<RF12demo>": 12, "band": 915,
                                        "group": int(groupId), "id": 31,
                                }
                                w.Recv.Send(hack)
                        }

                        // Now mimick RF12demo, sigh
                        info := map[string]int{"<node>": int(nodeId)}
                        switch flags {
                        case 5, 8: // CTL and ACK are set -> boot protocol, need to drop group/len from header
                                glog.V(2).Infof("Got boot packet len=%d src=%v", pktLen, pktSrc);
                                glog.V(4).Infof("  Pkt=%#v", data[0:min(len(data),10)]);
                                w.Oob.Send(info)
                                dd := data[2:]
                                dd[0] = (flags << 5) | (nodeId & 0x1f)
                                w.Oob.Send(dd)
                        case 9: // Special packet to log from UDG GW itself
                                glog.V(1).Infof("UDP-GW: %s", string(data[3:]))
                        default: // standard data packet
                                glog.V(2).Infof("Got packet len=%d src=%v node=%d",
					pktLen, pktSrc, nodeId&0x1f);
                                glog.V(4).Infof("  Pkt=%#v", data[0:min(len(data),10)]);
				// If an ACK is requested we should send that asap
				if flags & 1 != 0 {
					w.sendPacket(groupId, nodeId, 0x6, []byte{})
				}
				// Now process the packet
                                data[0] = (flags << 5) | (nodeId & 0x1f)
                                data[2] = byte( len(data)-3 )
                                w.Recv.Send(info)
                                w.Recv.Send(data)
                        }
                }
        }
}

// Open a UDP port
func (w *UDPGateway) Listen(port int) {
        udpAddr := net.UDPAddr{Port: port}
        sock, err := net.ListenUDP("udp4", &udpAddr)
        if err != nil {
                glog.Fatalf("Can't listen to UDP :%d : %s", port, err.Error())
        }
        w.sock = sock
        glog.Infof("Listening on UDP port %d", port)
}


//===== Map of group_id to UDP ip:port

// When a JeeUDP gw comes up it sends us some hello-world packets so we can learn about
// the RF network group id that it handles. We keep track of that here so we can send
// packets in the reverse direction.

var networks = make(map[byte]*net.UDPAddr) // group_id -> UDP address
var networksMutex sync.Mutex               // synchronize access to the networks map

// lookup group_id -> UDP addr, returns nil if we don't have a mapping
func mapGroupToAddr(groupId byte) *net.UDPAddr {
        networksMutex.Lock()
        defer networksMutex.Unlock()
        return networks[groupId]
}

// save group_id -> UDP addr mapping, return true if this is new groupId
func saveGroupToAddr(groupId byte, addr *net.UDPAddr) bool {
        networksMutex.Lock()
        defer networksMutex.Unlock()
        newGroup := networks[groupId] == nil
        if newGroup || !networks[groupId].IP.Equal(addr.IP) || networks[groupId].Port != addr.Port {
                glog.Infof("RF group %d now reachable via %s:%d", groupId, addr.IP, addr.Port)
        }
        networks[groupId] = addr
        return newGroup
}
