#!/usr/bin/env coffee
# Generate hard-to-edit JSON config file from an easy-to-edit CoffeeScript one.
jeeboot = {}

jeeboot.swids =
  1005: '/home/arduino/widuino/nodes/servo_test/servo_test.hex'
  1010: '/home/arduino/widuino/nodes/temp_node/temp_node.hex'
	
jeeboot.hwids =
  'b4a667a2545f4d458492d517c7185035':
    board: 1, group: 212, node: 14, swid: 1010

# write configuration to file, but keep a backup of the original, just in case
fs = require('fs')
try fs.renameSync 'jeeboot.json', 'jeeboot-prev.json'
fs.writeFileSync 'jeeboot.json', JSON.stringify jeeboot, null, 4
