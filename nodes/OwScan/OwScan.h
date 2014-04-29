// C 2013 Thorsten von Eicken
//
// One-wire bus scan
//
// This module expects that the max number of devices is set at init time and that each device
// is identified by an index. The first time the module runs, it populates the EEPROM with the
// one-wire addresses it finds (the unique 8-byte IDs assigned to each device by Dallas Semi).
// The devices can then be reordered such that each device index corresponds to the desired ID.
// On subsequent power-ups the known devices are located on the bus and a warning is printed
// if one is missing. If there is space, new devices are identified and added to the list.
// Devices are only removed from the list explicitly to avoid temporary hardware failures or
// glitches from causing devices to be lost from the list erroneously.
// In order to allow for changing out devices it is recommended to init the module with a count
// one higher than the number of actual devices, this leaves an empty spot for adding a
// replacement device before deleting the failed one (it also makes it easier to temporarily
// add a device for troubleshooting purposes).
// 
// Currently only a single OwScan object can be instantiated at a time because multiple ones
// would use the same EEPROM locations (this could be fixed easily).

#ifndef OwScan_h
#define OwScan_h

#define ONEWIRE_CRC8_TABLE 1
#include <OneWire.h>
#include <Config.h>

#define INT16_MIN ((int16_t)0x8000)

class OwScan : public Configured {
public:
  // Create OwScan object for given pin and max number of devices. Initializes the pin but
  // does not actually perform any one-wire communication
  OwScan (byte pin, uint8_t count=2);

  // Create OwScan object for given pin and max number of devices and also statically
  // configure the addresses of the devices. This causes setup() not to read or write
  // the EEPROM.
  OwScan (byte pin, uint8_t count, uint64_t *addr);

  // Scan the 1-wire bus. Find the existing and any new devices on the One-Wire bus.
  // Reads the old EEPROM config and updates it according to what it finds.
  // Scan can be called multiple times to update the config if devices are being added and
  // removed.
  // @return the number of devices found (may be higher than the number configured)
  uint8_t scan(Print *printer);

  // Swap the position of two devices
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
  
  uint8_t getCount() { return devCount; }
  uint8_t getMax() { return devMax; }
  OneWire getOneWire() { return ds; }

  // Get the One-Wire address of the nth device
  uint64_t getAddr(uint8_t i);

  // Configuration methods
  virtual void applyConfig(uint8_t *);
  virtual void receive(volatile uint8_t *pkt, uint8_t len);

private:
  OneWire ds;

  uint8_t devMax;                 // max number of devices 
  uint8_t devCount;               // number of devices (for which we have addr)
  uint64_t *devAddr;              // device addresses
  bool staticAddr;                // whether the addresses are static from the constructor
  uint16_t present;               // bit vector of present devices

  void init(byte pin, uint8_t count);           // helper for constructors
  void print(uint64_t addr);
};

#endif
