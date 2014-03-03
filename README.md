WIDUINO - Wireless Arduinos and JeeNodes
========================================

Miscellaneous software to build a wireless network of arduinos and JeeNodes that use
the HopeRF RFM12B and RFM69 wireless networks. It is intended to be compatible with
JeeBus and JeeBoot but at the time of writing those two projects were undergoing too
much change to provide a stable platform.

The widuino software is based around a MQTT message broker and RFM12B networks using the
JeeLib driver. The server side software is written in Go.

Components
----------

- udp-gw: gateway between RF networks and MQTT consisting of a sketch for a JeeNode+EtherCard
that bridges the RF network and UDP, and a Go application that bridges UDP and the MQTT network.
- utils: various utilities
- wiboot: a simple boot utility that is compatible with the JeeBoot bootloader and allows
sketches to be loaded into JeeBoot-enabled nodes remotely

RF Packets
----------

The RF packet format used by JeeLib for RFM12B nodes is documented is various hard to find
places. This table summarizes the packet types including the new JeeBoot types, notes how
Widuino uses them, and assigns a "type code".

    Type      Code Purpose                                 CTL DST ACK NODE  Widuino use
    bcast_push  0  Broadcast data packet, no ACK requested  0   0   0   src  node->core, QoS=0
    bcast_req   1  Broadcast data packet, ACK requested     0   0   1   src  node->core, QoS=1
    data_push   2  Normal data packet, no ACK requested     0   1   0  dest  core->node, QoS=0
    data_req    3  Normal data packet, ACK requested        0   1   1  dest  core->node, QoS=1
    ack_data    4  ACK reply packet for a data packet       1   0   0   src  node->core ack
    ack_bcast   6  ACK reply packet for a broadcast packet  1   1   0  dest  core->node ack
    boot_req    5  Boot protocol request                    1   0   1   src  node->core boot
    boot_reply  7  Boot protocol reply                      1   1   1  dest  core->node boot
    pairing     8  Pairing request (reply is a boot_reply)  1   1   1     1  node->core boot
    pairing     8  Pairing request (reply is a boot_reply)  1   1   1     0  node->core boot
    debug       9  Debug log messages from JeeUDP sketch                     gw->core logging

Widuino basically assumes that the GW node (relaying messages between the core software and RF
nodes) listens to all RF traffic (promiscuious mode) and forwards all messages shown as
"node->core" to the core. These messages are all broadcast type so they include a source node
ID, which allows the core to figure out the message type and encoding based on the sketch being
run by the node. Messages sent from the core to nodes use unicast (exceptions are possible).

MQTT Raw RF Messages
--------------------

The purpose of the raw RF messages is to allow gateways to inject the information in the
packets in pretty raw format so the next layer up can decode/encode those messages.

The Raw RF messages represent the packets to/from RF nodes pretty directly, except that
the various header fields are abstracted such that the MQTT messages are independent of
the packet encoding and field details.

### RF -> MQTT

Received data messages:
* topic : `rf/<rf_group>/<src_node_id>/rx`
* JSON value : `{_asof:<timestamp>, base64:<base64_payload>}`
* QoS : as specified in wants-ACK RF flag

Received boot messages:
* topic : `rf/<rf_group>/<src_node_id>/rb` ("r*b*" as in "boot")
* JSON value : `{_asof:<timestamp>, kind:<pkt_type>, base64:<base64_payload>}`
* QoS : 0

Where:
* the `rx` topic is for application messages and `rb` is for boot protocol messages
* the `source_node_id` is 0 if it's unknown (due to pkt format limitations)
* `pkt_type` is either `pairing` or `boot`

## MQTT -> RF

Sending data messages:
* topic : `rf/<rf_group>/<dest_node_id>/tx`
* JSON value : `{base64:<base64_payload>}`
* QoS : turned into wants-ACK RF flag

Sending boot messages:
* topic : `rf/<rf_group>/<dest_node_id>/tb`
* JSON value : `{kind:<pkt_type>, base64:<base64_payload>}`
* QoS : must be 0

Where:
* the `tx` topic is for application messages and `tb` is for boot protocol messages
* for data messages a `dest_node_id` of 0 causes a broadcast (for boot messages the pairing request and corresponding boot reply use node_id 0)
* `qos` mirrors the MQTT QoS and maps 0->no ACK, 1->ACK w/rexmit
* `pkt_type` is either `pairing` or `boot`
