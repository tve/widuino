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
  failed = 0;
  rlyState = (bool *)calloc(max, sizeof(bool));
#if DEBUG
  Serial.print("OWR: max=");
  Serial.print(rlyMax);
  Serial.println();
#endif
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

    bool state = rawRead(rlyAddr[s]);
    uint32_t bit = (uint32_t)1 << s;

    if (state < 0) {
      // read failed
      rlyState[s] = false;
      failed |= bit;
    } else {
      // read succeeded
      failed &= ~bit;
      rlyState[s] = state & 1;
    }
  }
  return true;
}

// ===== Accessors =====

bool OwRelay::get(uint8_t i) {
  return i < rlyMax && rlyState[i];
}

bool OwRelay::set(uint8_t i, bool value) {
  if (rlyState[i] == value) return true;
  bool ok = rawWrite(rlyAddr[i], value);
  if (ok) rlyState[i] = value;
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
    printer->print(s);
    printer->print(": ");
    printer->print(rlyState[s]);
    printer->println();
  }
}

// Read the state, returns -1 on failure
int8_t OwRelay::rawRead(uint64_t addr) {
  uint64_t rev_addr = reverse_ow_addr(&addr);
  byte data[12];
  ds->reset();
  ds->select((uint8_t *)&rev_addr);
  uint8_t value = ds->read_bit() ^ 1; // read the switch value

#if DEBUG
  Serial.print(F("OWR: Good data for "));
  printAddr(&Serial, addr);
  Serial.print("->");
  Serial.print(value);
  Serial.println();
#endif

  return value;
}

bool OwRelay::rawWrite(uint64_t addr, bool value) {
  uint64_t rev_addr = reverse_ow_addr(&addr);
  uint8_t *a = (uint8_t *)&rev_addr;
  ds->reset();
  ds->write(0x55, 1); // ROM select
  for(int i = 0; i < 8; i++) ds->write(a[i], 1);
  uint8_t bit = ds->read_bit();
  ds->reset();

  bool oops = false;
  value ^= 1;
  if ((bit&1) ^ value) {
    oops = true;
    // toggled the wrong way, toggle again...
    ds->write(0x55, 1); // ROM select
    for(int i = 0; i < 8; i++) ds->write(a[i], 1);
    bit = ds->read_bit();
    ds->reset();
  }
#if DEBUG
  Serial.print(F("OWR: Write "));
  printAddr(&Serial, addr);
  Serial.print("<-");
  Serial.print(value);
  if (oops) Serial.print(" 2xtoggle");
  Serial.print(" bit=");
  Serial.print(bit);
  Serial.println();
#endif

  return !((bit&1) ^ value);
}

