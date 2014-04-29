#include <JeeLib.h>
#include <OwTemp.h>
#include <Config.h>
#include <Log.h>

#define OWTEMP_CONVTIME 188 // milliseconds for a conversion (10 bits)
#define TEMP_OFFSET      88 // offset used to store min/max in 8 bits

#define MAX_COUNT        16 // max number of sensors, limited due to bitfield ops

#define DEBUG 0

// ===== Constructors =====

OwTemp::OwTemp(OwScan *owScan, uint8_t max) {
  init(owScan, max);
}

void OwTemp::init(OwScan *owScan, uint8_t max) {
	os = owScan;
  tempMax = max < MAX_COUNT ? max : MAX_COUNT;
  convState = 0;
  failed = 0;
	map = (uint8_t *)calloc(owScan->getMax(), sizeof(uint8_t));
	memset(map, 0xFF, owScan->getMax()*sizeof(uint8_t));
  sensTemp = (float *)calloc(tempMax, sizeof(float));
  sensMin  = (int8_t (*)[6])calloc(tempMax, 6*sizeof(int8_t));
  memset(sensMin, 0x80, tempMax*6*sizeof(int8_t));
  sensMax  = (int8_t (*)[6])calloc(tempMax, 6*sizeof(int8_t));
  memset(sensMax, 0x80, tempMax*6*sizeof(int8_t));
#if DEBUG
	Serial.print("OWT: max=");
	Serial.print(tempMax);
	Serial.println();
#endif
}

// ===== Operation =====

// Poll temperature sensors every <secs> seconds; use secs=0 to force conversion now
bool OwTemp::loop(uint8_t secs) {
  // rotate min/max temp every 4 hours
  if (minMaxTimer.poll(60000)) {
    minMaxCount++;
    if (minMaxCount >= 4*60) {
      minMaxCount = 0;
      // rotate min/max temps
      for (int s=0; s<tempMax; s++) {
        for (int i=5; i>0; i--) {
          sensMin[s][i] = sensMin[s][i-1];
          sensMax[s][i] = sensMax[s][i-1];
        }
				sensMin[s][0] = 0x80;
				sensMax[s][0] = 0x80;
      }
    }
  }

  unsigned long now = millis();
	if (convState == 0) {
		if (secs != 0 && lastConv != 0 && now - lastConv <= (unsigned long)secs * 1000) {
			// not time to do any conversion
			return false;
		}
		// update mapping from OwScan index to sensIx
		uint8_t n_found = 0;
		for (uint8_t s=0; s<os->getCount(); s++) {
			// see whether it's a temperature sensor
			uint8_t dev = (uint8_t)os->getAddr(s);
			if (dev == 0x22 || dev == 0x28) {
				if (map[s] != n_found) {
#if DEBUG
					Serial.print("OWT: new temp ");
					Serial.print(s);
					Serial.print("->");
					Serial.print(n_found);
					Serial.print(" ");
					os->printAddr((Print *)&Serial, os->getAddr(s));
					Serial.println();
#endif
					// change in mapping
					map[s] = n_found;
					// zero out current, min/max
					sensTemp[n_found] = NAN;
					memset(sensMin[n_found], 0x80, 6*sizeof(int8_t));
					memset(sensMax[n_found], 0x80, 6*sizeof(int8_t));
					// ensure we're running at 10 bit resolution
					setresolution(os->getAddr(s), 10);
				}
				n_found++;
			} else {
				map[s] = 0xff;
			}
		}
		// start a conversion
		if (n_found > 0) {
			start();
			lastConv = now;
			convState = 1;
		}

	} else if (convState == 1 && now - lastConv > OWTEMP_CONVTIME) {
		// time to read the results
		for (byte s=0; s<os->getCount(); s++) {
			byte ix = map[s];
			if (ix == 0xff) continue;
			float t = read(os->getAddr(s));
			uint16_t bit = (uint16_t)1 << s;
			if (isnan(t)) {
				// conversion failed
				if (failed & bit) {
					// sensor has been failing, make it NaN
					sensTemp[ix] = NAN;
				} else {
					// first time sensor failed, just keep old value
					failed |= bit;
				}
			} else {
				// conversion succeeded
				failed &= ~bit;
				sensTemp[ix] = t;
				// update min/max
				int8_t m = (int8_t)(round(t)-TEMP_OFFSET);
#if DEBUG
				Serial.print("OWT: "); Serial.print(ix); Serial.print(" -> ");
				Serial.print(t); Serial.println();
				Serial.print("Min: "); Serial.print(m+TEMP_OFFSET); Serial.print(" ");
				Serial.print(sensMin[ix][0]+TEMP_OFFSET); Serial.print(" ");
				Serial.println(m < sensMin[ix][0]);
#endif
				if (sensMin[ix][5] == -128)
					memset(sensMin[ix], m, 6); // sensMin is uninitialized so set it all
				else if (sensMin[ix][0] == -128 || m < sensMin[ix][0])
					sensMin[ix][0] = m;        // new minimum
#if DEBUG
				Serial.print("Max: "); Serial.print(m+TEMP_OFFSET); Serial.print(" ");
				Serial.print(sensMax[ix][0]+TEMP_OFFSET); Serial.print(" ");
				Serial.println(m > sensMax[ix][0]);
#endif
				if (sensMax[ix][5] == -128)
					memset(sensMax[ix], m, 6); // sensMax is uninitialized so set it all
				else if (m > sensMax[ix][0])
					sensMax[ix][0] = m;        // new maximum
			}
		}
		convState = 0;
		return true;
	}

	return false;
}

// ===== Accessors =====

float OwTemp::get(uint8_t i) {
  return i < tempMax ? sensTemp[i] : NAN;
}

/*
float OwTemp::getByAddr(uint64_t addr) {
  for (byte s=0; s<sensCount; s++) {
    if (sensAddr[s] == addr) return sensTemp[s];
  }
  return NAN;
}

uint64_t OwTemp::getAddr(uint8_t i) {
  return i < sensCount ? sensAddr[i] : NAN;
}
*/

uint16_t OwTemp::getMin(uint8_t i) {
  if (i >= tempMax) return 0x8000;
  int8_t t = sensMin[i][0];
  for (uint8_t h=1; h<6; h++)
    if (sensMin[i][h] < t)
      t = sensMin[i][h];
  return t + TEMP_OFFSET;
}

uint16_t OwTemp::getMax(uint8_t i) {
  if (i >= tempMax) return 0x8000;
  int8_t t = sensMax[i][0];
  for (uint8_t h=1; h<6; h++)
    if (sensMax[i][h] > t)
      t = sensMax[i][h];
  return t + TEMP_OFFSET;
}

void OwTemp::printDebug(Print *printer) {
  printer->print(F("OwTemp has "));
  printer->print(tempMax);
  printer->print(F(" sensors, minMaxTimer:"));
  printer->print(minMaxTimer.remaining()/1000);
  printer->print(F("s, minMaxCount:"));
  printer->println(minMaxCount);

  for (uint8_t s=0; s<os->getCount(); s++) {
		uint8_t ix = map[s];
		if (ix == 0xff) continue;
    printer->print("  #");
    printer->print(ix);
    printer->print(": ");
    os->printAddr(printer, os->getAddr(s));
		printer->print(" now:");
		printer->print(sensTemp[ix]);
		printer->print("F min:");
		for (uint8_t i=0; i<6; i++) {
			printer->print((int16_t)(sensMin[ix][i])+TEMP_OFFSET);
			printer->print(",");
		}
		printer->print(F(" max:"));
		for (uint8_t i=0; i<6; i++) {
			printer->print((int16_t)(sensMax[ix][i])+TEMP_OFFSET);
			printer->print(",");
    }
    printer->println();
  }
}

// ===== One Wire utilities =====

// Set the resolution
void OwTemp::setresolution(uint64_t addr, byte bits) {
	OneWire ds = os->getOneWire();
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
	OneWire ds = os->getOneWire();
  ds.reset();
  ds.skip();
  ds.write(0x44, 1);         // start conversion, with parasite power on at the end
}

// Raw reading of temperature, returns INT16_MIN on failure
int16_t OwTemp::rawRead(uint64_t addr) {
	OneWire ds = os->getOneWire();
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
    os->printAddr(&Serial, addr);
    Serial.print("->");
    os->printAddr(&Serial, *(uint64_t *)data);
    Serial.println();
#endif
    return INT16_MIN;
  }

  // handle missing sensor and double-check data
  if ((data[0] == 0 && data[1] == 0) || data[5] != 0xFF || data[7] != 0x10) {
#if DEBUG
    Serial.print(F("OWT: Bad data for "));
    os->printAddr(&Serial, addr);
    Serial.print("->");
    os->printAddr(&Serial, *(uint64_t *)data);
    Serial.println();
#endif
    return INT16_MIN;
  }

#if DEBUG
  Serial.print(F("OWT: Good data for "));
  os->printAddr(&Serial, addr);
  Serial.print("->");
  os->printAddr(&Serial, *(uint64_t *)data);
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
  if (raw == INT16_MIN || raw == 0x0550) return NAN; // 0x0550 is power-on value

  float celsius = (float)raw / 16.0;
  float fahrenheit = celsius * 1.8 + 32.0;
#if DEBUG
	Serial.print("OWT: ");
	os->printAddr((Print *)&Serial, addr);
	Serial.print(" has ");
	Serial.print(fahrenheit);
	Serial.println("F");
#endif
  
  return fahrenheit;
}
