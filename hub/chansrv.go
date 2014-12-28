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

// ===== Request handling loops

// ServeChan accepts connections and starts-up a handler goroutine for each one
func ServeChan(listener net.Listener) {
	tl, err := spdy.NewTransportListener(listener, spdy.NoAuthenticator)
	if err != nil {
		glog.Fatal(err)
	}
	defer func() {
		tl.Close()
		glog.Info("Done serving libchan connections")
	}()
	for {
		t, err := tl.AcceptTransport()
		if err != nil {
			glog.Error(err)
			return
		}
		go handleRequests(t)
	}
}

// handleRequests expects to receive a main channel and then reads and handles requests
// off of that
func handleRequests(t *spdy.Transport) {
	defer func() {
		t.Close()
		glog.Info("Closed libchan connection")
	}()
	// wait to receive the main channel on which the client will send requests
	receiver, err := t.WaitReceiveChannel()
	if err != nil {
		glog.Error(err)
		return
	}
	// request handling loop
	for {
		err := handleRequest(receiver)
		if err != nil {
			glog.Error(err)
			return
		}
	}
}

// handle one request, errors that are returned are deemed fatal and should cause the
// connection to be closed
func handleRequest(receiver libchan.Receiver) error {
	// receive a request
	var req gears.Request
	err := receiver.Receive(&req)
	if err != nil {
		return err
	}
	glog.V(2).Infof("Recv: %+v", req)

	// check that we have a reply channel, if it's missing we treat this as a fatal
	// error in order to signal to the client that something is very wrong here
	if req.Reply == nil {
		return fmt.Errorf("request is missing reply channel")
	}

	var rep gears.Reply
	switch {
	case req.ER != nil:
		rep = HandleEchoRequest(req.ER)
	case req.RFS != nil:
		rep = HandleRFSubRequest(req.RFS)
	case req.RF != nil:
		rep = HandleRFSendRequest(req.RF)
	case req.SI != nil:
		rep = HandleSensorInfoRequest(req.SI)
	case req.SD != nil:
		rep = HandleSensorDataRequest(req.SD)
	case req.SR != nil:
		rep = HandleSensorReadRequest(req.SR)
	case req.SS != nil:
		rep = HandleSensorSubRequest(req.SS)
	case req.PP != nil:
		rep = HandleParamPutRequest(req.PP)
	case req.PG != nil:
		rep = HandleParamGetRequest(req.PG)
	default:
		rep = gears.Reply{
			Code:  gears.CodeClientError,
			Error: fmt.Sprintf("unknown request: %+v", req),
		}
	}
	err = req.Reply.Send(rep)
	if err != nil {
		return fmt.Errorf("cannot send reply: %s", err.Error())
	}
	if rep.Code != gears.CodeOK {
		// we don't return an error 'cause this is not fatal and we don't
		// want to close the connection (maybe we should for CodeServerError)
		// but we can also leave that decision up to the client and thereby
		// avoid killing pipelined incoming requests
		glog.Infof("Request handler error: %s", err.Error())
	}
	return nil
}
