// Copyright 2014 by Thorsten von Eicken, see LICENSE in top-level directory

package main

import (
	"fmt"
	"io"
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
	c := db.RFSubscribe(req.StartAt)
	dt := req.StartAt/1000 - time.Now().Unix()
	if req.StartAt <= 0 {
		dt = 0
	}
	glog.Infof("Start RF subscriber %v at now%+ds", c, dt)

	go func() {
		defer req.Reply.Close()
		for m := range c {
			glog.V(2).Infof("Sending to %v: %+v", c, m)
			err := req.Reply.Send(m)
			if err != nil {
				db.RFUnsubscribe(c)
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

// Sensor Requests

func HandleSensorInfoRequest(req *gears.SensorInfoRequest) error {
	if req.Reply == nil {
		return fmt.Errorf("Sensor info request has nil reply channel")
	}
	info, err := db.GetSensorInfo(req.Name)
	if err != nil {

	info.Metric = db.Get(fmt.Sprintf("/sens/%s/metric", req.Name), &info.Metric)
	info.Unit = db.Get(fmt.Sprintf("/metric/%s/unit", info.Metric))
	info.Rate = db.Get(fmt.Sprintf("/metric/%s/rate", info.Metric)) == "true"
	err := req.Reply.Send(info)
	return nil
}

func HandleSensorSubRequest(req *gears.SensorSubRequest) error {
	if req.Reply == nil {
		return fmt.Errorf("Sensor sub request has nil reply channel")
	}
	c := db.SensorSubscribe(req.Name, req.StartAt)
	dt := req.StartAt/1000 - time.Now().Unix()
	if req.StartAt <= 0 {
		dt = 0
	}
	glog.Infof("Start sensor subscriber %v at now%+ds", c, dt)

	go func() {
		defer req.Reply.Close()
		for m := range c {
			glog.V(2).Infof("Sending to %v: %+v", c, m)
			err := req.Reply.Send(m)
			if err != nil {
				db.SensorUnsubscribe(c)
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

func HandleSensorDataRequest(req *gears.SensorDataRequest) error {
	if req.Values == nil {
		return fmt.Errorf("Sensor data request has nil values channel")
	}

	glog.Infof("Start sensor data push for %s metric %s", req.Name, req.Metric)

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
