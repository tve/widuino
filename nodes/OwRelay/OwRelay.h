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
#include <Config.h>

#define INT16_MIN ((int16_t)0x8000)

class OwRelay : public Configured {
public:
  // Create OwRelay object for given pin and max number of switches. Initializes the pin but
  // does not actually perform any one-wire communication
  OwRelay (byte pin, uint8_t count=2);

  // Create OwRelay object for given pin and max number of switches and also statically
  // configure the addresses of the switches. This causes setup() not to read or write
  // the EEPROM.
  OwRelay (byte pin, uint8_t count, uint64_t *addr);

  // Set everything up, starting with finding the existing and any new switches on the
  // One-Wire bus. Reads the old EEPROM config and updates it according to what it finds.
  // Setup can be called multiple times to update the config if switches are being added and
  // removed.
  // @return the number of switches found (may be higher than the number configured)
  uint8_t setup(Print *printer);

  // Poll all the switches every seconds interval. This can be called every iteration of the
  // wiring loop() function and keeps track of when an actual poll is necessary internally.
  // Use secs=0 to force a conversion ot start now
  // @return true if a conversion just finished
  bool loop(uint8_t secs);

  // Get the state of the nth switch
  bool get(uint8_t i);

  // Get the state of a switch by address
  bool getByAddr(uint64_t addr);

  // Set the state of the nth switch, returns true on success
  bool set(uint8_t i, bool value);

  // Swap the position of two switches
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

  // Get the One-Wire address of the nth switch
  uint64_t getAddr(uint8_t i);

  // Configuration methods
	virtual void applyConfig(uint8_t *);
	virtual void receive(volatile uint8_t *pkt, uint8_t len);


private:
  OneWire ds;

  uint8_t rlyCount;               // number of switches
  byte convState;                 // temp conversion state: 0=off, 1=idle, 2=converting
  unsigned long lastPoll;         // timestamp of last poll
  uint64_t *rlyAddr;              // switch addresses
  bool staticAddr;                // whether the addresses are static from the constructor
  bool *rlyState;                 // current state for each switch
  uint16_t failed;                // bit vector of failed switches

  void init(byte pin, uint8_t count);           // helper for constructors
	void print(uint64_t addr);

	int8_t read(uint64_t addr);
  bool write(uint64_t addr, bool value);
};

#endif
