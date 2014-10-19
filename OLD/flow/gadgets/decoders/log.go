package decoders

import (
        "strings"
        "github.com/golang/glog"
	"github.com/jcw/flow"
)

func init() {
	flow.Registry["Module-Log"] = func() flow.Circuitry { return &Log{} }
}

// Log prints messages it receives into the log, this is used to print debug/logging information
// within sketches. It does not output anything, i.e., the log messages disappear.
type Log struct {
	flow.Gadget
	In flow.Input
	Out flow.Output
}

// Start logging incoming messages.
func (w *Log) Run() {
	for m := range w.In {
                if pm, ok := m.(flow.PacketMap); ok {
                        txt := string(pm.Bytes("data"))
                        txt = strings.Trim(txt, " \t\n\r")
                        glog.Infof("LOG g%di%d: %s\n", pm.Int("group"), pm.Int("node"), txt)
                }
		//w.Out.Send(m)
	}
}
