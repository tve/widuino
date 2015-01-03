#include <JeeLib.h>
#include <OwTemp.h>
#include <OwScan.h>
#include <OneWire.h>
#include <Logger.h>

#define OWTEMP_CONVTIME 188 // milliseconds for a conversion (10 bits)
#define TEMP_OFFSET      88 // offset used to store min/max in 8 bits

#define MAX_COUNT        32 // max number of sensors, limited due to bitfield ops

#define DEBUG 0

// ===== Constructors =====

OwTemp::OwTemp(uint8_t pin, uint64_t *addr, uint8_t max) {
  init(pin, addr, max);
}

void OwTemp::init(uint8_t pin, uint64_t *addr, uint8_t max) {
  ds = new OneWire(pin);
  tempAddr = addr;
  tempMax = max < MAX_COUNT ? max : MAX_COUNT;
  convState = 0;
  failed = 0;
  sensTemp = (float *)calloc(tempMax, sizeof(float));
  sensMin  = (int8_t (*)[6])calloc(tempMax, 6*sizeof(int8_t));
  memset(sensMin, 0x80, tempMax*6*sizeof(int8_t));
  sensMax  = (int8_t (*)[6])calloc(tempMax, 6*sizeof(int8_t));
  memset(sensMax, 0x80, tempMax*6*sizeof(int8_t));
#if DEBUG
  logger->print("OWT: max=");
  logger->print(tempMax);
  logger->println();
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
      for (uint8_t s=0; s<tempMax; s++) {
        for (uint8_t i=5; i>0; i--) {
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
    // count the number of actual temperature sensors
    uint8_t n_found = 0;
    for (uint8_t s=0; s<tempMax; s++) {
      if (isTemp(s)) n_found++;
    }
    // start a conversion
    if (n_found > 0) {
            start();
            lastConv = now;
            convState = 1;
    }

  } else if (convState == 1 && now - lastConv > OWTEMP_CONVTIME) {
    // time to read the results
    for (uint8_t s=0; s<tempMax; s++) {
      if (!isTemp(s)) continue;

      float t = read(s);
      uint32_t bit = (uint32_t)1 << s;
      if (isnan(t)) {
        // conversion failed
        if (failed & bit) {
          // sensor has been failing, make it NaN
          sensTemp[s] = NAN;
        } else {
          // first time sensor failed, just keep old value
          failed |= bit;
        }
      } else {
        // conversion succeeded
        failed &= ~bit;
        sensTemp[s] = t;
        // update min/max
        int8_t m = (int8_t)(round(t)-TEMP_OFFSET);
#if DEBUG
        logger->print("OWT: "); logger->print(s); logger->print(" -> ");
        logger->print(t); logger->println();
        logger->print("Min: "); logger->print(m+TEMP_OFFSET); logger->print(" ");
        logger->print(sensMin[s][0]+TEMP_OFFSET); logger->print(" ");
        logger->println(m < sensMin[s][0]);
#endif
        if (sensMin[s][5] == -128)
          memset(sensMin[s], m, 6); // sensMin is uninitialized so set it all
        else if (sensMin[s][0] == -128 || m < sensMin[s][0])
          sensMin[s][0] = m;        // new minimum
#if DEBUG
        logger->print("Max: "); logger->print(m+TEMP_OFFSET); logger->print(" ");
        logger->print(sensMax[s][0]+TEMP_OFFSET); logger->print(" ");
        logger->println(m > sensMax[s][0]);
#endif
        if (sensMax[s][5] == -128)
          memset(sensMax[s], m, 6); // sensMax is uninitialized so set it all
        else if (m > sensMax[s][0])
          sensMax[s][0] = m;        // new maximum
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

bool OwTemp::isTemp(uint8_t s) {
  uint8_t dev = get_ow_type(tempAddr+s);
  return dev == 0x22 || dev == 0x28;
}

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

  for (uint8_t s=0; s<tempMax; s++) {
    if (!isTemp(s)) continue;
    printer->print(s);
    //print_ow_addr(printer, tempAddr[s]);
    printer->print(": ");
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
    printer->println();
  }
}

// ===== One Wire utilities =====

// Set the resolution
void OwTemp::setresolution(uint64_t addr, byte bits) {
  addr = reverse_ow_addr(&addr);
  // write scratchpad
  ds->reset();
  ds->select((uint8_t *)&addr);
  ds->write(0x4E, 0);
  ds->write(0, 0);                    // temp high
  ds->write(0, 0);                    // temp low
  ds->write(((bits-9)<<5) + 0x1F, 0); // configuration
  // copy to sensor's EEPROM
  ds->reset();
  ds->select((uint8_t *)&addr);
  ds->write(0x48, 1);                  // copy to eeprom with strong pullup
  delay(10);                          // needs 10ms
  ds->depower();
}

// Start temperature conversion
void OwTemp::start() {
  ds->reset();
  ds->skip();
  ds->write(0x44, 1);         // start conversion, with parasite power on at the end
}

// Raw reading of temperature, returns INT16_MIN on failure
int16_t OwTemp::rawRead(uint64_t addr) {
  uint64_t rev_addr = reverse_ow_addr(&addr);
  byte data[12];
  ds->reset();
  ds->select((uint8_t *)&rev_addr);
  ds->write(0xBE);         // Read Scratchpad

  for (byte i = 0; i < 9; i++) {           // we need 9 bytes
    data[i] = ds->read();
  }
  if (OneWire::crc8(data, 8) != data[8]) {
#if DEBUG
    logger->print(F("OWT: Bad CRC   for "));
    print_ow_addr(logger, addr);
    logger->print("->");
    print_ow_addr(logger, *(uint64_t *)data);
    logger->println();
#endif
    return INT16_MIN;
  }

  // handle missing sensor and double-check data
  if ((data[0] == 0 && data[1] == 0) || data[5] != 0xFF || data[7] != 0x10) {
#if DEBUG
    logger->print(F("OWT: Bad data for "));
    print_ow_addr(logger, addr);
    logger->print("->");
    print_ow_addr(logger, *(uint64_t *)data);
    logger->println();
#endif
    return INT16_MIN;
  }

#if DEBUG
  logger->print(F("OWT: Good data for "));
  print_ow_addr(logger, addr);
  logger->print("->");
  print_ow_addr(logger, *(uint64_t *)data);
  logger->println();
#endif

  // mask out bits according to precision of conversion
  int16_t raw = ((uint16_t)data[1] << 8) | data[0];
  int16_t t_mask[4] = {0x7, 0x3, 0x1, 0x0};
  byte cfg = (data[4] & 0x60) >> 5;
  raw &= ~t_mask[cfg];
  return raw;
}

// Read the temperature
float OwTemp::read(uint8_t s) {
  uint64_t addr = tempAddr[s];
  int16_t raw = rawRead(addr);
  if (raw == INT16_MIN) raw = rawRead(addr);
  if (raw == INT16_MIN) raw = rawRead(addr);
  if (raw == INT16_MIN) raw = rawRead(addr);
  if (raw == INT16_MIN || raw == 0x0550) return NAN; // 0x0550 is power-on value

  float celsius = (float)raw / 16.0;
  float fahrenheit = celsius * 1.8 + 32.0;
#if DEBUG
  logger->print("OWT: ");
  print_ow_addr(logger, addr);
  logger->print(" has ");
  logger->print(fahrenheit);
  logger->println("F");
#endif

  return fahrenheit;
}
