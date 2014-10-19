// Copyright 2014 by Thorsten von Eicken, see LICENSE in top-level directory

// Booter for widuino - JeeBoot compatible boot loader
// Listens to MQTT messages on topics /rf/<group_id>/<node_id>/rb and responds using the
// corresponding .../tx topics.

// TODO: this has lots of code overlap with udpgw.go, need to factor stuff out or use JeeBus
// if that stabilizes...

package main

import (
	"flag"
	"os"

	//"bytes"
	//"encoding/base64"
	//"encoding/binary"
	//"encoding/json"
	//"fmt"
	//"github.com/golang/glog"
	//"log"
	//"net"
	//"regexp"
	//"strconv"
	//"time"

	//proto "github.com/huin/mqtt"
	//"github.com/jeffallen/mqtt"
)

var bootConfig = flag.String("bootConfig", "sketches.json", "config file for boot server")

//var host = flag.String("mqtt", "localhost:1883", "hostname:port of mqtt broker")
//var id = flag.String("id", "wiboot", "client id")
//var user = flag.String("user", "", "username")
//var pass = flag.String("pass", "", "password")

//===== MQTT Json Message (this should be defined in some shared global place)

type RFMessage struct {
	At     int64  `json:"_at",omitempty`
	Kind   string `json:"kind",omitempty`
	Base64 string `json:"base64",omitempty`
}

//===== Main

func main() {
	flag.Parse()
	recv := make(chan Message, 10000)
	xmit := make(chan Message, 10000)

	//fmt.Fprintln(os.Stdout, "STDOUT")
	//fmt.Fprintln(os.Stderr, "STDERR")
	//glog.Error("glog.Error")
	//glog.Warning("glog.Warning")
	//glog.Info("glog.Info")

	booter := NewBooter(*bootConfig)
	if booter == nil {
		os.Exit(1)
	}
	udpGw := &UDPGateway{Port: 9999, Recv: recv, Xmit: xmit, Boot: booter}
	udpGw.Run()

}

/*

func main() {
	flag.Parse()

		// MQTT connection
		mqttSock, err := net.Dial("tcp", *host)
		if err != nil {
			log.Fatalf("MQTT broker: %s", err)
		}
		cc := mqtt.NewClientConn(mqttSock)
		cc.ClientId = *id
		if err := cc.Connect(*user, *pass); err != nil {
			log.Fatalf("Mqtt connect failed: %v\n", err)
		}
		log.Println("Connected with client id", cc.ClientId)

	// do some work..
	bootServer(cc)
}

//===== Boot message processor

var bootTopics = []string{"/rf/+/+/rb"}

func bootServer(cc *mqtt.ClientConn) {
	// subscribe to boot topics
	bt := make([]proto.TopicQos, len(bootTopics))
	for i, t := range bootTopics {
		bt[i].Topic = t
		bt[i].Qos = proto.QosAtMostOnce
	}
	cc.Subscribe(bt)

	// now sit there and listen to incoming messages and dispatch them
	for m := range cc.Incoming {
		payload := []byte(m.Payload.(proto.BytesPayload))
		log.Printf("RCV mqtt: %s %d bytes", m.TopicName, len(payload))
		groupId, nodeId, err := decodeTopic(m.TopicName)
		if err != nil {
			continue
		}
		msg, data, err := decodeJsonPayload(payload)
		if err != nil {
			continue
		}
		handleBoot(cc, groupId, nodeId, msg, data)
	}
}

var topicRE = regexp.MustCompile(`^/rf/(\d+)/(\d+)/rb`)

func decodeTopic(topic string) (byte, byte, error) {
	// decode the topic to determine where it's coming from
	tt := topicRE.FindStringSubmatch(topic)
	if len(tt) != 3 {
		log.Printf("Cannot parse MQTT message topic: %s", topic)
		return 0, 0, fmt.Errorf("error")
	}
	groupId, _ := strconv.Atoi(tt[1])
	nodeId, _ := strconv.Atoi(tt[2])
	return byte(groupId), byte(nodeId), nil
}

func decodeJsonPayload(payload []byte) (*RFMessage, []byte, error) {
	// Parse message payload as JSON
	var msg RFMessage
	err := json.Unmarshal(payload, &msg)
	if err != nil {
		log.Printf("Cannot parse JSON payload of MQTT message: %s", err)
		return nil, nil, fmt.Errorf("error")
	}
	data, _ := base64.StdEncoding.DecodeString(msg.Base64)
	return &msg, data, nil
}

func encodeBootMessage(groupId, nodeId byte, msgStruct interface{}) *proto.Publish {
	topic := fmt.Sprintf("/rf/%d/%d/tb", groupId, nodeId)

	// "Serialize" the binary message struct
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, msgStruct)

	// Create the payload
	payload, _ := json.Marshal(RFMessage{
		AsOf:   time.Now().UnixNano() / 1000000, // Javascript time: milliseconds
		Base64: base64.StdEncoding.EncodeToString(buf.Bytes()),
		Kind:   "boot",
	})

	// Create the publish info
	return &proto.Publish{
		Header:    proto.Header{QosLevel: proto.QosLevel(0)},
		TopicName: topic,
		Payload:   proto.BytesPayload(payload),
	}
}

//===== Handle

func handleBoot(cc *mqtt.ClientConn, groupId, nodeId byte, msg *RFMessage, data []byte) {
	switch msg.Kind {
	case "pairing":
		// Decode the pairing message
		pr := PairingRequest{}
		err := binary.Read(bytes.NewReader(data), binary.LittleEndian, &pr)
		if err != nil {
			log.Printf("Can't parse pairing message: %s", err)
			return
		}
		log.Printf("  Pairing from %d/%d type=%d crc=0x%04x hwId=%x", pr.GroupId,
			pr.NodeId, pr.NodeType, pr.Check, pr.HwId)
		// Reply with a groupId/nodeId assignment
		prr := PairingReply{pr.NodeType, groupId, 17, [16]byte{}}
		cc.Publish(encodeBootMessage(groupId, pr.NodeId, prr))

	case "boot":
		// Decode the boot message
		switch len(data) {
		case 8: // Upgrade Request
			ur := UpgradeRequest{}
			err := binary.Read(bytes.NewReader(data), binary.LittleEndian, &ur)
			if err != nil {
				log.Printf("Can't parse upgrade request message: %s", err)
				return
			}
			log.Printf("  Upgrade request from %d/%d type=%d swId=%d", groupId,
				nodeId, ur.NodeType, ur.SwId)
			// Reply with a software assignment
			urr := UpgradeReply{ur.NodeType, 55, 1024, 0x1234}
			cc.Publish(encodeBootMessage(groupId, nodeId, urr))

		case 4: // Download Request
		default:
			log.Printf("Unknown boot message with length=%d", len(data))
		}
	default:
		log.Printf("Unknown boot message with kind=%s", msg.Kind)
	}
}
*/
