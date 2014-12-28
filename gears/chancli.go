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

func (gc *GearConn) doRequest(req *Request) error {
	replyRecv, replySend := libchan.Pipe()
	req.Reply = replySend

	// send the request
	err := gc.mainChan.Send(req)
	if err != nil {
		replySend.Close()
		return err
	}

	// wait for a reply
	var r Reply
	err = replyRecv.Receive(&r)
	if err != nil {
		return err
	}
	switch r.Code {
	case CodeOK:
		return nil
	case CodeClientError:
		return fmt.Errorf("client error: %s", r.Error)
	case CodeServerError:
		return fmt.Errorf("server error: %s", r.Error)
	case CodeAckTimeout:
		return AckTimeoutError
	default:
		return fmt.Errorf("unknown error type: %s", r.Error)
	}
}

func (gc *GearConn) RFSubscribe(start int64) (<-chan RFMessage, error) {
	subRecv, subSend := libchan.Pipe()
	c := make(chan RFMessage, 0)

	req := Request{
		RFS: &RFSubRequest{StartAt: start, Match: RFMessage{}},
	}

	err := gc.doRequest(&req)
	if err != nil {
		subSend.Close()
		close(c)
		return nil, err
	}

	go func() {
		for {
			var m RFMessage
			err := subRecv.Receive(&m)
			if err != nil {
				log.Printf("Error receiving RF message: %s", err.Error())
				close(c)
				return
			}
			c <- m
		}
	}()

	return c, nil
}

func (gc *GearConn) RFSend(msg RFMessage) error {
	req := Request{RF: (*RFSendRequest)(&msg)}
	return gc.doRequest(&req)
}

func (gc *GearConn) SensorSubscribe(name string, startAt int64) (<-chan SensorDataValue, error) {
	subRecv, subSend := libchan.Pipe()
	c := make(chan SensorDataValue, 0)

	req := Request{SS: &SensorSubRequest{name, startAt, subSend}}
	err := gc.doRequest(&req)
	if err != nil {
		subSend.Close()
		close(c)
		return nil, err
	}

	go func() {
		for {
			var m SensorDataValue
			err := subRecv.Receive(&m)
			if err != nil {
				log.Printf("Error receiving SensorData message: %s", err.Error())
				close(c)
				return
			}
			c <- m
		}
	}()

	return c, nil
}

func (gc *GearConn) SensorRead(name string, startAt, endAt, step int64) (
	<-chan SensorDataValue, error) {
	valuesRecv, valuesSend := libchan.Pipe()
	c := make(chan SensorDataValue, 0)

	req := Request{
		SR: &SensorReadRequest{name, startAt, endAt, step, valuesSend},
	}
	err := gc.doRequest(&req)
	if err != nil {
		valuesSend.Close()
		close(c)
		return nil, err
	}

	go func() {
		for {
			var m SensorDataValue
			err := valuesRecv.Receive(&m)
			if err != nil {
				log.Printf("Error receiving SensorData message: %s", err.Error())
				close(c)
				return
			}
			c <- m
		}
	}()

	return c, nil
}

func (gc *GearConn) SensorSendData(name string, si SensorInfo) (chan<- SensorDataValue, error) {
	dataRecv, dataSend := libchan.Pipe()

	req := Request{SD: &SensorDataRequest{name, si, dataRecv}}
	err := gc.doRequest(&req)
	if err != nil {
		dataSend.Close()
		return nil, err
	}

	c := make(chan SensorDataValue, 10)

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
	txt := "Hello world!"
	replyRecv, replySend := libchan.Pipe()

	req := Request{ER: (*EchoRequest)(&txt), Reply: replySend}

	err := sender.Send(req)
	if err != nil {
		replySend.Close()
		return err
	}

	var reply Reply
	err = replyRecv.Receive(&reply)
	if err != nil {
		return err
	}
	if reply.Code != CodeOK {
		return fmt.Errorf("%s", reply.Error)
	}
	if string(*reply.ER) != txt {
		return fmt.Errorf("echo returned bad message: '%s'", reply.ER)
	}
	return nil
}
