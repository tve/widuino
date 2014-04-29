// C 2013 Thorsten von Eicken
//
// One-Wire temperature sensors (typ DS18B20) connected to one pin. Use a 4.7K pullup for a
// simple network but for a more messy set of wires a 150ohm series and 1nF capacitor can be
// added to reduce reflections.
//
// If a sensor fails to respond to a conversion request it will immediately be retried a
// couple of times. After that the old temperature remains unchanged but a flag is set internally.
// If the next poll also fails, a NAN (not a number) floating point value will be returned for
// the sensor. If the failure persists for hours the min/max will remain frozen at their last
// values.
// 
// This module keeps track of the minimum and maximum temperature over the past 24 hours for
// each sensor in a relatively simplistic way. It does this by keeping min/max for 6 4-hour
// periods and shifts these every 4 hours. It also keeps the min/max as integers to save space.
//
// Currently only a single OwTemp object can be instantiated at a time because multiple ones
// would use the same EEPROM locations (this could be fixed easily).
// This module operates in farenheit, a change to centigrade is trivial for the current
// temperatures but may need some tweaking for the 24-hr min/max if fractional temps are
// desired.

#ifndef OwTemp2_h
#define OwTemp2_h

#define ONEWIRE_CRC8_TABLE 1
#include <OneWire.h>
#include <OwScan.h>

#define INT16_MIN ((int16_t)0x8000)

class OwTemp {
public:
  // Create OWTemp object based on OwScan object and for given max number of sensors.
  OwTemp (OwScan *owScan, uint8_t max=2);

  // Poll all the sensors every seconds interval. This can be called every iteration of the
  // wiring loop() function and keeps track of when an actual poll is necessary internally.
  // Use secs=0 to force a conversion ot start now
  // @return true if a conversion just finished
  bool loop(uint8_t secs);

  // Get the current temperature for the nth sensor
  float get(uint8_t i);

  // Get the 24-hr minimum temperature for the nth sensor (note integer!)
  uint16_t getMin(uint8_t i);

  // Get the 24-hr maximum temperature for the nth sensor (note integer!)
  uint16_t getMax(uint8_t i);

  // ---- lower level methods ----

  // Get the temperature from a sensor by address
  //float getByAddr(uint64_t addr);

  // Get the One-Wire address of the nth sensor
  //uint64_t getAddr(uint8_t i);

  void printDebug(Print *printer);

private:
  OwScan *os;

  uint8_t tempMax;                // max number of sensors
  uint8_t *map;                   // map from OwSens index to sensTemp/sensMin/senseMax index
  byte convState;                 // temp conversion state: 0=off, 1=idle, 2=converting
  unsigned long lastConv;         // timestamp of last conversion
  float *sensTemp;                // current temperature for each sensor
  uint16_t failed;                // bit vector of failed sensors

  MilliTimer minMaxTimer;
  uint16_t minMaxCount;
  int8_t (*sensMin)[6];           // min/max temps (-88 offset -> supports -40F..215F)
  int8_t (*sensMax)[6];

  void init(OwScan *owScan, uint8_t count);        // helper for constructors
  void setresolution(uint64_t addr, uint8_t bits); // set the resolution of a sensor
  void start();
  int16_t rawRead(uint64_t addr);
  float read(uint64_t addr);
  void print(uint64_t addr);
};

#endif
