#include <JeeLib.h>
#include <OwRelay.h>
#include <OwScan.h>
#include <OneWire.h>
#include <Logger.h>

#define MAX_COUNT        32 // max number of devices, limited due to bitfield ops

#define DEBUG 0

// ===== Constructors =====

OwRelay::OwRelay(uint8_t pin, uint64_t *addr, uint8_t max) {
  init(pin, addr, max);
}

void OwRelay::init(uint8_t pin, uint64_t *addr, uint8_t max) {
  ds = new OneWire(pin);
  rlyAddr = addr;
  rlyMax = max < MAX_COUNT ? max : MAX_COUNT;
  rlyState = 0;
}

// ===== Operation =====

// Poll relay switches every <secs> seconds; use secs=0 to force conversion now
bool OwRelay::loop(uint8_t secs) {
  unsigned long now = millis();
  if (secs != 0 && lastPoll != 0 && now - lastPoll <= (unsigned long)secs * 1000) {
    // not time to poll
    return false;
  }
  lastPoll = now;

  // Check all relays
  for (uint8_t s=0; s<rlyMax; s++) {
    // see whether it's a relay
    if (!isRelay(s)) continue;

    bool state = rawRead(s);
    uint32_t bit = (uint32_t)1 << s;
    rlyState = (rlyState & ~bit) | ((uint32_t)state << s);

#if DEBUG
    logger->print("OWR ");
    logger->print(s);
    logger->print("=");
    logger->println(state&1);
#endif
  }
  return true;
}

// ===== Accessors =====

bool OwRelay::get(uint8_t i) {
  return i < rlyMax && ((rlyState>>i)&1);
}

bool OwRelay::set(uint8_t i, bool value) {
  if (i >= rlyMax || !isRelay(i)) return false;
  bool ok = rawWrite(i, value);
  uint32_t bit = (uint32_t)1 << i;
  if (ok) rlyState = (rlyState & ~bit) | ((uint32_t)value << i);
  return ok;
}

bool OwRelay::isRelay(uint8_t s) {
  uint8_t dev = get_ow_type(rlyAddr+s);
  return dev == 0x05;
}

void OwRelay::printDebug(Print *printer) {
  printer->print(F("OwRelay has "));
  printer->print(rlyMax);
  printer->println(F(" switches"));

  for (uint8_t s=0; s<rlyMax; s++) {
    if (!isRelay(s)) continue;
    printer->print(s);
    printer->print('=');
    printer->print(get(s));
    printer->print(' ');
  }
  printer->println();
}

// Read the state, returns -1 on failure
bool OwRelay::rawRead(uint8_t r) {
  uint64_t rev_addr = reverse_ow_addr(rlyAddr+r);
  byte data[12];

  // run a search on the bus for this device 'cause it's the only way to read it without
  // toggling it, sigh
  ds->reset();
  //ds->select((uint8_t *)&rev_addr); // the select call does a ROM select, which toggles :-(
  ds->write(0xF0, 1); // search
  for (uint8_t i=0; i<64; i++) {
    uint8_t bit = (rev_addr>>i) & 1;
    uint8_t id_bit_pos = ds->read_bit();
    uint8_t id_bit_neg = ds->read_bit();
    if (bit && id_bit_neg || !bit && id_bit_pos) {
      logger->print("OWR read failed: bit ");
      logger->print(i);
      logger->print(" need ");
      logger->print(bit);
      logger->print(" got ");
      logger->print(id_bit_pos);
      logger->print('/');
      logger->println(id_bit_neg);
      return 0;
    }
    ds->write_bit(bit);
  }
  bool state =  ds->read_bit() & 1; // read the switch value
  ds->reset();
  return state;
}

bool OwRelay::rawWrite(uint8_t r, bool value) {
  bool cur = rawRead(r);
  if (cur == value) return true; // it's already looking good

  // gotta toggle the value
  uint64_t rev_addr = reverse_ow_addr(rlyAddr+r);
  uint8_t *a = (uint8_t *)&rev_addr;
#if DEBUG
  logger->print("OWR toggle ");
  logger->println(r);
#endif

  ds->reset();
  ds->write(0x55, 1); // ROM select
  for(int i = 0; i < 8; i++) ds->write(a[i], 0);
  bool bit = ds->read_bit() & 1;
  ds->reset();

  return bit == value;
}

