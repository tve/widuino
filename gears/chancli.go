// Copyright 2014 by Thorsten von Eicken, see LICENSE in top-level directory

package gears

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/docker/libchan"
	"github.com/docker/libchan/spdy"
)

// ===== Open / close channels =====

type GearConn struct {
	conn     net.Conn
	mainChan libchan.Sender
	addr     string
}

func Dial(addr string) (*GearConn, error) {
	log.Printf("Opening libchan connection to %s", addr)

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}

	transport, err := spdy.NewClientTransport(conn)
	if err != nil {
		conn.Close()
		return nil, err
	}

	main, err := transport.NewSendChannel()
	if err != nil {
		conn.Close()
		return nil, err
	}

	if err := doEcho(main, time.Second); err != nil {
		conn.Close()
		return nil, fmt.Errorf("initial echo %s", err.Error())
	}
	log.Printf("Libchan connection to %s open", addr)
	gc := &GearConn{conn: conn, mainChan: main, addr: addr}

	go gc.pinger()

	return gc, nil
}

func (gc *GearConn) Close() {
	gc.mainChan.Close()
	time.Sleep(100 * time.Millisecond)
	gc.conn.Close()
	log.Printf("Libchan connection to %s closed", gc.addr)
}

// ===== Request functions =====

func (gc *GearConn) RFSubscribe(start int64) <-chan RFMessage {
	replyRecv, replySend := libchan.Pipe()
	c := make(chan RFMessage, 0)

	req := struct{ RFS RFSubRequest }{
		RFSubRequest{StartAt: start, Match: RFMessage{}, Reply: replySend},
	}

	err := gc.mainChan.Send(req)
	if err != nil {
		close(c)
		return c
	}

	go func() {
		for {
			var m RFMessage
			err := replyRecv.Receive(&m)
			if err != nil {
				log.Printf("Error receiving RF message: %s", err.Error())
				close(c)
				return
			}
			c <- m
		}
	}()

	return c
}

func (gc *GearConn) RFSend(msg RFMessage) error {
	var ackRecv libchan.Receiver
	var ackSend libchan.Sender
	if msg.DoAck {
		ackRecv, ackSend = libchan.Pipe()
	}

	req := struct{ RF RFSendRequest }{
		RFSendRequest{msg, ackSend},
	}

	err := gc.mainChan.Send(req)
	if err != nil {
		return err
	}
	if !msg.DoAck {
		return nil
	}

	ack := AckReply{}
	err = ackRecv.Receive(&ack)
	if err != nil {
		return err
	}
	if ack.Err != "" {
		return fmt.Errorf("RFMessage send failed with %s", ack.Err)
	}
	return nil
}

func (gc *GearConn) SensorSubscribe(name string, startAt int64) <-chan SensorDataValue {
	replyRecv, replySend := libchan.Pipe()
	c := make(chan SensorDataValue, 0)

	req := struct{ SS SensorSubRequest }{
		SensorSubRequest{name, startAt, replySend},
	}

	err := gc.mainChan.Send(req)
	if err != nil {
		close(c)
		return c
	}

	go func() {
		for {
			var m SensorDataValue
			err := replyRecv.Receive(&m)
			if err != nil {
				log.Printf("Error receiving SensorData message: %s", err.Error())
				close(c)
				return
			}
			c <- m
		}
	}()

	return c
}

func (gc *GearConn) SensorRead(name string, startAt, endAt, step int64) <-chan SensorDataValue {
	replyRecv, replySend := libchan.Pipe()
	c := make(chan SensorDataValue, 0)

	req := struct{ SR SensorReadRequest }{
		SensorReadRequest{name, startAt, endAt, step, replySend},
	}

	err := gc.mainChan.Send(req)
	if err != nil {
		close(c)
		return c
	}

	go func() {
		for {
			var m SensorDataValue
			err := replyRecv.Receive(&m)
			if err != nil {
				log.Printf("Error receiving SensorData message: %s", err.Error())
				close(c)
				return
			}
			c <- m
		}
	}()

	return c
}

func (gc *GearConn) DefineMetric(name, unit string, rate bool) error {
	req := struct{ DM DefineMetricRequest }{
		DefineMetricRequest{name, unit, rate},
	}
	return gc.mainChan.Send(req)
}

func (gc *GearConn) SensorSendData(name, metric string) (chan<- SensorDataValue, error) {
	dataRecv, dataSend := libchan.Pipe()
	ackRecv, ackSend := libchan.Pipe()

	req := struct{ SD SensorDataRequest }{
		SensorDataRequest{name, metric, dataRecv, ackSend},
	}

	err := gc.mainChan.Send(req)
	if err != nil {
		return nil, err
	}
	c := make(chan SensorDataValue, 10)

	go func() {
		for {
			var ack AckReply
			err := ackRecv.Receive(&ack)
			if err != nil {
				return
			}
		}
	}()

	go func() {
		for sdv := range c {
			err := dataSend.Send(sdv)
			if err != nil {
				log.Printf("Error sending SensorDataValue message: %s", err.Error())
				return
			}
		}
		dataSend.Close()
	}()

	return c, nil
}

// ===== Helper functions =====

func (gc *GearConn) pinger() {
	for {
		t0 := time.Now()
		if err := doEcho(gc.mainChan, time.Second); err != nil {
			log.Printf("Libchan connection to %s closed due to %s",
				gc.addr, err.Error())
			gc.Close()
			return
		}
		time.Sleep(time.Second - (time.Now().Sub(t0)))
	}
}

func doEcho(sender libchan.Sender, timeout time.Duration) error {
	const txt = "Hello world!"
	replyRecv, replySend := libchan.Pipe()

	req := struct{ ER EchoRequest }{
		EchoRequest{Text: txt, Reply: replySend},
	}

	err := sender.Send(req)
	if err != nil {
		return err
	}

	reply := &EchoReply{}
	err = replyRecv.Receive(reply)
	if err != nil {
		return err
	}
	if reply.Text != txt {
		return fmt.Errorf("echo returned bad message: '%s'", reply.Text)
	}
	return nil
}
