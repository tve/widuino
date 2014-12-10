// Copyright 2014 by Thorsten von Eicken, see LICENSE in top-level directory

package main

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/tve/widuino/gears"
)

func HandleEchoRequest(req *gears.EchoRequest) error {
	if req.Reply == nil {
		return fmt.Errorf("Echo request has nil reply channel")
	}
	rep := gears.EchoReply{req.Text}
	err := req.Reply.Send(rep)
	if err != nil {
		return err
	}
	req.Reply.Close()
	return nil
}

// ===== RF Message Requests

func HandleRFSubRequest(req *gears.RFSubRequest) error {
	if req.Reply == nil {
		return fmt.Errorf("Sub request has nil reply channel")
	}
	c := db.Subscribe(req.StartAt)
	dt := req.StartAt/1000 - time.Now().Unix()
	if req.StartAt <= 0 {
		dt = 0
	}
	glog.Infof("StartAt subscriber %v at now%+ds", c, dt)

	go func() {
		defer req.Reply.Close()
		for m := range c {
			glog.V(2).Infof("Sending to %v: %+v", c, m)
			err := req.Reply.Send(m)
			if err != nil {
				db.Unsubscribe(c)
				req.Reply.Close()
				glog.Infof("Closed subscriber %v due to error: %s", c, err.Error())
				for _ = range c {
					// drain the channel so the sender doesn't block
				}
				return
			}
		}
		req.Reply.Close()
		glog.Infof("Closed subscriber %v due to incoming EOF", c)
		return
	}()
	return nil
}

func HandleRFSendRequest(req *gears.RFSendRequest) error {
	xmitChan <- req.Msg
	if req.Reply == nil {
		return nil
	}
	req.Reply.Send(gears.AckReply{""})
	return nil
}

//  Metric Requests

func HandleDefineMetricRequest(req *gears.DefineMetricRequest) error {
	return nil
}

// Sensor Data Requests

func HandleSensorSubRequest(req *gears.SensorSubRequest) error {
	return nil
}

func HandleSensorDataRequest(req *gears.SensorDataRequest) error {
	return nil
}

func HandleSensorReadRequest(req *gears.SensorReadRequest) error {
	return nil
}

// Params Requests

func HandleParamPutRequest(req *gears.ParamPutRequest) error {
	return nil
}

func HandleParamGetRequest(req *gears.ParamGetRequest) error {
	return nil
}
