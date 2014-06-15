#!/usr/bin/env coffee

circuits = {}

# init circuit for HouseMon, which starts its own http server.
circuits.init =
  gadgets: [
    #{ name: "mqtt", type: "MQTTServer" }
    #{ name: "sub", type: "DataSub" }
    #{ name: "pub", type: "MQTTPub" }
    { name: "dummy", type: "Pipe" } # needed for dispatcher in HouseMon
    { name: "driverFill", type: "driverFill" } # pre-load the database
    { name: "tableFill", type: "tableFill" }   # pre-load the database
    { name: "sub2", type: "DataSub" }
    { name: "aggr", type: "Aggregator" }
    { name: "db", type: "LevelDB" }
    { name: "udpgw", type: "udpgw" }
  ]
  wires: [
    #{ from: "mqtt.PortOut", to: "pub.Port" }
    #{ from: "sub.Out", to: "pub.In" }
    { from: "sub2.Out", to: "aggr.In" }
    { from: "aggr.Out", to: "db.In" }
  ]
  feeds: [
    #{ data: ":1883",  to: "mqtt.Port" }
    #{ data: "/",  to: "sub.In" }
    { data: "sensor/",  to: "sub2.In" }
    # { data: "1m",  to: "aggr.Step" }
  ]
  labels: [
    { external: "In", internal: "dummy.In" }
    { external: "Out", internal: "dummy.Out" }
  ]

# define the websocket handler using a loop in and out of RpcHandler
circuits["WebSocket-jeebus"] =
  gadgets: [
    { name: "rpc", type: "RpcHandler" }
  ]
  labels: [
    { external: "In", internal: "rpc.In" }
    { external: "Out", internal: "rpc.Out" }
  ]

# UDP Gateway
circuits.udpgw =
  gadgets: [
    { name: "udp", type: "UDP-Gateway" }
    { name: "f1", type: "FanOut" }
    { name: "lg", type: "Logger" }
    { name: "db", type: "rf12toDatabase" }
    { name: "jb", type: "Booter" }
  ]
  wires: [
    { from: "udp.Recv", to: "f1.In" }
    { from: "f1.Out:lg", to: "lg.In" }
    { from: "f1.Out:db", to: "db.In" }
    { from: "udp.Oob", to: "jb.In" }
    { from: "jb.Out", to: "udp.Xmit" }
  ]
  feeds: [
    { data: 9999, to: "udp.Port" }
    { data: "./logger", to: "lg.Dir" }
  ]

# booter circuit ready to hook up to rf12 look-alike OOb (e.g. UDP-Gateway)
circuits["Booter"] =
  gadgets: [
    { name: "jb", type: "JeeBoot" }
    { name: "wf", type: "WatchFile" }
    { name: "cf", type: "ReadFileJSON" }
    { name: "rd", type: "ReadFileText" }
    { name: "hx", type: "IntelHexToBin" }
    { name: "bf", type: "BinaryFill" }
    { name: "cs", type: "CalcCrc16" }
    { name: "bd", type: "BootData" }
  ]
  wires: [
    { from: "cf.Out", to: "jb.Cfg", capacity: 10 }
    { from: "jb.Files", to: "wf.In", capacity: 10 }
    { from: "wf.Out", to: "rd.In", capacity: 10 }
    { from: "rd.Out", to: "hx.In", capacity: 100 }
    { from: "hx.Out", to: "bf.In", capacity: 100 }
    { from: "bf.Out", to: "cs.In", capacity: 100 }
    { from: "cs.Out", to: "bd.In", capacity: 100 }
  ]
  labels: [
    { external: "In", internal: "jb.In" }
    { external: "Out", internal: "jb.Out" }
  ]
  feeds: [
    { data: "jeeboot.json", to: "cf.In" }
    { data: 64, to: "bf.Len" }
  ]

# the node mapping for nodes at WCC, as pre-configured circuit
circuits.nodesWCC =
  gadgets: [
    { name: "nm", type: "NodeMapM" }
    { name: "di", type: "PacketMapDispatcher" }
  ]
  feeds: [
    { data: "sketch",                  to: "di.Field" }
    { data: "Sketch-",                 to: "di.Prefix" }
    { data: "RFg212i14,tempNode,desk", to: "nm.Info" }
  ]
  wires: [
    { from: "nm.Out", to: "di.In" }
  ]
  labels: [
    { external: "In",  internal: "nm.In" }
    { external: "Out", internal: "di.Out" }
  ]

# the modules mapping, as pre-configured circuit -- a module is a piece of a sketch
circuits.modules =
  gadgets: [
    { name: "mi", type: "ModuleIdent" } # extract module ID from packet
    { name: "mm", type: "ModuleMap" }   # map module ID to module name
    { name: "di", type: "PacketMapDispatcher" }
  ]
  wires: [
    { from: "mi.Out", to: "mm.In" }
    { from: "mm.Out", to: "di.In" }
  ]
  feeds: [
    { data: "1,Net",          to: "mm.Info" }
    { data: "2,Log",          to: "mm.Info" }
    { data: "4,OwTemp",       to: "mm.Info" }
    { data: "Module-",        to: "di.Prefix" }
    { data: "module_name",    to: "di.Field" }
  ]
  labels: [
    { external: "In",  internal: "mi.In" }
    { external: "Out", internal: "di.Out" }
  ]

# pipeline used for decoding RF12demo data and storing it in the database
circuits.rf12toDatabase =
  gadgets: [
    #{ name: "d1", type: "DebugLog" }
    { name: "ni", type: "nodesWCC" }
    { name: "mm", type: "modules" }
    { name: "dd", type: "DebugLog" }
    { name: "ss", type: "PutReadingsM" }
    { name: "f2", type: "FanOut" }
    { name: "sr", type: "SplitReadingsM" }
    { name: "db", type: "LevelDB" }
    { name: "d2", type: "DebugLog" }
  ]
  wires: [
    #{ from: "d1.Out", to: "ni.In" }
    { from: "ni.Out", to: "mm.In" }
    { from: "mm.Out", to: "dd.In" }
    { from: "dd.Out", to: "ss.In" }
    { from: "ss.Out", to: "f2.In" }
    { from: "f2.Out:sr", to: "sr.In" }
    { from: "f2.Out:db", to: "d2.In" }
    { from: "sr.Out", to: "d2.In" }
    { from: "d2.Out", to: "db.In" }
  ]
  labels: [
    { external: "In", internal: "ni.In" }
  ]

# jeeboot server test
#circuits.jeeboot =
#  gadgets: [
#    { name: "sp", type: "SerialPort" }
#    { name: "rf", type: "Sketch-RF12demo" }
#    { name: "sk", type: "Sink" }
#    { name: "jb", type: "JeeBoot" }
#  ]
#  wires: [
#    { from: "sp.From", to: "rf.In" }
#    { from: "rf.Out", to: "sk.In" }
#    { from: "rf.Rej", to: "sk.In" }
#    { from: "rf.Oob", to: "jb.In" }
#    { from: "jb.Out", to: "sp.To" }
#  ]
#  feeds: [
#    { data: "/dev/tty.usbserial-A901ROSM", to: "sp.Port" }
#  ]
  
# simple never-ending demo
circuits.demo =
  gadgets: [
    { name: "c", type: "Clock" }
  ]
  feeds: [
    { data: "1s", to: "c.Rate" }
  ]
  
# pre-load some driver info into the database
circuits.driverFill =
  gadgets: [
    { name: "db", type: "LevelDB" }
  ]
  feeds: [
    { to: "db.In", tag: "/driver/tempNode/temp", \
      data: { name: "Temperature", unit: "°F", scale: 1 } }

#    { to: "db.In", tag: "/driver/roomNode/temp", \
#      data: { name: "Temperature", unit: "°C", scale: 1 } }
#    { to: "db.In", tag: "/driver/roomNode/humi", \
#      data: { name: "Humidity", unit: "%" } }
#    { to: "db.In", tag: "/driver/roomNode/light", \
#      data: { name: "Light intensity", unit: "%", factor: 0.392, scale: 0 } }
#    { to: "db.In", tag: "/driver/roomNode/moved", \
#      data: { name: "Motion", unit: "(0/1)" } }
#      
#    { to: "db.In", tag: "/driver/smaRelay/yield", \
#      data: { name: "PV daily yield", unit: "kWh", scale: 3 } }
#    { to: "db.In", tag: "/driver/smaRelay/dcv1", \
#      data: { name: "PV level east", unit: "V", scale: 2 } }
#    { to: "db.In", tag: "/driver/smaRelay/dcv2", \
#      data: { name: "PV level west", unit: "V", scale: 2 } }
#    { to: "db.In", tag: "/driver/smaRelay/acw", \
#      data: { name: "PV power AC", unit: "W" } }
#    { to: "db.In", tag: "/driver/smaRelay/dcw1", \
#      data: { name: "PV power east", unit: "W" } }
#    { to: "db.In", tag: "/driver/smaRelay/dcw2", \
#      data: { name: "PV power west", unit: "W" } }
#    { to: "db.In", tag: "/driver/smaRelay/total", \
#      data: { name: "PV total", unit: "MWh", scale: 3 } }
#      
#    { to: "db.In", tag: "/driver/homePower/c1", \
#      data: { name: "Counter stove", unit: "kWh", factor: 0.5, scale: 3 } }
#    { to: "db.In", tag: "/driver/homePower/p1", \
#      data: { name: "Usage stove", unit: "W", scale: 1 } }
#    { to: "db.In", tag: "/driver/homePower/c2", \
#      data: { name: "Counter solar", unit: "kWh", factor: 0.5, scale: 3 } }
#    { to: "db.In", tag: "/driver/homePower/p2", \
#      data: { name: "Production solar", unit: "W", scale: 1 } }
#    { to: "db.In", tag: "/driver/homePower/c3", \
#      data: { name: "Counter house", unit: "kWh", factor: 0.5, scale: 3 } }
#    { to: "db.In", tag: "/driver/homePower/p3", \
#      data: { name: "Usage house", unit: "W", scale: 1 } }
  ]
  
# pre-load some table info into the database
circuits.tableFill =
  gadgets: [
    { name: "db", type: "LevelDB" }
  ]
  feeds: [
    { to: "db.In", tag: "/table/table", data: { attr: "id attr" } }
    { to: "db.In", tag: "/column/table/id", data: { name: "Ident" } }
    { to: "db.In", tag: "/column/table/attr", data: { name: "Attributes" } }

    { to: "db.In", tag: "/table/column", data: { attr: "id name" } }
    { to: "db.In", tag: "/column/column/id", data: { name: "Ident" } }
    { to: "db.In", tag: "/column/column/name", data: { name: "Name" } }

    { to: "db.In", tag: "/table/driver", data: { attr: "id name unit factor scale" } }
    { to: "db.In", tag: "/column/driver/id", data: { name: "Parameter" } }
    { to: "db.In", tag: "/column/driver/name", data: { name: "Name" } }
    { to: "db.In", tag: "/column/driver/unit", data: { name: "Unit" } }
    { to: "db.In", tag: "/column/driver/factor", data: { name: "Factor" } }
    { to: "db.In", tag: "/column/driver/scale", data: { name: "Scale" } }

    { to: "db.In", tag: "/table/reading", data: { attr: "id loc val ms typ" } }
    { to: "db.In", tag: "/column/reading/id", data: { name: "Ident" } }
    { to: "db.In", tag: "/column/reading/loc", data: { name: "Location" } }
    { to: "db.In", tag: "/column/reading/val", data: { name: "Values" } }
    { to: "db.In", tag: "/column/reading/ms", data: { name: "Timestamp" } }
    { to: "db.In", tag: "/column/reading/typ", data: { name: "Type" } }
  ]

# write configuration to file, but keep a backup of the original, just in case
fs = require 'fs'
try fs.renameSync 'setup.json', 'setup-prev.json'
fs.writeFileSync 'setup.json', JSON.stringify circuits, null, 4
