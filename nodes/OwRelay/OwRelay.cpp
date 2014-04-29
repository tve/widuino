#include <JeeLib.h>
#include <OwRelay.h>
#include <OneWire.h> // needed by makefile, ugh
#include <Config.h>
#include <Log.h>

#define DEBUG 0

// ===== Constructors =====

OwRelay::OwRelay(byte pin, uint8_t count) : ds(pin) {
  init(pin, count);
  rlyAddr = (uint64_t *)calloc(rlyCount, sizeof(uint64_t));
  staticAddr = false;
}

OwRelay::OwRelay(byte pin, uint8_t count, uint64_t *addr) : ds(pin) {
  init(pin, count);
  rlyAddr = addr;
  staticAddr = true;
}

void OwRelay::init(byte pin, uint8_t count) {
  rlyCount = count < 16 ? count : 16;
  convState = 0;
  failed = 0;
  rlyState = (bool *)calloc(rlyCount, sizeof(bool));
  moduleId = OWRELAY_MODULE;
  configSize = sizeof(uint64_t)*rlyCount;
#if DEBUG
	Serial.print("OWR: count=");
	Serial.print(rlyCount);
	Serial.print(" configSize=");
	Serial.print(configSize);
	Serial.println();
#endif
}

// ===== Operation =====

uint8_t OwRelay::setup(Print *printer) {
  // run a search on the bus to see what we actually find
  uint64_t addr;                   // next detected switch
  uint16_t found = 0;              // which addrs we actually found
  uint16_t added = 0;              // which addrs are new
  byte n_found = 0;                // number of switches actually discovered

#if DEBUG
  printer->print(F("OWR <EEPROM:"));
  for (byte s=0; s<rlyCount; s++) {
    printer->print(" ");
    printAddr(printer, rlyAddr[s]);
  }
  printer->println();
#endif

	ds.reset_search();
  while (ds.search((uint8_t *)&addr)) {
    // make sure the CRC is valid
		byte crc = OneWire::crc8((uint8_t *)&addr, 7);
    if (crc != (addr>>56)) continue;

    // make sure it's a DS2405
    if ((addr&0xff) != 0x05) continue;

    n_found++;

    // see whether we know this switch already
    for (byte s=0; s<rlyCount; s++) {
      if (addr == rlyAddr[s]) {
        printer->print(F("OWR: found #"));
        printAddr(printer, addr);
        printer->println();
        found |= (uint16_t)1 << s;  // mark switch as found
        goto cont;
      }
    }

    // new switch, if we have space add it
    for (byte s=0; s<rlyCount; s++) {
      if (rlyAddr[s] == 0) {
        rlyAddr[s] = addr;
#       if DEBUG
        printer->print("OWR: new #");
        printAddr(printer, addr);
        printer->println();
#       endif
        added |= (uint16_t)1 << s;  // mark switch as added
        break;
      }
    }

  cont: ;
  }
	ds.reset_search();

  // print info about additional switches found
  if (added) {
    printer->print(F("OWR: New switches:    "));
    for (byte s=0; s<rlyCount; s++) {
      if (added & ((uint16_t)1 << s)) {
        printer->print(" ");
        printAddr(printer, rlyAddr[s]);
      }
    }
    printer->println();
  }

  // print info about missing switches
  uint16_t missing = (((uint16_t)1 << rlyCount)-1) & ~(found | added);
  if (missing) {
    printer->print(F("OWR: Missing switches:"));
    for (byte s=0; s<rlyCount; s++) {
      if (missing & ((uint16_t)1 << s)) {
        printer->print(" ");
        printAddr(printer, rlyAddr[s]);
      }
    }
    printer->println();
  }

  printer->print(F("OWR: ready with "));
  printer->print(rlyCount);
  printer->println(F(" switches"));

  config_write(OWRELAY_MODULE, rlyAddr);

  // start a poll
  loop(0);

  return n_found;
}

// Poll relay switches every <secs> seconds; use secs=0 to force conversion now
bool OwRelay::loop(uint8_t secs) {
  unsigned long now = millis();
  if (secs == 0 || now - lastPoll > (unsigned long)secs * 1000) {
    // poll now
    lastPoll = now;

    for (byte s=0; s<rlyCount; s++) {
      if (rlyAddr[s] == 0) continue;
      uint8_t state = read(rlyAddr[s]);
      uint16_t bit = (uint16_t)1 << s;

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
	return false;
}

// ===== Accessors =====

bool OwRelay::get(uint8_t i) {
  return i < rlyCount && rlyState[i];
}

bool OwRelay::set(uint8_t i, bool value) {
  if (rlyState[i] == value) return true;
  bool ok = write(rlyAddr[i], value);
  if (ok) rlyState[i] = value;
  return ok;
}

bool OwRelay::getByAddr(uint64_t addr) {
  for (byte s=0; s<rlyCount; s++) {
    if (rlyAddr[s] == addr) return rlyState[s];
  }
  return false;
}

uint64_t OwRelay::getAddr(uint8_t i) {
  return i < rlyCount ? rlyAddr[i] : 0;
}

void OwRelay::swap(uint8_t i, uint8_t j) {
  if (i >= rlyCount || j >= rlyCount) return;
  // swap states
  bool t = rlyState[i];
  rlyState[i] = rlyState[j];
  rlyState[j] = t;
  // swap addresses
  uint64_t a = rlyAddr[i];
  rlyAddr[i] = rlyAddr[j];
  rlyAddr[j] = a;
}

void OwRelay::printAddrRev(Print *printer, uint64_t addr) {
  uint8_t *a = (uint8_t *)&addr;
  printer->print("0x");
  for (byte b=7; b>=0; b--) {
    printer->print(a[b] >> 4, HEX);
    printer->print(a[b] & 0xF, HEX);
  }
}

void OwRelay::printAddr(Print *printer, uint64_t addr) {
  uint8_t *a = (uint8_t *)&addr;
  printer->print("0x");
  // print only the top 3 bytes, the rest is always 31000000 (at least for me)
  for (byte b=0; b<3; b++) {
    printer->print(*a >> 4, HEX);
    printer->print(*a & 0xF, HEX);
    a++;
  }
}

void OwRelay::printDebug(Print *printer) {
  printer->print(F("OwRelay has "));
  printer->print(rlyCount);
  printer->println(F(" switches"));

  for (uint8_t s=0; s<rlyCount; s++) {
    printer->print("  #");
    printer->print(s);
    printer->print(": ");
    printAddr(printer, rlyAddr[s]);
    if (rlyAddr[s] != 0) { 
      printer->print(" now:");
      printer->print(rlyState[s]);
    }
    printer->println();
  }
}

// ===== Configuration =====

void OwRelay::applyConfig(uint8_t *cf) {
  if (cf) {
    if (rlyAddr[0] == 0) {
      // address array is empty -> restore from EEPROM
      memcpy(rlyAddr, cf, sizeof(uint64_t)*rlyCount);
#     if DEBUG
      Serial.println(F("Config OwRelay: restored addrs from EEPROM"));
#     endif
    }
  } else {
    // no valid config in EEPROM, leave switch address array as-is
  }
}

void OwRelay::receive(volatile uint8_t *pkt, uint8_t len) {
  // sorry, we ain't processing no packets...
}

// ===== One Wire utilities =====

// Read the state, returns -1 on failure
int8_t OwRelay::read(uint64_t addr) {
  byte data[12];
  ds.reset();

  ds.write(0xF0); // search
  for(int i=0; i<64; i++) {
    uint8_t myBit = (addr>>i) & 1;        // bit of the device we want to address
    uint8_t bit  = ds.read_bit();         // devices write bit
    uint8_t bitI = ds.read_bit();         // devices write inverse bit
    if (bit && bitI) {
      ds.reset();
#if DEBUG
      Serial.print(F("OWR: Read failed for "));
      printAddr(&Serial, addr);
      Serial.println();
#endif
      return -1;
    }
    ds.write_bit(myBit);
  }
  uint8_t value = ds.read_bit() ^ 1;
  ds.reset();

#if DEBUG
  Serial.print(F("OWR: Good data for "));
  printAddr(&Serial, addr);
  Serial.print("->");
  Serial.print(value);
  Serial.println();
#endif
  
  return value;
}

bool OwRelay::write(uint64_t addr, bool value) {
  value ^= 1;
  uint8_t *a = (uint8_t *)&addr;
  ds.reset();
  ds.write(0x55, 1); // ROM select
  for(int i = 0; i < 8; i++) ds.write(a[i], 1);
  uint8_t bit = ds.read_bit();
  ds.reset();

  bool oops = false;
  if ((bit&1) ^ value) {
    oops = true;
    // toggled the wrong way, toggle again...
    ds.write(0x55, 1); // ROM select
    for(int i = 0; i < 8; i++) ds.write(a[i], 1);
    bit = ds.read_bit();
    ds.reset();
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

