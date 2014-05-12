// Copyright (c) 2013-2014 Thorsten von Eicken
//
// RFG12 network transport layer with retransmissions and dispatch of received messages.

#ifndef Net_h
#define Net_h

// Special node IDs
#define NET_GW_NODE      1  // gateway to the IP network

// Operating modes
#define NET_MODE_NORMAL   1   // normal mode: full power, 115Kbps
#define NET_MODE_LOW      2   // low power mode for testing: low tx power, low sensitivity, 115Kbps
#define NET_MODE_SLOW     3   // slow mode for extended range: full power, 19Kbps

// rf12 packet minus the leading group byte
typedef struct {
  uint8_t   hdr;
  uint8_t   len;
  uint8_t   data[RF12_MAXDATA];
} net_packet;
#ifndef NET_PKT
#define NET_PKT        3                // number of packet buffers allocated
#endif

// Global variables that are managed by the network module
extern uint8_t node_id;         // this node's rf12 ID

class Net : public Configured {
  // variables related to sending packets
  net_packet buf[NET_PKT];      // buffer for outgoing packets
  uint8_t bufCnt;               // number of outgoing packets in buffer
  uint8_t sendCnt;              // number of transmissions of last packet
  uint32_t sendTime;            // when last packet was sent (for retries)
  uint8_t queuedAck;            // header of queued ACK
  uint8_t queuedRssi;           // RSSI being sent back with ACK
  uint8_t group_id;             // network group id

  void doSend(void);
  void getRssi(void);
  void queueAck(byte nodeId);
  void announce(void);
  void handleInit(void);
  void initRadio(uint8_t);

public:
  uint8_t lastAckRssi;          // RSSI received in the last ACK
  uint8_t lastRcvRssi;          // RSSI of the last received packet

  // Constructor, doesn't init any HW yet; the HW is configured by applyConfig() which is
  // called by the EEPROM config system after the EEPROM is read
  // @node_id is the rf12 node_id to use if not overridden by JeeBoot
  // @group_id is the rf12 group_id to use if not overridden by JeeBoot
  Net(uint8_t node_id, uint8_t group_id=0xD4);

  // alloc allocates a packet buffer, allowing multiple packets to be queued. This comes
  // in handy particularly when data packets and debug message packets are sent in rapid
  // succession.
  // @return pointer to buffer (corresponding to data payload) or null if no buffer is available
  uint8_t *alloc(void);

  // send the last allocated buffer as a packet to the management server. Note that
  // this means that each alloc call must be followed by a send or bcast call.
  // @len is the length of the payload
  // @ack says whether an ACK should be requested
  void send(uint8_t len, bool ack=true);

  // raw form of send where full header gets passed-in
  void rawSend(uint8_t len, uint8_t hdr);

  // bcast broadcasts the last allocated buffer as a packet to all nodes.
  // @len is the length of the payload
  void bcast(uint8_t len);

  // poll must be called in the arduino loop() function to keep the network moving (both
  // send and receive)
  // @return the first byte (module_id) of a received packet, 0 when there's no packet
  uint8_t poll(void);

  // Flush any queued packets and wait for ACKs until they come in or time out.
  // @return 0 when there are no more packets queued and nothing has been received.
  // Returns the module ID (first byte of packet) if a packet came in.
  uint8_t Net::flush(void);

  void reXmit(void);

  // Configuration methods
  virtual void applyConfig(uint8_t *);
  virtual void receive(volatile uint8_t *pkt, uint8_t len);
};

extern Net net; // must be allocated in sketch's main

#endif // Net_h
