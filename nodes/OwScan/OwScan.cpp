#include <JeeLib.h>
#include <OwScan.h>
#include <OneWire.h> // needed by makefile, ugh
#include <Config.h>
#include <Log.h>

#define DEBUG 0

// ===== Constructors =====

OwScan::OwScan(byte pin, uint8_t count) : ds(pin) {
  init(pin, count);
  devAddr = (uint64_t *)calloc(devMax, sizeof(uint64_t));
  staticAddr = false;
}

OwScan::OwScan(byte pin, uint8_t count, uint64_t *addr) : ds(pin) {
  init(pin, count);
  devAddr = addr;
  staticAddr = true;
}

void OwScan::init(byte pin, uint8_t count) {
  devMax = count < 16 ? count : 16;
	devCount = 0;
	present = 0;
  moduleId = OWSCAN_MODULE;
  configSize = sizeof(uint32_t)*devMax;
#if DEBUG
	Serial.print("OW: max=");
	Serial.print(devMax);
	Serial.print(" configSize=");
	Serial.print(configSize);
	Serial.println();
#endif
}

// ===== Operation =====

uint8_t OwScan::scan(Print *printer) {
  // run a search on the bus to see what we actually find
  uint64_t addr;                   // next detected device
  uint16_t found = 0;              // which addrs we actually found
  uint16_t added = 0;              // which addrs are new
  byte n_found = 0;                // number of devices actually discovered

#if DEBUG
	if (devCount == 0) {
		printer->print(F("OW <EEPROM:"));
		for (byte s=0; s<devMax; s++) {
			printer->print(" ");
			printAddr(printer, devAddr[s]);
		}
		printer->println();
	}
#endif

	ds.reset_search();
  while (ds.search((uint8_t *)&addr)) {
    // make sure the CRC is valid
		byte crc = OneWire::crc8((uint8_t *)&addr, 7);
    if (crc != (addr>>56)) continue;
    n_found++;

    // see whether we know this device already
    for (byte s=0; s<devMax; s++) {
      // we only restore 32 bits from EEPROM, so only compare that much...
      if ((uint32_t)addr == (uint32_t)(devAddr[s])) {
        devAddr[s] = addr;
#       if DEBUG
        printer->print("OW: found #");
        printAddr(printer, addr);
        printer->println();
#       endif
        found |= (uint16_t)1 << s;  // mark device as found
        goto cont;
      }
    }

    // new device, if we have space add it
    for (byte s=0; s<devMax; s++) {
      if (devAddr[s] == 0) {
        devAddr[s] = addr;
#       if DEBUG
        printer->print("OW: new #");
        printAddr(printer, addr);
        printer->println();
#       endif
        added |= (uint16_t)1 << s;  // mark device as added
        break;
      }
    }

  cont: ;
  }
	ds.reset_search();

	// Figure out how many devices we know
	devCount = devMax;
	while (devCount > 0 && devAddr[devCount-1] == 0)
		devCount--;

	// Print info if this is the first scan or if something has changed
	if (devCount == 0 || (found|added) != present) {
		// print info about additional devices found
		if (added) {
			printer->print(F("OW: New devices:    "));
			for (byte s=0; s<devMax; s++) {
				if (added & ((uint16_t)1 << s)) {
					printer->print(" ");
					printAddr(printer, devAddr[s]);
				}
			}
			printer->println();
		}

		// print info about missing devices
		uint16_t missing = (((uint16_t)1 << devCount)-1) & ~(found | added);
		if (missing) {
			printer->print(F("OW: Missing devices:"));
			for (byte s=0; s<devCount; s++) {
				if (missing & ((uint16_t)1 << s) && devAddr[s] != 0) {
					printer->print(" ");
					printAddr(printer, devAddr[s]);
				}
			}
			printer->println();
		}
	}
	present = found;

  printer->print(F("OW: found "));
  printer->print(n_found);
  printer->print(F(" of "));
  printer->print(devCount);
  printer->println(F(" devices"));

  // save the config in EEPROM
  uint32_t save[devMax];
  for (byte s=0; s<devMax; s++)
    save[s] = devAddr[s]; // loose top 32 bits
  config_write(OWSCAN_MODULE, save);

  return n_found;
}

// ===== Accessors =====

uint64_t OwScan::getAddr(uint8_t i) {
  return i < devCount ? devAddr[i] : 0;
}

void OwScan::swap(uint8_t i, uint8_t j) {
  if (i >= devCount || j >= devCount) return;
  // swap addresses
  uint64_t a = devAddr[i];
  devAddr[i] = devAddr[j];
  devAddr[j] = a;
}

void OwScan::printAddrRev(Print *printer, uint64_t addr) {
  uint8_t *a = (uint8_t *)&addr;
  printer->print("0x");
  for (int8_t b=7; b>=0; b--) {
    printer->print(a[b] >> 4, HEX);
    printer->print(a[b] & 0xF, HEX);
  }
}

void OwScan::printAddr(Print *printer, uint64_t addr) {
  uint8_t *a = (uint8_t *)&addr;
  printer->print("0x");
  // print only the top 4 bytes, the rest are boring
  for (byte b=0; b<4; b++) {
    printer->print(*a >> 4, HEX);
    printer->print(*a & 0xF, HEX);
    a++;
  }
}

void OwScan::printDebug(Print *printer) {
  printer->print(F("OwScan has "));
  printer->print(devCount);
  printer->println(F(" devices"));

  for (uint8_t s=0; s<devCount; s++) {
		if (devAddr[s] == 0) continue;
    printer->print("  #");
    printer->print(s);
    printer->print(": ");
    printAddr(printer, devAddr[s]);
    printer->println();
  }
}

// ===== Configuration =====

void OwScan::applyConfig(uint8_t *cf) {
  if (cf) {
    if (devAddr[0] == 0) {
      // address array is empty -> restore from EEPROM
      for (byte i=0; i<devMax; i++) {
        // we store only the lower 32 bits in the EEPROM to save EEPROM space
        devAddr[i] = ((uint32_t*)cf)[i];
      }
      //memcpy(devAddr, cf, sizeof(uint64_t)*devCount);
#     if DEBUG
      Serial.println(F("Config OwScan: restored addrs from EEPROM"));
#     endif
    }
  } else {
    // no valid config in EEPROM, leave device address array as-is
  }
}

void OwScan::receive(volatile uint8_t *pkt, uint8_t len) {
  // sorry, we ain't processing no packets...
}

// ===== One Wire utilities =====
