// Copyright 2014 by Thorsten von Eicken, see LICENSE in top-level directory

package main

import (
	"io"
	"time"

	"github.com/golang/glog"
	"github.com/tve/widuino/gears"
)

func HandleEchoRequest(req *gears.EchoRequest) gears.Reply {
	return gears.Reply{Code: gears.CodeOK, ER: (*gears.EchoReply)(req)}
}

// ===== RF Message Requests

// subscribe to RF messages
func HandleRFSubRequest(req *gears.RFSubRequest) gears.Reply {
	if req.Messages == nil {
		return gears.Reply{Code: gears.CodeClientError, Error: "Messages channel is nil"}
	}
	c := db.RFSubscribe(req.StartAt)
	dt := req.StartAt/1000 - time.Now().Unix()
	if req.StartAt <= 0 {
		dt = 0
	}
	glog.Infof("Start RF subscriber %v at now%+dsecs", c, dt)

	// goroutine that will actually stream the subscription data
	go func() {
		defer req.Messages.Close()
		for m := range c {
			glog.V(2).Infof("Sending to %v: %+v", c, m)
			err := req.Messages.Send(m)
			if err != nil {
				db.RFUnsubscribe(c)
				glog.Infof("Closing subscriber %v due to error: %s", c, err.Error())
				for _ = range c {
					// drain the channel so the sender doesn't block
				}
				return
			}
		}
		glog.Infof("Closed subscriber %v due to incoming EOF", c)
		return
	}()

	return gears.Reply{Code: gears.CodeOK}
}

func HandleRFSendRequest(req *gears.RFSendRequest) gears.Reply {
	xmitChan <- gears.RFMessage(*req)
	return gears.Reply{Code: gears.CodeOK}
}

// Sensor Requests

func HandleSensorInfoRequest(req *gears.SensorInfoRequest) gears.Reply {
	info, err := db.GetSensorInfo(req.Name)
	if err != nil {
		return gears.Reply{Code: gears.CodeClientError, Error: err.Error()}
	}

	return gears.Reply{Code: gears.CodeOK, SI: &info}
}

func HandleSensorSubRequest(req *gears.SensorSubRequest) gears.Reply {
	if req.Values == nil {
		return gears.Reply{Code: gears.CodeClientError, Error: "Values channel is nil"}
	}
	c := db.SensorSubscribe(req.Name, req.StartAt)
	dt := req.StartAt/1000 - time.Now().Unix()
	if req.StartAt <= 0 {
		dt = 0
	}
	glog.Infof("Start sensor subscriber %v at now%+dsecs", c, dt)

	go func() {
		defer req.Values.Close()
		for m := range c {
			glog.V(2).Infof("Sending to %v: %+v", c, m)
			err := req.Values.Send(m)
			if err != nil {
				db.SensorUnsubscribe(c)
				glog.Infof("Closing subscriber %v due to error: %s", c, err.Error())
				for _ = range c {
					// drain the channel so the sender doesn't block
				}
				return
			}
		}
		glog.Infof("Closed subscriber %v due to incoming EOF", c)
		return
	}()
	return gears.Reply{Code: gears.CodeOK}
}

func HandleSensorDataRequest(req *gears.SensorDataRequest) gears.Reply {
	glog.Infof("Start sensor data push for %s", req.Name)
	if req.Values == nil {
		return gears.Reply{Code: gears.CodeClientError, Error: "Values channel is nil"}
	}

	// TODO: handle req.Info !!!

	go func() {
		for {
			var m gears.SensorDataValue
			err := req.Values.Receive(&m)
			if err == io.EOF {
				glog.Infof("EOF reading sensor data values for %s", req.Name)
				return
			} else if err != nil {
				glog.Warningf("Error reading sensor data values for %s: %s",
					req.Name, err.Error())
				return
			}

			db.PutSensorValue(req.Name, m)
		}
	}()
	return gears.Reply{Code: gears.CodeOK}
}

func HandleSensorReadRequest(req *gears.SensorReadRequest) gears.Reply {
	glog.Infof("Start sensor read of %s start=%d end=%d step=%d",
		req.Name, req.StartAt/1000-time.Now().Unix(),
		req.EndAt/1000-time.Now().Unix(), req.Step)
	return gears.Reply{Code: gears.CodeServerError, Error: "Not yet implemented"}
}

// Params Requests

func HandleParamPutRequest(req *gears.ParamPutRequest) gears.Reply {
	return gears.Reply{Code: gears.CodeServerError, Error: "Not yet implemented"}
}

func HandleParamGetRequest(req *gears.ParamGetRequest) gears.Reply {
	return gears.Reply{Code: gears.CodeServerError, Error: "Not yet implemented"}
}
