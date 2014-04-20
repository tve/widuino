UDP-GW - Connect JeeNodes to JeeBus via UDP
===========================================

The UDP Gateway consists of two parts:
 - a sketch for a rf12-udp-gw JeeNode with an EtherCard that acts as the GW between the
   JeeNode RF network and the Ethernet
 - a udp-gw Go program that connects the UDP packets into JeeBus' MQTT message exchange

The actual function of the udp-gw is very simple. The rf12-udp-gw puts itself into promiscuous
mode so it receives all messages on the RF network and forwards all "interesting" ones over UDP,
and vice versa, it receives packets over UDP and forwards them on RF.

The udpgw Go program has two functions: it converts the binary message encoding over UDP into
JSON and it hooks-up to MQTT. The reason for having the Go program as opposed to having the
rf12-udp-gw talk MQTT directly is three-fold: JSON encoding/decoding in a JeeNode is no fun, the
MQTT protocol is no fun either, and the TCP stack of the EtherCard is unreliable for long
lived connections.


MQTT Messages
-------------

The udp-gw produces standardized JSON MQTT messages as follows:

### UDP -> MQTT

Received data messages:
* topic : `rf/<rf_group>/<src_node_id>/rx`
* JSON value : `{_asof:<timestamp>, base64:<base64_payload>}`
* QoS : as specified in wants-ACK RF flag

Received boot messages:
* topic : `io/udp-<local_port>/<remote_ip>-<remote_port>/<src_node_id>/rb` ("r*b*" as in "boot")
* JSON value : `{_asof:<timestamp>, kind:<pkt_type>, base64:<base64_payload>}`
* QoS : 0

Where:
* the `rx` topic is for application messages and `rb` is for boot protocol messages
* `remote_*` refers to the rf12-udp-gw
* `pkt_type` is either `pairing` or `boot`

## MQTT -> UDP

Sending data messages:
* topic : `io/udp-<local_port>/<remote_ip>-<remote_port>/+/tx` where the `+` component is the destination node id or `null` to broadcast
* JSON value : `{base64:<base64_payload>}`
* QoS : turned into wants-ACK RF flag

Sending boot messages:
* topic : `io/udp-<local_port>/<remote_ip>-<remote_port>/+/tb`
* JSON value : `{kind:<pkt_type>, base64:<base64_payload>}`
* QoS : must be 0

Where:
* the `tx` topic is for application messages and `tb` is for boot protocol messages
* `qos` mirrors the MQTT QoS and maps 0->no ACK, 1->ACK w/rexmit
* `pkt_type` is either `pairing` or `boot`

UDP Messages
------------

    Byte Content
      0  message type code
      1  group_id
      2  node_id
    3..N data

Where:
- The type codes are as defined in the table above.
- The group_id is redundant on transmission because the rf12-udp-gw is locked to one group, but
  it's useful on reception to detect what group the JeeNode is on.


