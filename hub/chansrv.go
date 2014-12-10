// Copyright 2014 by Thorsten von Eicken, see LICENSE in top-level directory

package main

import (
	"fmt"
	"net"

	"github.com/docker/libchan"
	"github.com/docker/libchan/spdy"
	"github.com/golang/glog"
	"github.com/tve/widuino/gears"
)

// ===== Request type dictionary

/*
var RequestTypes map[string]reflect.Type = make(map[string]reflect.Type)
var requestTypesLock sync.Mutex

func AddRequestType(name string, t interface{}) {
	requestTypesLock.Lock()
	defer requestTypesLock.Unlock()
	RequestTypes[name] = reflect.TypeOf(t)
}

func init() {
	AddRequestType("echo", EchoRequest{})
	AddRequestType("rf-sub", SubRequest{})
}
*/

// ===== Request handling loops

func handleRequest(receiver libchan.Receiver) error {
	var req gears.MainRequest
	err := receiver.Receive(&req)
	if err != nil {
		return err
	}
	glog.V(2).Infof("Recv: %+v", req)

	switch {
	case req.ER != nil:
		return HandleEchoRequest(req.ER)
	case req.RFS != nil:
		return HandleRFSubRequest(req.RFS)
	case req.RF != nil:
		return HandleRFSendRequest(req.RF)
	case req.DM != nil:
		return HandleDefineMetricRequest(req.DM)
	case req.SD != nil:
		return HandleSensorDataRequest(req.SD)
	case req.SR != nil:
		return HandleSensorReadRequest(req.SR)
	case req.SS != nil:
		return HandleSensorSubRequest(req.SS)
	case req.PP != nil:
		return HandleParamPutRequest(req.PP)
	case req.PG != nil:
		return HandleParamGetRequest(req.PG)
	default:
		return fmt.Errorf("Sorry, no handler available for %+v", req)
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
