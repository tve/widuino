// C 2013 Thorsten von Eicken
//
// One-wire relays controlled via DS2405 ICs.
//
// This module expects that the max number of switches is set at init time and that each switch
// is identified by an index. The first time the module runs, it populates the EEPROM with the
// one-wire addresses it finds (the unique 8-byte IDs assigned to each switch by Dallas Semi).
// The switches can then be reordered such that each switch index corresponds to the correct ID.
// On subsequent power-ups the known switches are located on the bus and a warning is printed
// if one is missing. If there is space, new switches are identified and added to the list.
// Switches are only removed from the list explicitly to avoid temporary hardware failures or
// glitches from causing switches to be lost from the list erroneously.
// In order to allow for changing out switches it is recommended to init the module with a count
// one higher than the number of actual switches, this leaves an empty spot for adding a
// replacement switch before deleting the failed one (it also makes it easier to temporarily
// add a switch for troubleshooting purposes).
// If a switch fails to respond to a command it will immediately be retried a couple of times.
//
// Currently only a single OwRelay object can be instantiated at a time because multiple ones
// would use the same EEPROM locations (this could be fixed easily).

#ifndef OwRelay_h
#define OwRelay_h

#define ONEWIRE_CRC8_TABLE 1
#include <OneWire.h>
#include <OwScan.h>

#define INT16_MIN ((int16_t)0x8000)

class OwRelay {
public:
  // Create OwRelay object based on OwScan object and for given max number of switches.
  OwRelay (OwScan *owScan, uint8_t max=2);

  // Poll all the switches every seconds interval. This can be called every iteration of the
  // wiring loop() function and keeps track of when an actual poll is necessary internally.
  // Use secs=0 to force a conversion ot start now
  // @return true if a conversion just finished
  bool loop(uint8_t secs);

  // Get the state of the nth switch
  bool get(uint8_t i);

  // Set the state of the nth switch, returns true on success
  bool set(uint8_t i, bool value);

  // Print debug info
  void printDebug(Print *printer);

private:
  OwScan *os;

  uint8_t rlyMax;                 // max number of relays
  //uint8_t rlyCount;               // number of switches
  bool *rlyState;                 // current state for each switch
  uint16_t failed;                // bit vector of failed switches
  unsigned long lastPoll;         // timestamp of last poll

  void init(OwScan *owScan, uint8_t count);           // helper for constructors
	void print(uint64_t addr);

	int8_t rawRead(uint64_t addr);
  bool rawWrite(uint64_t addr, bool value);
};

#endif
