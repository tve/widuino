package main

import (
        "flag"
        "encoding/base64"
        "fmt"
        "encoding/json"
        "log"
        "net"
        "sync"
        "regexp"
        "strconv"
        "time"

        proto "github.com/huin/mqtt"
        "github.com/jeffallen/mqtt"
)

var host = flag.String("mqtt", "localhost:1883", "hostname:port of mqtt broker")
var udp  = flag.Int("udp", 9999, "UDP port to listen on")
var id = flag.String("id", "udp-gw", "client id")
var user = flag.String("user", "", "username")
var pass = flag.String("pass", "", "password")
var dump = flag.Bool("dump", false, "dump messages?")

//===== MQTT Json Message (this should be defined in some shared global place)

type RFMessage struct {
        AsOf            int64   `json:"_asof",omitempty`
        Kind            string  `json:"kind",omitempty`
        Base64          string  `json:"base64",omitempty`
}

// message type codes used in UDP packets
const (
        RF_BcastPush    = iota
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

//===== Map of group_id to UDP ip:port

// When a JeeUDP gw comes up it sends us some hello-world packets so we can learn about
// the RF network group id that it handles. We keep track of that here so we can send
// packets in the reverse direction.

var networks = make(map[byte]*net.UDPAddr)       // group_id -> UDP address
var networksMutex sync.Mutex                    // synchronize access to the networks map

// lookup group_id -> UDP addr, returns nil if we don't have a mapping
func mapGroupToAddr(groupId byte) (*net.UDPAddr) {
        networksMutex.Lock()
        defer networksMutex.Unlock()
        return networks[groupId]
}

// save group_id -> UDP addr mapping
func saveGroupToAddr(groupId byte, addr *net.UDPAddr) {
        networksMutex.Lock()
        defer networksMutex.Unlock()
        if networks[groupId] == nil || !networks[groupId].IP.Equal(addr.IP) ||
           networks[groupId].Port != addr.Port {
                log.Printf("RF group %d now reachable via %s:%d", groupId, addr.IP, addr.Port)
        }
        networks[groupId] = addr
}

//===== Main

func main() {
        flag.Parse()

        // MQTT connection

        mqttSock, err := net.Dial("tcp", *host)
        if err != nil {
                log.Fatalf("MQTT broker: %s", err)
        }
        cc := mqtt.NewClientConn(mqttSock)
        cc.Dump = *dump
        cc.ClientId = *id
        if err := cc.Connect(*user, *pass); err != nil {
                log.Fatalf("Mqtt connect failed: %v\n", err)
        }
        log.Println("Connected with client id", cc.ClientId)

        // UDP Socket

        udpAddr := net.UDPAddr{Port: *udp}
        udpSock, err := net.ListenUDP("udp4", &udpAddr)
        if err != nil {
                log.Fatalf("Can't listen to UDP :%d : %s", *udp, err.Error())
        }
        log.Printf("Listening on UDP port %d", *udp);
        go udpServer(udpSock, cc);

        // do some work..
        mqttServer(cc, udpSock)
}

//===== MQTT message processor

func mqttServer(cc *mqtt.ClientConn, udp *net.UDPConn) {
        for m := range cc.Incoming {
                payload := []byte(m.Payload.(proto.BytesPayload))
                log.Printf("RCV mqtt: %s %d bytes", m.TopicName, len(payload))
                encode(udp, m.QosLevel>0, m.TopicName, payload)
        }

/*
        tq := make([]proto.TopicQos, flag.NArg())
        for i := 0; i < flag.NArg(); i++ {
                tq[i].Topic = flag.Arg(i)
                tq[i].Qos = proto.QosAtMostOnce
        }
        cc.Subscribe(tq)
*/
}

//===== UDP packet processor

func udpServer(udp *net.UDPConn, mqttConn *mqtt.ClientConn) {
        pkt := make([]byte, 1600)
        for {
                pktLen, pktSrc, err := udp.ReadFromUDP(pkt)
                if err != nil {
                        log.Printf("UDP error: " + err.Error())
                        continue
                }
                if pktLen < 4 {
                        log.Printf("UDP: got too short a packet (%d) from %v",
                                pktLen, pktSrc)
                        continue
                }
                //log.Printf("Got packet len=%d src=%v", pktLen, pktSrc);
                decode(mqttConn, pktSrc, pkt[0:pktLen])
        }
}

//===== Encode MQTT message into a UDP packet

var txTopicRE = regexp.MustCompile(`^rf/(\d+)/(\d+)/(t.)$`) // rf/group_id/node_id/(tx|tb)

func encode(udp *net.UDPConn, ack bool, topic string, payload []byte) {
        // decode the topic to determine where we're sending this
        tt := txTopicRE.FindStringSubmatch(topic)
        if len(tt) != 4 {
                log.Printf("Cannot parse MQTT message topic for transmission: %s", topic)
                return
        }
        groupId, _ := strconv.Atoi(tt[1])
        nodeId, _  := strconv.Atoi(tt[2])
        boot := tt[3] == "tb"
        unicast := nodeId > 0 && nodeId < 31
        if !unicast && nodeId != 0 {
                log.Printf("Invalid nodeId=%d in MQTT message", nodeId)
                return
        }

        // map the group_id to the appropriate UDP destination
        addr := mapGroupToAddr(byte(groupId))
        if addr == nil {
                log.Printf("No GW known for RF group %d", groupId)
                return
        }

        // Parse message payload as JSON
        var msg RFMessage
        err := json.Unmarshal(payload, &msg)
        if err != nil {
                log.Printf("Cannot parse JSON payload of MQTT message: %s", err)
                return
        }
        data, _ := base64.StdEncoding.DecodeString(msg.Base64)

        // Figure out the message type code
        var code byte
        switch {
        //case boot && msg.Kind == "pairing":     code = RF_Pairing // not sent out to nodes!
        case  boot && msg.Kind == "boot" && unicast:     code = RF_BootReply
        case !boot &&  ack &&  unicast:                  code = RF_DataReq
        case !boot &&  ack && !unicast:                  code = RF_BcastReq
        case !boot && !ack &&  unicast:                  code = RF_DataPush
        case !boot && !ack && !unicast:                  code = RF_BcastPush
        default:
                log.Printf("Invalid MQTT message combo: boot=%t kind=%s ack=%t",
                        boot, msg.Kind, ack)
        }

        // actually send the packet
        buf := make([]byte, len(data)+3)
        buf[0]  = code
        buf[1]  = byte(groupId)
        buf[2]  = byte(nodeId)
        copy(buf[3:], data)
        udp.WriteToUDP(buf, addr)
}

//===== Decode UDP packet into MQTT message

func decode(mqtt *mqtt.ClientConn, src *net.UDPAddr, pkt []byte) {
        // Parse packet
        code    := pkt[0]
        groupId := pkt[1]
        nodeId  := pkt[2] & 0x1f
        ack     := 0 // really need to decode pkt[2]
        data    := pkt[3:]

        // Record the groupId -> addr mapping
        saveGroupToAddr(groupId, src)

        // Create the topic
        if code > RF_Debug && code != RF_BootReply {
                log.Printf("Dropping UDP packet due to unprocessable code=%d", code)
                log.Printf("%#v", pkt[0:9])
                return
        }
        // handle boot protocol
        rxrb := "rx"; kind := ""
        switch code {
        case RF_Pairing: rxrb = "rb"; kind = "pairing"
        case RF_BootReq: rxrb = "rb"; kind = "boot"
        }
        // handle packets with no source node id
        switch code {
        case RF_DataPush, RF_DataReq, RF_AckBcast, RF_BootReply, RF_Debug: nodeId = 0;
        }
        // finally the topic
        topic := fmt.Sprintf("/rf/%d/%d/%s", groupId, nodeId, rxrb)

        // Create the payload
        payload, _ := json.Marshal(RFMessage{
                AsOf: time.Now().UnixNano() / 1000000, // Javascript time: milliseconds
                Base64: base64.StdEncoding.EncodeToString(data),
                Kind: kind,
        })

        // Send it off
        mqtt.Publish(&proto.Publish{
                Header:    proto.Header{QosLevel: proto.QosLevel(ack)},
                TopicName: topic,
                Payload:   proto.BytesPayload(payload),
        })

        // Log message, if appropriate
        if code == 9 {
                log.Printf("JeeUDP: %s", data)
        }
        log.Printf("MQTT PUB %s code=%d len=%d", topic, code, len(pkt))
}
