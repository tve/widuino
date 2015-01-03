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

uint32_t ow_scan(byte pin, uint64_t *devices, uint8_t max, Print *printer);

// Print the address in hex in standard one-wire order (high byte = family first)
// The printer must implement the Print interface, Serial is one example for this.
void print_ow_addr(Print *printer, uint64_t addr);

uint64_t reverse_ow_addr(const uint64_t *addr);

inline uint8_t get_ow_type(const uint64_t *addr) { return ((uint8_t*)addr)[7]; }

#endif
