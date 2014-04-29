#include <JeeLib.h>
#include <OwTemp.h>
#include <OneWire.h> // needed by makefile, ugh
#include <Config.h>
#include <Log.h>

#define OWTEMP_CONVTIME 188 // milliseconds for a conversion (10 bits)
#define TEMP_OFFSET      88 // offset used to store min/max in 8 bits

#define MAX_COUNT        16 // max number of sensors, limited due to bitfield ops

#define DEBUG 0

// ===== Constructors =====

OwTemp::OwTemp(byte pin, uint8_t count) : ds(pin) {
  init(pin, count);
  sensAddr = (uint64_t *)calloc(sensCount, sizeof(uint64_t));
  staticAddr = false;
}

OwTemp::OwTemp(byte pin, uint8_t count, uint64_t *addr) : ds(pin) {
  init(pin, count);
  sensAddr = addr;
  staticAddr = true;
}

void OwTemp::init(byte pin, uint8_t count) {
  sensCount = count < MAX_COUNT ? count : MAX_COUNT;
  convState = 0;
  failed = 0;
  sensTemp = (float *)calloc(sensCount, sizeof(float));
  sensMin  = (int8_t (*)[6])calloc(sensCount, 6*sizeof(int8_t));
  memset(sensMin, 0x80, sensCount*6*sizeof(int8_t));
  sensMax  = (int8_t (*)[6])calloc(sensCount, 6*sizeof(int8_t));
  memset(sensMax, 0x80, sensCount*6*sizeof(int8_t));
  moduleId = OWTEMP_MODULE;
  configSize = sizeof(uint32_t)*sensCount;
#if DEBUG
	Serial.print("OWT: count=");
	Serial.print(sensCount);
	Serial.print(" configSize=");
	Serial.print(configSize);
	Serial.println();
#endif
}

// ===== Operation =====

uint8_t OwTemp::setup(Print *printer) {
  // run a search on the bus to see what we actually find
  uint64_t addr;                   // next detected sensor
  uint16_t found = 0;              // which addrs we actually found
  uint16_t added = 0;              // which addrs are new
  byte n_found = 0;                // number of sensors actually discovered

#if DEBUG
  printer->print(F("OWT <EEPROM:"));
  for (byte s=0; s<sensCount; s++) {
    printer->print(" ");
    printAddr(printer, sensAddr[s]);
  }
  printer->println();
#endif

	ds.reset_search();
  while (ds.search((uint8_t *)&addr)) {
    // make sure the CRC is valid
		byte crc = OneWire::crc8((uint8_t *)&addr, 7);
    if (crc != (addr>>56)) continue;

    n_found++;

    // see whether we know this sensor already
    for (byte s=0; s<sensCount; s++) {
#if 0
      printer->print("OWT: ");
      printer->print((uint32_t)(addr&0xffffffff));
      printer->print("=?=");
      printer->print((uint32_t)(sensAddr[s]&0xffffffff));
      printer->println();
#endif
      // we only restore 32 bits from EEPROM, so only compare that much...
      if ((uint32_t)addr == (uint32_t)(sensAddr[s])) {
        sensAddr[s] = addr;
#       if DEBUG
        printer->print("OWT: found #");
        printAddr(printer, addr);
        printer->println();
#       endif
        found |= (uint16_t)1 << s;  // mark sensor as found
        goto cont;
      }
    }

    // new sensor, if we have space add it
    for (byte s=0; s<sensCount; s++) {
      if (sensAddr[s] == 0) {
        sensAddr[s] = addr;
#       if DEBUG
        printer->print("OWT: new #");
        printAddr(printer, addr);
        printer->println();
#       endif
        added |= (uint16_t)1 << s;  // mark sensor as added
        break;
      }
    }

  cont: ;
  }
	ds.reset_search();

  // make sure all temp sensors are set to 10 bits of resolution
  for (byte s=0; s<sensCount; s++)
    if ((sensAddr[s]&0xff) == 0x22 || (sensAddr[s]&0xff) == 0x28)
      setresolution(sensAddr[s], 10);

  // print info about additional sensors found
  if (added) {
    printer->print(F("OWT: New sensors:    "));
    for (byte s=0; s<sensCount; s++) {
      if (added & ((uint16_t)1 << s)) {
        printer->print(" ");
        printAddr(printer, sensAddr[s]);
      }
    }
    printer->println();
  }

  // print info about missing sensors
  uint16_t missing = (((uint16_t)1 << sensCount)-1) & ~(found | added);
  if (missing) {
    printer->print(F("OWT: Missing sensors:"));
    for (byte s=0; s<sensCount; s++) {
      if (missing & ((uint16_t)1 << s)) {
        printer->print(" ");
        printAddr(printer, sensAddr[s]);
      }
    }
    printer->println();
  }

  printer->print(F("OWT: ready with "));
  printer->print(sensCount);
  printer->println(F(" sensors"));

  // save the config in EEPROM
  uint32_t save[sensCount];
  for (byte s=0; s<sensCount; s++)
    save[s] = sensAddr[s]; // loose top 32 bits
  config_write(OWTEMP_MODULE, save);

  // start a conversion
  lastConv = millis();
  start();

  return n_found;
}

// Poll temperature sensors every <secs> seconds; use secs=0 to force conversion now
bool OwTemp::loop(uint8_t secs) {
  // rotate min/max temp every 4 hours
  if (minMaxTimer.poll(60000)) {
    minMaxCount++;
    if (minMaxCount >= 4*60) {
      minMaxCount = 0;
      // rotate min/max temps
      for (int s=0; s<sensCount; s++) {
        for (int i=5; i>0; i--) {
          sensMin[s][i] = sensMin[s][i-1];
          sensMax[s][i] = sensMax[s][i-1];
        }
      }
    }
  }

  unsigned long now = millis();
  switch (convState) {
    case 0: // no sensors
      break;
    case 1: // idle
      if (secs == 0 || now - lastConv > (unsigned long)secs * 1000) {
        // start a conversion
        lastConv = now;
        start();
      }
      break;
    case 2: // conversion in progress
      if (now - lastConv > OWTEMP_CONVTIME) {
        // time to read the results
        for (byte s=0; s<sensCount; s++) {
          if (sensAddr[s] == 0) continue;
          float t = read(sensAddr[s]);
          uint16_t bit = (uint16_t)1 << s;
          if (isnan(t)) {
            // conversion failed
            if (failed & bit) {
              // sensor has been failing
              sensTemp[s] = NAN;
            } else {
              failed |= bit;
            }
          } else {
            // conversion succeeded
            failed &= ~bit;
            sensTemp[s] = t;
            // update min/max
            int8_t m = (int8_t)(round(t)-TEMP_OFFSET);
            //Serial.print("Min: "); Serial.print(m+TEMP_OFFSET); Serial.print(" ");
            //Serial.print(sensMin[s][0]+TEMP_OFFSET); Serial.print(" ");
            //Serial.println(m < sensMin[s][0]);
            if (sensMin[s][5] == -128)
              memset(sensMin[s], m, 6); // sensMin is uninitialized so set it all
            else if (m < sensMin[s][0])
              sensMin[s][0] = m;        // new minimum
            //Serial.print("Max: "); Serial.print(m+TEMP_OFFSET); Serial.print(" ");
            //Serial.print(sensMax[s][0]+TEMP_OFFSET); Serial.print(" ");
            //Serial.println(m > sensMax[s][0]);
            if (sensMax[s][5] == -128)
              memset(sensMax[s], m, 6); // sensMax is uninitialized so set it all
            else if (m > sensMax[s][0])
              sensMax[s][0] = m;        // new maximum
          }
        }
        convState = 1;
				return true;
      }
  }
	return false;
}

// ===== Accessors =====

float OwTemp::get(uint8_t i) {
  return i < sensCount ? sensTemp[i] : NAN;
}

float OwTemp::getByAddr(uint64_t addr) {
  for (byte s=0; s<sensCount; s++) {
    if (sensAddr[s] == addr) return sensTemp[s];
  }
  return NAN;
}

uint64_t OwTemp::getAddr(uint8_t i) {
  return i < sensCount ? sensAddr[i] : NAN;
}

uint16_t OwTemp::getMin(uint8_t i) {
  if (i >= sensCount) return 0x8000;
  int8_t t = sensMin[i][0];
  for (uint8_t h=1; h<6; h++)
    if (sensMin[i][h] < t)
      t = sensMin[i][h];
  return t + TEMP_OFFSET;
}

uint16_t OwTemp::getMax(uint8_t i) {
  if (i >= sensCount) return 0x8000;
  int8_t t = sensMax[i][0];
  for (uint8_t h=1; h<6; h++)
    if (sensMax[i][h] > t)
      t = sensMax[i][h];
  return t + TEMP_OFFSET;
}

void OwTemp::swap(uint8_t i, uint8_t j) {
  if (i >= sensCount || j >= sensCount) return;
  // swap temperatures
  float t = sensTemp[i];
  sensTemp[i] = sensTemp[j];
  sensTemp[j] = t;
  // swap addresses
  uint64_t a = sensAddr[i];
  sensAddr[i] = sensAddr[j];
  sensAddr[j] = a;
  // clear min and max
  memset(sensMin[i], 0, 6);
  memset(sensMax[i], 0, 6);
}


void OwTemp::printAddrRev(Print *printer, uint64_t addr) {
  uint8_t *a = (uint8_t *)&addr;
  printer->print("0x");
  for (byte b=7; b>=0; b--) {
    printer->print(a[b] >> 4, HEX);
    printer->print(a[b] & 0xF, HEX);
  }
}

void OwTemp::printAddr(Print *printer, uint64_t addr) {
  uint8_t *a = (uint8_t *)&addr;
  printer->print("0x");
  // print only the top 3 bytes, the rest is always 31010000 (at least for me)
  for (byte b=0; b<8; b++) {
    printer->print(*a >> 4, HEX);
    printer->print(*a & 0xF, HEX);
    a++;
  }
}

void OwTemp::printDebug(Print *printer) {
  printer->print(F("OwTemp has "));
  printer->print(sensCount);
  printer->print(F(" sensors, minMaxTimer:"));
  printer->print(minMaxTimer.remaining()/1000);
  printer->print(F("s, minMaxCount:"));
  printer->println(minMaxCount);

  for (uint8_t s=0; s<sensCount; s++) {
    printer->print("  #");
    printer->print(s);
    printer->print(": ");
    printAddr(printer, sensAddr[s]);
    if (sensAddr[s] != 0) { 
      printer->print(" now:");
      printer->print(sensTemp[s]);
      printer->print("F min:");
      for (uint8_t i=0; i<6; i++) {
        printer->print((int16_t)(sensMin[s][i])+TEMP_OFFSET);
        printer->print(",");
      }
      printer->print(F(" max:"));
      for (uint8_t i=0; i<6; i++) {
        printer->print((int16_t)(sensMax[s][i])+TEMP_OFFSET);
        printer->print(",");
      }
    }
    printer->println();
  }
}

// ===== Configuration =====

void OwTemp::applyConfig(uint8_t *cf) {
  if (cf) {
    if (sensAddr[0] == 0) {
      // address array is empty -> restore from EEPROM
      for (byte i=0; i<sensCount; i++) {
        // we store only the lower 32 bits in the EEPROM to save EEPROM space
        sensAddr[i] = ((uint32_t*)cf)[i];
      }
      //memcpy(sensAddr, cf, sizeof(uint64_t)*sensCount);
#     if DEBUG
      Serial.println(F("Config OwTemp: restored addrs from EEPROM"));
#     endif
    }
  } else {
    // no valid config in EEPROM, leave sensor address array as-is
  }
}

void OwTemp::receive(volatile uint8_t *pkt, uint8_t len) {
  // sorry, we ain't processing no packets...
}

// ===== One Wire utilities =====

// Set the resolution
void OwTemp::setresolution(uint64_t addr, byte bits) {
  // write scratchpad
  ds.reset();
  ds.select((uint8_t *)&addr);
  ds.write(0x4E, 0);
  ds.write(0, 0);                    // temp high
  ds.write(0, 0);                    // temp low
  ds.write(((bits-9)<<5) + 0x1F, 0); // configuration
  // copy to sensor's EEPROM
  ds.reset();
  ds.select((uint8_t *)&addr);
  ds.write(0x48, 1);                  // copy to eeprom with strong pullup
  delay(10);                          // needs 10ms
  ds.depower();
}

// Start temperature conversion
void OwTemp::start() {
  ds.reset();
  ds.skip();
  ds.write(0x44, 1);         // start conversion, with parasite power on at the end
  convState = 2;
}

// Raw reading of temperature, returns INT16_MIN on failure
int16_t OwTemp::rawRead(uint64_t addr) {
  byte data[12];
  ds.reset();
  ds.select((uint8_t *)&addr);    
  ds.write(0xBE);         // Read Scratchpad
  
  for (byte i = 0; i < 9; i++) {           // we need 9 bytes
    data[i] = ds.read();
  }
  if (OneWire::crc8(data, 8) != data[8]) {
#if DEBUG
    Serial.print(F("OWT: Bad CRC   for "));
    printAddr(&Serial, addr);
    Serial.print("->");
    printAddr(&Serial, *(uint64_t *)data);
    Serial.println();
#endif
    return INT16_MIN;
  }

  // handle missing sensor and double-check data
  if ((data[0] == 0 && data[1] == 0) || data[5] != 0xFF || data[7] != 0x10) {
#if DEBUG
    Serial.print(F("OWT: Bad data for "));
    printAddr(&Serial, addr);
    Serial.print("->");
    printAddr(&Serial, *(uint64_t *)data);
    Serial.println();
#endif
    return INT16_MIN;
  }

#if DEBUG
  Serial.print(F("OWT: Good data for "));
  printAddr(&Serial, addr);
  Serial.print("->");
  printAddr(&Serial, *(uint64_t *)data);
  Serial.println();
#endif
  
  // mask out bits according to precision of conversion
  int16_t raw = ((uint16_t)data[1] << 8) | data[0];
  int16_t t_mask[4] = {0x7, 0x3, 0x1, 0x0};
  byte cfg = (data[4] & 0x60) >> 5;
  raw &= ~t_mask[cfg];
  return raw;
}

// Read the temperature
float OwTemp::read(uint64_t addr) {
  int16_t raw = rawRead(addr);
  if (raw == INT16_MIN) raw = rawRead(addr);
  if (raw == INT16_MIN) raw = rawRead(addr);
  if (raw == INT16_MIN) raw = rawRead(addr);
  if (raw == INT16_MIN) return NAN;

  float celsius = (float)raw / 16.0;
  float fahrenheit = celsius * 1.8 + 32.0;
  
  return fahrenheit;
}
