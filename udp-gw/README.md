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

The udpgw uses "Raw RF Messages" as documented in ../README.md

UDP Packet Format
-----------------

The UDP packet format is a very simple binary representation of the RFM12B packet format.
Because UDP already provides a length as well as a CRC the corresponding RFM12B fields
are omitted. The message type bits in the RFM12B node_id byte are separated off into a
message type code byte. The result is as follows:

    Byte Content
      0  message type code
      1  group_id
      2  node_id
    3..N data

Where:
- The type codes are as defined in the table above.
- The group_id is redundant on transmission because the rf12-udp-gw is locked to one group, but
  it's useful on reception to detect what group the JeeNode is on.
- The packet length is given in the UDP header
- A CRC is calculated and checked as part of UDP transmission/reception


