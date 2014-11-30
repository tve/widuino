// Copyright 2014 by Thorsten von Eicken, see LICENSE in top-level directory

package main

import (
	"./database"
	"fmt"
	"net"
	"reflect"
	"sync"

	"github.com/docker/libchan"
	"github.com/docker/libchan/spdy"
	"github.com/golang/glog"
	"github.com/mitchellh/mapstructure"
)

// ===== Request type disctionary

var RequestTypes map[string]reflect.Type = make(map[string]reflect.Type)
var requestTypesLock sync.Mutex

func AddRequestType(name string, t interface{}) {
	requestTypesLock.Lock()
	defer requestTypesLock.Unlock()
	RequestTypes[name] = reflect.TypeOf(t)
}

// ===== Echo Requests

type EchoRequest struct {
	Cmd   string // "echo"
	Text  string
	Reply libchan.Sender
}

func init() { AddRequestType("echo", EchoRequest{}) }

type EchoReply struct {
	Text string
}

func (req *EchoRequest) Handle() error {
	if req.Reply == nil {
		return fmt.Errorf("Echo request has nil reply channel")
	}
	rep := EchoReply{req.Text}
	err := req.Reply.Send(rep)
	if err != nil {
		return err
	}
	req.Reply.Close()
	return nil
}

// ===== Subscription Requests

type SubRequest struct {
	Cmd   string // "sub"
	Start int64
	Match database.RFMessage
	Reply libchan.Sender
}

func init() { AddRequestType("sub", SubRequest{}) }

func (req *SubRequest) Handle() error {
	if req.Reply == nil {
		return fmt.Errorf("Sub request has nil reply channel")
	}
	c := db.Subscribe(req.Start)

	go func() {

		defer req.Reply.Close()
		for m := range c {
			err := req.Reply.Send(m)
			if err != nil {
				db.Unsubscribe(c)
				for _ = range c {
					// drain the channel so the sender doesn't block
				}
				return
			}
		}
		return
	}()
	return nil
}

// ===== Request handling loops

func handleRequest(receiver libchan.Receiver) error {
	req := make(map[string]interface{})
	err := receiver.Receive(&req)
	if err != nil {
		return err
	}
	glog.V(2).Info("Recv: %#v", req)

	cmd, _ := req["Cmd"].(string)
	if t, ok := RequestTypes[cmd]; ok {
		reqStruct := reflect.New(t)
		handler := reqStruct.MethodByName("Handle")
		if !handler.IsValid() {
			return fmt.Errorf("cmd=%s: %s is missing Handle function", cmd, t.Name())
		}
		err := mapstructure.Decode(req, reqStruct.Interface())
		if err != nil {
			return fmt.Errorf("cmd=%s: cannot decode into %s", cmd, t.Name())
		}
		resArr := handler.Call(make([]reflect.Value, 0))
		if len(resArr) != 1 {
			return fmt.Errorf("cmd=%s: %s.handle returned %d results, not 1",
				cmd, t.Name(), len(resArr))
		}
		res := resArr[0]
		if res.IsNil() {
			return nil
		}
		err, ok := res.Interface().(error)
		if !ok {
			return fmt.Errorf("cmd=%s: %s.handle returned non-error type: %#v",
				cmd, t.Name(), res)
		}
		return err
	} else {
		return fmt.Errorf("unknown request: %s", cmd)
	}
}

func handleRequests(t *spdy.Transport) {
	for {
		receiver, err := t.WaitReceiveChannel()
		if err != nil {
			glog.Error(err)
			t.Close()
			glog.Info("Closed libchan transport")
			return
		}
		for {
			err := handleRequest(receiver)
			if err != nil {
				glog.Error(err)
				break
			}
		}
	}
}

func ServeChan(listener net.Listener) {
	tl, err := spdy.NewTransportListener(listener, spdy.NoAuthenticator)
	if err != nil {
		glog.Fatal(err)
	}
	for {
		t, err := tl.AcceptTransport()
		if err != nil {
			glog.Error(err)
			break
		}
		go handleRequests(t)
	}
	glog.Info("Done serving libchan")
}
