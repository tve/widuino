// Statstics utilities for HouseMon.
package stats

import (
	"strconv"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/jcw/flow"
)

func init() {
	flow.Registry["Aggregator"] = func() flow.Circuitry { return new(Aggregator) }
}

// Aggregate sensor values into bins, with min/max/avg/stddev calculation.
type Aggregator struct {
	flow.Gadget
	Step flow.Input
	In   flow.Input
	Out  flow.Output

	step   string
	stepMs int64
	stats  map[string]accumulator
}

type accumulator struct {
	Num  int `json:"n"`
	Low  int `json:"l"`
	High int `json:"h"`
	// StdDev int   `json:"d"`
	Sum int64 `json:"s"`

	// m2  int64
	slot int
}

// Start collecting values, emit aggregated results when a time step occurs.
func (g *Aggregator) Run() {
	g.stats = map[string]accumulator{}

	g.step = "1h"
	if s, ok := <-g.Step; ok {
		g.step = s.(string)
	}
	d, err := time.ParseDuration(g.step)
	flow.Check(err)
	g.stepMs = d.Nanoseconds() / 1e6

	// collect data and aggregate for each parameter
	for m := range g.In {
		if t, ok := m.(flow.Tag); ok {
			n := strings.LastIndex(t.Tag, "/")
			// expects input tags like these:
			// 	sensor/meterkast/c3/1396556362024 = 2396
			if n > 0 {
				prefix := t.Tag[:n+1]
				ms, err := strconv.ParseInt(t.Tag[n+1:], 10, 64)
				flow.Check(err)
				g.process(prefix, ms, t.Msg.(int))
			}
		}
	}

	for k := range g.stats {
		g.flush(k)
	}
}

func (g *Aggregator) process(prefix string, ms int64, val int) {
	slot := int(ms / g.stepMs)
	accum, ok := g.stats[prefix]
	if !ok || slot != accum.slot {
		if ok {
			g.flush(prefix)
		}
		accum = accumulator{slot: slot, Low: val, High: val}
	}
	accum.Num++
	accum.Sum += int64(val)
	if val < accum.Low {
		accum.Low = val
	}
	if val > accum.High {
		accum.High = val
	}
	g.stats[prefix] = accum
}

func (g *Aggregator) flush(prefix string) {
	// tags sent out look like this, once converted to JSON:
	// 	aggregate/meterkast/p3/1m/23275939
	//	 = {"n":3,"l":2375,"h":2401,"s":7172}
	accum := g.stats[prefix]
	glog.Infoln("flush", prefix, accum)
	n := strings.Index(prefix, "/")
	key := "aggregate" + prefix[n:] + g.step + "/" + strconv.Itoa(accum.slot)
	g.Out.Send(flow.Tag{key, accum})
}
