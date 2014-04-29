// C 2013 Thorsten von Eicken
//
// One-Wire temperature sensors (typ DS18B20) connected to one pin. Use a 4.7K pullup for a
// simple network but for a more messy set of wires a 150ohm series and 1nF capacitor can be
// added to reduce reflections.
//
// This module expects that the max number of sensors is set at init time and that each sensor
// is identified by an index. The first time the module runs, it populates the EEPROM with the
// one-wire addresses it finds (the unique 8-byte IDs assigned to each sensor by Dallas Semi).
// The sensors can then be reordered such that each sensor index corresponds to the correct ID.
// Sensors are then polled automatically and the last value as well as 24-hr min/max can be
// queried anytime.
// On subsequent power-ups the known sensors are located on the bus and a warning is printed
// if one is missing. If there is space, new sensors are identified and added to the list.
// Sensors are only removed from the list explicitly to avoid temporary hardware failures or
// glitches from causing sensors to be lost from the list erroneously.
// In order to allow for changing out sensors it is recommended to init the module with a count
// one higher than the number of actual sensors, this leaves an empty spot for adding a
// replaceent sensor before deleting the failed one (it also makes it easier to temporarily
// add a sensor for troubleshooting purposes).
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

#ifndef OwTemp_h
#define OwTemp_h

#define ONEWIRE_CRC8_TABLE 1
#include <OneWire.h>
#include <Config.h>

#define INT16_MIN ((int16_t)0x8000)

class OwTemp : public Configured {
public:
  // Create OWTemp object for given pin and max number of sensors. Initializes the pin but
  // does not actually perform any one-wire communication
  OwTemp (byte pin, uint8_t count=2);

  // Create OWTemp object for given pin and max number of sensors and also statically
  // configure the addresses of the sensors. This causes setup() not to read or write
  // the EEPROM.
  OwTemp (byte pin, uint8_t count, uint64_t *addr);

  // Set everything up, starting with finding the existing and any new sensors on the
  // One-Wire bus. Reads the old EEPROM config and updates it according to what it finds.
  // Setup can be called multiple times to update the config if sensors are being added and
  // removed.
  // @return the number of sensors found (may be higher than the number configured)
  uint8_t setup(Print *printer);

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

  // Swap the position of two sensors
  void swap(uint8_t i, uint8_t j);

  // Print the address in hex in standard one-wire order (high byte = family first)
  // The printer must implement the Print interface, Serial is one example for this.
  void printAddr(Print *printer, uint64_t addr);

  // Print the address in hex in reverse order, which is how constants need to be
  // entered into uint64_t variables.
  void printAddrRev(Print *printer, uint64_t addr);

  // Print debug info
  void printDebug(Print *printer);

  // ---- lower level methods ----

  // Get the temperature from a sensor by address
  float getByAddr(uint64_t addr);

  // Get the One-Wire address of the nth sensor
  uint64_t getAddr(uint8_t i);

  // Configuration methods
	virtual void applyConfig(uint8_t *);
	virtual void receive(volatile uint8_t *pkt, uint8_t len);

private:
  OneWire ds;

  uint8_t sensCount;              // number of sensors
  byte convState;                 // temp conversion state: 0=off, 1=idle, 2=converting
  unsigned long lastConv;         // timestamp of last conversion
  uint64_t *sensAddr;             // sensor addresses
  bool staticAddr;                // whether the addresses are static from the constructor
  float *sensTemp;                // current temperature for each sensor
  uint16_t failed;                // bit vector of failed sensors

  MilliTimer minMaxTimer;
  uint16_t minMaxCount;
  int8_t (*sensMin)[6];           // min/max temps (-88 offset -> supports -40F..215F)
  int8_t (*sensMax)[6];

  void init(byte pin, uint8_t count);           // helper for constructors
	void setresolution(uint64_t addr, uint8_t bits); // set the resolution of a sensor
	void start();
	int16_t rawRead(uint64_t addr);
	float read(uint64_t addr);
	void print(uint64_t addr);
};

#endif
