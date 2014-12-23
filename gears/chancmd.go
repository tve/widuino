// Copyright 2014 by Thorsten von Eicken, see LICENSE in top-level directory

package gears

import (
	"fmt"

	"github.com/docker/libchan"
)

const FormatAt = "2006-01-02 15:04:05.999"

// ===== Request and Reply formats =====

// "Union" of requests made over the main channel
type Request struct {
	ER    *EchoRequest       // simple ping-pong test request
	RFS   *RFSubRequest      // subscribe to raw RF messages
	RF    *RFSendRequest     // send a raw RF message
	SI    *SensorInfoRequest // get sensor info
	SD    *SensorDataRequest // send sensor data
	SR    *SensorReadRequest // read averaged sensor data
	SS    *SensorSubRequest  // subscribe to real-time sensor data
	PP    *ParamPutRequest   // put an arbitrary parameter
	PG    *ParamGetRequest   // get an arbitrary parameter
	Reply libchan.Sender
}

type Reply struct {
	Code  int
	Error string
	ER    *EchoReply
	PG    *ParamReply
	SI    *SensorInfo
}

const (
	CodeOK = iota
	CodeClientError
	CodeServerError
)

// Echo request - simple ping-pong test request
type EchoRequest string // string to echo
type EchoReply string

// Put an arbitrary parameter - used to store random stuff
type ParamPutRequest struct {
	Name  string
	Value string
}
type ParamGetRequest struct {
	Name string
}
type ParamReply struct {
	Value string
}

// RFMessage Subscription request - subscribes to all RF Messages received by hub. The subscription
// can start in the past, in which case messages are replayed from the database and then seamlessly
// switched-over into the real-time stream.
type RFSubRequest struct {
	StartAt  int64          // timestamp of first message, 0=start with real-time stream
	Match    RFMessage      // matcher for messages (not yet implemented)
	Messages libchan.Sender // channel of RFMessage
}
type RFSendRequest RFMessage

// RF Message
type RFMessage struct {
	At    int64  // milliseconds since unix epoch
	Group byte   // RF network group
	Node  byte   // RF node ID (Node=0 -> bcast)
	DoAck bool   // Send with ack requested
	Kind  byte   // Message payload type ("module" numbers)
	Data  []byte // Message payload
}

func (m RFMessage) RfTag() string {
	return fmt.Sprintf("RFg%03di%02dk%02d", m.Group, m.Node, m.Kind)
}

// Sensor Info requests

type SensorInfoRequest struct {
	Name string
}
type SensorInfo struct {
	Unit string
	Rate bool
}

// Sensor Data request
type SensorDataRequest struct {
	Name   string // hierarchical sensor name & location
	Info   SensorInfo
	Values libchan.Receiver // channel of SensorDataValue
}
type SensorDataValue struct {
	At    int64 // milliseconds since unix epoch
	Value float64
}

// Sensor Read request
type SensorReadRequest struct {
	Name    string
	StartAt int64          // first data point, milliseconds since unix epoch
	EndAt   int64          // last data point (inclusive), milliseconds since unix epoch
	Step    int64          // step in millisecsond, (EndAt-StartAt)%Step must be 0
	Values  libchan.Sender // channel of SensorDataValue
}

// Sensor Subscription Request
type SensorSubRequest struct {
	Name    string
	StartAt int64          // first data point, milliseconds since unix epoch
	Values  libchan.Sender // channel of SensorDataValue
}
