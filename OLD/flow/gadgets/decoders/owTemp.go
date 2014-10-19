package decoders

import (
        "strconv"
        "github.com/golang/glog"
	"github.com/jcw/flow"
)

func init() {
	flow.Registry["Module-OwTemp"] = func() flow.Circuitry { return &OwTemp{} }
}

// Decoder for the OwTemp sketch module.
type OwTemp struct {
	flow.Gadget
	In  flow.Input
	Out flow.Output
}

// Start decoding OwTemp packets.
func (w *OwTemp) Run() {
	for m := range w.In {
                glog.V(2).Infof("OwTemp got %+v", m)

                // byte array, [0]=hdr, [1]=module_id, [2..]=temp
                if v, ok := m.(flow.PacketMap); ok && len(v.Bytes("data")) > 0{
                        readings := map[string]float64{}
                        for i, t := range v.Bytes("data") {
                                name := "temp" + strconv.Itoa(i)
                                if t > 0 {
				        readings[name] = float64(t)
                                }
			}
                        v["readings"] = readings
		}

                glog.V(2).Infof("OwTemp sending %+v", m)
		w.Out.Send(m)
	}
}
