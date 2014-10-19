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
	"github.com/tve/widuino/interpol8"
	"math"
	"strconv"
	"strings"

	"github.com/golang/glog"
	"github.com/jcw/flow"
	"github.com/jcw/jeebus/gadgets/database"
	//"github.com/jcw/jeebus/gadgets"
)

const fillFct = 4 // intervals to interpolate over

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
	startT := params.Uint64("start")
	endT := params.Uint64("end")
	step := params.Uint64("step")
	glog.Infof("TimeRange for %s from %d to %d by %d", key, startT, endT, step)

	// input validation
	switch {
	case endT <= startT:
		glog.Error("TimeRange: error end<=start")
		return
	case step < 2:
		glog.Error("TimeRange: step must be > 1")
		return
	}
	count := (endT - startT) / step // round-down...

	// Retrieve the data TODO: need to grab maxFill*step extra data
	raw := make([]interpol8.RawPoint, 0, count)
	database.DbIterateOverKeys(fmt.Sprintf("%s/%d", key, startT-step),
		fmt.Sprintf("%s/%d", key, endT+step),
		func(k string, v []byte) {
			if !strings.HasPrefix(k, key+"/") {
				glog.Warning("TimeRange: huh? k=", k)
				return
			}
			t := strings.TrimPrefix(k, key+"/")
			time, err := strconv.ParseUint(t, 10, 64)
			if err != nil || time < startT || time > endT {
				glog.Warning("TimeRange: huh? t=", t)
				return
			}
			val, err := strconv.ParseFloat(string(v), 64)
			if err != nil {
				glog.Warning("TimeRange: huh? v=", v)
				return
			}
			raw = append(raw, interpol8.RawPoint{time, val})
		})

	// TODO: Interpolate
	data, err := interpol8.Raw(raw, interpol8.Absolute, startT, endT, step, fillFct*step)
	if err != nil {
		glog.Error("TimeRange: ", err)
		return
	}

	// Convert NaN to null so we can output to JSON. Huh? Yup, it's a great hack!
	type PointOrNaN struct {
		Asof          uint64
		Avg, Min, Max interface{} // either float64 or nil
	}
	dataHack := make([]PointOrNaN, len(data))
	for i := range data {
		dataHack[i].Asof = data[i].Asof
		if math.IsNaN(data[i].Avg) { // if Avg is NaN then Min&Max also
			dataHack[i].Avg = nil
			dataHack[i].Min = nil
			dataHack[i].Max = nil
		} else {
			dataHack[i].Avg = data[i].Avg
			dataHack[i].Min = data[i].Min
			dataHack[i].Max = data[i].Max
		}
	}

	// Output what we've got
	g.Out.Send(flow.Tag{"<range>", key})
	for _, d := range dataHack {
		g.Out.Send(d)
	}
	g.Out.Send(flow.Tag{"<sync>", ""})
}
