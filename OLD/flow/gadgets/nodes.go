// Driver and decoders for RF12/RF69 packet data.
package widuino

import (
	"fmt"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/jcw/flow"
	"github.com/jcw/jeebus/gadgets"
)

func init() {
	flow.Registry["NodeMapM"] = func() flow.Circuitry { return &NodeMap{} }
	flow.Registry["PutReadingsM"] = func() flow.Circuitry { return &PutReadings{} }
	flow.Registry["SplitReadingsM"] = func() flow.Circuitry { return &SplitReadings{} }
}

// Lookup the group/node information to determine what sketch the node is running.
// The sketch name is written into to a sketch:string field, and the location into a
// location:string field.
// If the node is not in the map, the packet is sent to the reject output to reduce
// noise from bogus packets.
// Registers as "NodeMap".
type NodeMap struct {
	flow.Gadget
	Info flow.Input  // expects strings of the form RFg00i00,<location>
	In   flow.Input  // expects PacketMaps with group:int and node:int fields
	Out  flow.Output // outputs PacketMaps with added sketch:string and location:string
	Rej  flow.Output // outputs packets from nodes not found in the map
}

// Start looking up node ID's in the node map.
func (g *NodeMap) Run() {
	nodeMap := map[string]string{}
	locations := map[string]string{}
	for m := range g.Info {
		f := strings.Split(m.(string), ",")
		nodeMap[f[0]] = f[1]
		if len(f) > 2 {
			locations[f[0]] = f[2]
		}
	}

	for m := range g.In {
		if pm, ok := m.(flow.PacketMap); ok {
			key := fmt.Sprintf("RFg%di%d", pm.Int("group"), pm.Int("node"))
			if loc, ok := locations[key]; ok {
				pm["location"] = loc
			}
			if sketch, ok := nodeMap[key]; ok {
				pm["sketch"] = sketch
				g.Out.Send(m)
				continue
			}
		}
		g.Rej.Send(m)
	}
}

// Save readings in database.
type PutReadings struct {
	flow.Gadget
	In  flow.Input
	Out flow.Output
}

// Convert each loosely structured reading object into a strict map for storage.
func (g *PutReadings) Run() {
	glog.V(2).Info("Starting PacketMap based PutReadings")
	for m := range g.In {
		pm, ok := m.(flow.PacketMap)
		if !ok {
			continue
		}
		if _, ok := pm["readings"]; !ok {
			continue
		}

		values := pm["readings"].(map[string]float64)
		if rssi, ok := pm["rssi"]; ok {
			values["rssi"] = float64(rssi.(int))
		}
		asof := time.Now()
		if _, ok := pm["asof"]; ok {
			asof = pm.Time("asof")
		}

		id := pm.String("rf12")
		data := map[string]interface{}{
			"ms":  jeebus.TimeToMs(asof),
			"val": values,
			"loc": pm.String("location"),
			"typ": pm.String("decoder"),
			"id":  id,
		}
		glog.V(2).Infof("PutReading /reading/%s: %+v", id, data)
		g.Out.Send(flow.Tag{"/reading/" + id, data})
	}
}

// Split reading data into individual values.
type SplitReadings struct {
	flow.Gadget
	In  flow.Input
	Out flow.Output
}

// Split combined readings into separate sensor values, for separate storage.
func (g *SplitReadings) Run() {
	for m := range g.In {
		data := m.(flow.Tag).Msg.(map[string]interface{})
		for k, v := range data["val"].(map[string]float64) {
			key := fmt.Sprintf("sensor/%s/%s/%d",
				data["loc"].(string), k, data["ms"].(int64))
			g.Out.Send(flow.Tag{key, v})
		}
	}
}
