// Copyright (c) 2013-2014 Thorsten von Eicken
//
// TODO: combine ACKs with responses for packet types that send an immediate response
// TODO: get config CRC from EEPROM

// Reminder: rf12 header (per JeeLib rf12.cpp)
// DST=0: broadcast from named node
// DST=1: unicast to named node
// CTL=0, ACK=0: normal packet, no ack requested
// CTL=0, ACK=1: normal packet, ack requested
// CTL=1, ACK=0: ack packet
// CTL=1, ACK=1: unused
// node 0: reserved for OOK
// node 31: reserved for receive-all-packets

// compile-time definitions to select network implementation
// valid values: NET_NONE, NET_RF12B, NET_SERIAL
#if !defined(NET_NONE) && !defined(NET_SERIAL)
#define NET_RF12B
#endif

#include <JeeLib.h>
#include <JeeBoot.h>
#include <alloca.h>
#include <Config.h>
#include <Net.h>

#ifdef NET_SERIAL
#include <Base64.h>
#include <util/crc16.h>
#endif

// Serial Proto States
#define SPS_IDLE   0
#define SPS_START  1 // got start character, reading length
#define SPS_DATA   3 // got length, reading data

// Packet buffers and retries
#define NET_RETRY_MS 100
#define NET_RETRY_MAX  8

// Method for getting RSSI of received packets, this works well by connecting the appropriate
// capacitor on the RF12B module to the SCL/ADC5/PC5 pin on the JeeNode and placing a 1nF capacitor
// across that to ground.
#define RSSI_PIN 5       // read using analogRead

uint8_t node_id;         // this node's rf12 ID

#define DEBUG 1

// EEPROM configuration data
typedef struct {
  uint8_t   radio_mode;   // remember the NET_MODE_XX setting
} net_config;

// Allocate a packet buffer and return a pointer to it
uint8_t *Net::alloc(void) {
#ifndef NET_NONE
  if (bufCnt < NET_PKT) return buf[bufCnt].data;
#endif
  return 0;
}

// send the packet at the top of the queue
void Net::doSend(void) {
#ifdef NET_NONE
  return;
#else
  if (bufCnt > 0) {
    uint8_t hdr = buf[0].hdr;
    // Don't ask for an ACK on the last retry
    if (sendCnt+1 >= NET_RETRY_MAX)
      hdr &= ~RF12_HDR_ACK;
    // send as broadcast packet without ACK
#ifdef NET_RF12B
    rf12_sendStart(hdr, &buf[0].data, buf[0].len);
#else
#ifdef NET_SERIAL
#endif
#endif

#if DEBUG
    Serial.print(F("Net::doSend: "));
    Serial.print(hdr & RF12_HDR_ACK ? "  w/ACK " : " no-ACK ");
    Serial.print("#");
    Serial.print(sendCnt);
    Serial.print(" 0x");
    Serial.print(buf[0].data[0], 16);
    Serial.print(":"); Serial.println(buf[0].len);
#endif
    // pop packet from queue if we don't expect an ACK
    if ((hdr & RF12_HDR_ACK) == 0) {
      if (bufCnt > 1)
        memcpy(buf, &buf[1], sizeof(net_packet)*(NET_PKT-1));
      bufCnt--;
      sendCnt=0;
    } else {
      sendCnt++;
      sendTime = millis();
    }
  }
#endif
}

// Send a new packet, this is what user code should call
void Net::send(uint8_t len, bool ack) {
#ifndef NET_NONE
  //uint8_t hdr = RF12_HDR_DST | (ack ? RF12_HDR_ACK : 0) | NET_GW_NODE;
  uint8_t hdr = (ack ? RF12_HDR_ACK : 0) | node_id;
  rawSend(len, hdr);
#endif
}

// raw form of send where full header gets passed-in
void Net::rawSend(uint8_t len, uint8_t hdr) {
#ifndef NET_NONE
  if (bufCnt >= NET_PKT) return; // error?
  buf[bufCnt].len = len;
  buf[bufCnt].hdr = hdr;
  bufCnt++;
  // if there was no packet queued just go ahead and send the new one
  if (bufCnt == 1) {
    if (rf12_canSend()) {
      doSend();
    } else {
      //Serial.println(F("Cannot send"));
    }
  }
#endif
}

// Broadcast a new packet, this is what user code should call
void Net::bcast(uint8_t len) {
#ifndef NET_NONE
  if (bufCnt >= NET_PKT) return; // error?
  buf[bufCnt].len = len;
  buf[bufCnt].hdr = node_id;
  bufCnt++;
  // if there was no packet queued just go ahead and send the new one
  if (bufCnt == 1 && rf12_canSend()) {
    doSend();
  }
#endif
}

void Net::getRssi(void) {
#ifndef NET_NONE
  // go and get the analog RSSI -- TODO: this needs to be configurable
# ifdef RSSI_PIN
    lastRcvRssi = (uint8_t)((analogRead(RSSI_PIN)-300) >> 2);
# else
    lastRcvRssi = 0;
# endif
# endif
}

// Queue an ACK packet
void Net::queueAck(byte dest_node) {
  // ACK packets have CTL=1, ACK=0; and
  // either DST=1 and the dest of the ack, or DST=0 and this node as source
  int8_t hdr = RF12_HDR_CTL;
  hdr |= dest_node == node_id ? node_id : (RF12_HDR_DST|dest_node);
  queuedAck = hdr; // queue the ACK
  queuedRssi = lastRcvRssi;
}

// Poll the rf12 network and return true if a packet has been received
// ACKs are processed automatically (and are expected not to have data,
// but do include the RSSI to make for simple round-trip measurements)
uint8_t Net::poll(void) {
#ifndef NET_NONE
  bool rcv = rf12_recvDone();
  if (rcv && rf12_crc == 0) {
    //Serial.print("Got packet with HDR=0x");
    //Serial.println(rf12_hdr, HEX);
    // at this point either it's a broadcast or it's directed at this node
    if (!(rf12_hdr & RF12_HDR_CTL)) {
      // Normal packet (CTL=0), queue an ACK if that's requested
      // (can't immediately send 'cause we need the buffer)
      getRssi();
      if (rf12_hdr & RF12_HDR_ACK) queueAck(rf12_hdr & RF12_HDR_MASK);
      return rf12_data[0];
    } else if (!(rf12_hdr & RF12_HDR_ACK)) {
      // Ack packet, check that it's for us and that we're waiting for an ACK
      //Serial.print("Got ACK for "); Serial.println(rf12_hdr, 16);
      getRssi();
      if ((rf12_hdr&RF12_HDR_MASK) == node_id && bufCnt > 0 && sendCnt > 0) {
        lastAckRssi = rf12_len == 1 ? rf12_data[0] : 0;
        // pop packet from queue
        if (bufCnt > 1) {
          memcpy(buf, &buf[1], sizeof(net_packet)*(NET_PKT-1));
        }
        bufCnt--;
        sendCnt=0;
      }
    }
  } else if (rcv && rf12_crc != 0) {
    //Serial.println("Got packet with bad CRC");
  }

  reXmit();
#endif
  return 0;
}

void Net::reXmit(void) {
#ifndef NET_NONE
  // If we have a queued ack, try to send it
  if (queuedAck && rf12_canSend()) {
    rf12_sendStart(queuedAck, &queuedRssi, sizeof(queuedRssi));
    queuedAck = 0;
    queuedRssi = 0;

  // We have a freshly queued message (never sent), try to send it
  } else if (bufCnt > 0 && sendCnt == 0) {
    if (rf12_canSend()) doSend();

  // We have a queued message  that hasn't been acked and it's time to retry
  } else if (bufCnt > 0 && sendCnt > 0 && millis() >= sendTime+NET_RETRY_MS) {
    if (rf12_canSend()) {
      //Serial.print("Rexmit to 0x");
      //Serial.print(buf[0].hdr, 16);
      //Serial.print(" 0x");
      //Serial.print(buf[0].data[0], 16);
      //Serial.println((char *)(buf[0].data+1));
      doSend();
    }

  }
#endif
}

// Constructor
Net::Net(uint8_t c_node_id, uint8_t group_id) {
#ifdef NET_NOJEEBOOT
  this->group_id = group_id;
  node_id = c_node_id;
#else
  if (jb_group_id != 0 && jb_node_id > 0 && jb_node_id < 31) {
    this->group_id = jb_group_id;
    node_id = jb_node_id;
  } else {
    this->group_id = group_id;
    node_id = c_node_id;
  }
#endif

  moduleId = NET_MODULE;
  configSize = sizeof(net_config);
}

// ===== Configuration =====

void Net::receive(volatile uint8_t *pkt, uint8_t len) {
#ifndef NET_NONE
  // TODO: accept packets to change mode
#endif
}

// ApplyConfig() not just processes the EEPROM config but also initializes the RF12 module
void Net::applyConfig(uint8_t *cf) {
  net_config *eeprom = (net_config *)cf;

  // do we have data from EEPROM or not?
  if (eeprom) {
    Serial.print(F("Config: ")); Serial.println(eeprom->radio_mode);
  } else {
    eeprom = (net_config *)alloca(sizeof(net_config));
    // we need to punch-in some default values
    eeprom->radio_mode = NET_MODE_NORMAL;
    config_write(NET_MODULE, eeprom);
  }
  initRadio(eeprom->radio_mode);
}

void Net::initRadio(uint8_t radio_mode) {
#ifdef NET_NONE
  Serial.println(F("Config Net: RF12B disabled"));
  return;
#else
  // initialize rf12 module
  Serial.print(F("Config Net: node_id="));
  Serial.print(node_id);
  Serial.print(F(" group_id="));
  Serial.print(group_id);
  Serial.print(F(" mode="));
  Serial.print(radio_mode);
  rf12_initialize(node_id, RF12_915MHZ, group_id);
  switch (radio_mode) {
  case NET_MODE_LOW:
    Serial.println(F("  low TX power"));
    rf12_control(0x9857); // reduce tx power
    rf12_control(0x94B2); // attenuate receiver 0x94B2 or 0x94Ba
    break;
  case NET_MODE_SLOW:
    Serial.println(F("  19kbps"));
    rf12_control(0xC611); // 19.1kbps
    rf12_control(0x94C1); // VDI:fast,-97dBm,67khz
    rf12_control(0x9820); // 45khz
  default:
    Serial.println();
  }

  //Serial.println("  rf12 initialized");
  // if we're collecting RSSIs then init the analog pin
#ifdef RSSI_PIN
  analogReference(INTERNAL);
  pinMode(RSSI_PIN, INPUT);
#endif
#endif
}
