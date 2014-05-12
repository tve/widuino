// Copyright 2014 by Thorsten von Eicken, see LICENSE in top-level directory

// Modules are pieces of code that are loaded into sketches to provide functionality.
// For example, a module may read 1-wire temperature sensors, another may control a servo motor
// yet another a relay. Each sketch module has a unique integer ID that is used in the first
// byte of all packets in order to facilitate dispatch. This dispatch happens both when sending
// messages node-to-server but also when sending command server-to-node.
// This package contains a gadget to annotate the packet with the module ID, a map gadget
// to map numeric IDs into names, a dispatch gadget to dispatch to an appropriate decoder.

package widuino

import (
        "strconv"
        "strings"
        "github.com/golang/glog"
        "github.com/jcw/flow"
)

func init() {
        flow.Registry["ModuleIdent"] = func() flow.Circuitry { return &ModuleIdent{} }
        flow.Registry["ModuleMap"] = func() flow.Circuitry { return &ModuleMap{} }
}

//===== ModuleIdent =====

// Identify the module based on the first byte in the raw payload
// Registers as "ModuleIdent"
type ModuleIdent struct {
        flow.Gadget
        In   flow.Input         // Expects PacketMaps as input
        Out  flow.Output        // Produces PacketMaps with "module": int added
}

func (g *ModuleIdent) Run() {
        for m := range g.In {
                if v, ok := m.(flow.PacketMap); ok {
                        v["module"] = int(v.Bytes("raw")[1])
                        v["data"] = v.Bytes("raw")[2:]
                        g.Out.Send(m)
                } else {
                        g.Out.Send(m)
                }
        }
}

//===== ModuleMap =====

// Map the module ID to a module name to facilitate dispatch
// Registers as "ModuleMap".
type ModuleMap struct {
	flow.Gadget
	Info flow.Input         // Expects strings with "id,name"
	In   flow.Input         // Expects PacketMaps with "module":int field
	Out  flow.Output        // Produces PacketMaps with "module_name":string added
}

// Start looking up modules
func (w *ModuleMap) Run() {
	moduleMap := map[int]string{}
	for m := range w.Info {
		f := strings.Split(m.(string), ",")
                id, _ := strconv.Atoi(f[0])
		moduleMap[id] = f[1]
	}

	for m := range w.In {
                if v, ok := m.(flow.PacketMap); ok {
                        id := v.Int("module")
                        if name, ok := moduleMap[id]; ok {
                                v["module_name"] = name
                        } else {
                                glog.Warningf("Unknown module %d in packet %+v", id, m)
                        }
                        w.Out.Send(m)
                } else {
                        w.Out.Send(m)
                }
        }
}
