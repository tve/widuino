// Produce values for a time range of levelDB data, The TimeRange circuit expects
// a Map on its input with the following fields: key (db prefix), start (javascript time),
// end (optional, javascript time), and count (approx number of data points).
// If end is specified, then the values are sent followed by a sync tag and closing of the
// channel. If no end is specified then existing values are sent, followed dby a sync tag,
// followed by new values if and when they come in and can be interpolated to the same step
// as previously produced.

package widuino

import (
	"fmt"
	"strconv"
	"strings"
	//"time"

	"github.com/golang/glog"
	"github.com/jcw/flow"
	"github.com/jcw/jeebus/gadgets/database"
	//"github.com/jcw/jeebus/gadgets"
)

func init() {
	flow.Registry["TimeRange"] = func() flow.Circuitry { return &TimeRange{} }

	/*
		// timeRange returns items from a database range and follows up with updates.
		flow.Registry["TimeRange"] = func() flow.Circuitry {
			c := flow.NewCircuit()
			c.Add("trr", "TimeRangeRPC")
			c.Add("sub", "DataSub")
			c.Add("db", "LevelDB")
			c.Add("ti", "TimeInterpolate")
			c.Connect("trr.SubOut:sub", "sub.In", 0)
			c.Connect("trr.DBOut:tag", "db.In", 0)
			c.Connect("db.Out", "ti.DB", 100)
			c.Connect("sub.Out", "ti.Sub", 100) // with buffering
			c.Label("In", "trr.In")
			c.Label("Out", "ti.Out")
			return c
		}
	*/
}

// Receive the Map that specifies the parameters and output the right shtuff for the
// LevelDB gadget and the DataSub gadget
type TimeRange struct {
	flow.Gadget
	In  flow.Input  // expects PacketMap with key, start, end, count fields
	Out flow.Output // outputs to drive the DataSub gadget
}

func (g *TimeRange) Run() {
	// get the parameters
	in := <-g.In
	inMap, ok := in.(map[string]interface{})
	if !ok {
		glog.Infof("TimeRangeRPC expected PacketMap received %T", in)
		return
	}
	params := flow.PacketMap(inMap)
	//glog.V(4).Infof("TimeRange got %#v", params)
	glog.Info("**************************************************************")

	// parse the params
	key := params.String("key")
	startT := params.Int64("start")
	endT := params.Int64("end")
	count := params.Int64("count")
	glog.Infof("TimeRange for %s from %d to %d count %d", key, startT, endT, count)

	// input validation
	switch {
	case endT <= startT:
		glog.Error("TimeRange: error end<=start")
		return
	case count < 3:
		glog.Error("TimeRange: count must be > 2")
		return
	case (endT - startT) < count:
		glog.Error("TimeRange: 1 millisecond step minimum")
		return
	}
	step := (endT - startT) / count // round-down...
	// ensure we get enough data to interpolate
	startT -= step
	endT += step

	// Retrieve the data
	type timeData struct {
		t int64
		v float64
	}
	data := make([]timeData, 0)
	database.DbIterateOverKeys(fmt.Sprintf("%s/%d", key, startT),
		fmt.Sprintf("%s/%d", key, endT),
		func(k string, v []byte) {
			if !strings.HasPrefix(k, key+"/") {
				glog.Warning("TimeRange: huh? k=", k)
				return
			}
			t := strings.TrimPrefix(k, key+"/")
			time, err := strconv.ParseInt(t, 10, 64)
			if err != nil || time < startT || time > endT {
				glog.Warning("TimeRange: huh? t=", t)
				return
			}
			val, err := strconv.ParseFloat(string(v), 64)
			if err != nil {
				glog.Warning("TimeRange: huh? v=", v)
				return
			}
			data = append(data, timeData{time, val})
		})

	// TODO: Interpolate

	// Output what we've got
	g.Out.Send(flow.Tag{"<range>", key})
	for _, d := range data {
		g.Out.Send(d)
	}
	g.Out.Send(flow.Tag{"<sync>", ""})
}
