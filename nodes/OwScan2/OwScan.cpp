#include <JeeLib.h>
#include <OwScan.h>
#include <OneWire.h> // needed by makefile, ugh
#include <Logger.h>

#define DEBUG 0

static byte count_bits(uint32_t vector);
static void printPrefix(Print *printer, byte pin);

// ===== Scanner =====

uint32_t ow_scan(byte pin, uint64_t *devices, uint8_t max, Print *printer) {
  // run a search on the bus to see what we actually find
  uint64_t addr;                   // next detected device
  uint32_t expected = 0;           // which addrs we expect to find
  uint32_t found = 0;              // which addrs we actually found
  uint32_t added = 0;              // which addrs are new
  byte n_found = 0;                // number of devices actually discovered
  byte n_added = 0;                // number of devices added

  // tally which devices we expect from non-zero addresses in the array
  for (byte i=0; i<max; i++) {
    if (devices[i] != 0) expected |= (uint32_t)1 << i;
  }
  byte n_expected = count_bits(expected);

  OneWire ds(pin);
  ds.reset_search();
  while (ds.search((uint8_t *)&addr)) {
    // make sure the CRC is valid
    byte crc = OneWire::crc8((uint8_t *)&addr, 7);
    if (crc != (addr>>56)) continue;
    n_found++;
    uint64_t rev_addr = reverse_ow_addr(&addr);

    // see whether we know this device already
    for (byte s=0; s<max; s++) {
      if (rev_addr == devices[s]) {
        if (printer) {
          printPrefix(printer, pin);
          printer->print(F(" found #"));
          printer->print(s);
          printer->print(": ");
          print_ow_addr(printer, addr);
          printer->println();
        }
        found |= (uint32_t)1 << s;  // mark device as found
        goto cont;
      }
    }

    // new device, if we have space add it
    for (byte s=0; s<max; s++) {
      if (devices[s] == 0) {
        devices[s] = rev_addr;
        if (printer) {
          printPrefix(printer, pin);
          printer->print(F(" new #"));
          printer->print(s);
          printer->print(": ");
          print_ow_addr(printer, addr);
          printer->println();
        }
        added |= (uint32_t)1 << s;  // mark device as added
        n_added++;
        break;
      }
    }

    cont: ;
  }
  ds.reset_search();
  byte n_missing = n_expected + n_added - n_found;

  // print info about missing devices
  if (n_missing > 0) {
    printPrefix(printer, pin);
    printer->print(F(" missing "));
    for (byte s=0; s<max; s++) {
      if (~(found|added) & ((uint32_t)1 << s) && devices[s] != 0) {
        printer->print(" #");
        printer->print(s);
        printer->print(':');
        print_ow_addr(printer, devices[s]);
      }
    }
    printer->println();
  }

  if (printer) {
    printPrefix(printer, pin);
    printer->print(F(" found "));
    printer->print(n_found);
    printer->print(F(", expected "));
    printer->print(n_expected);
    printer->print(F(", added "));
    printer->print(n_added);
    printer->print(F(", missed "));
    printer->println(n_missing);
  }

  return found;
}

static byte count_bits(uint32_t vector) {
  byte n = 0;
  for (byte i=0; i<32; i++)
    if (vector & ((uint32_t)1<<i))
      n++;
  return n;
}

static void printPrefix(Print *printer, byte pin) {
  printer->print("OW[");
  printer->print(pin);
  printer->print("]:");
}

uint64_t reverse_ow_addr(const uint64_t *addr) {
  uint64_t rev;
  uint8_t *a=(uint8_t *)addr, *r=(uint8_t *)(&rev)+7;
  uint8_t i=8;
  while (i-- > 0) *r-- = *a++;
  return rev;
}

void print_ow_addr(Print *printer, uint64_t addr) {
  uint8_t *a = (uint8_t *)&addr;
  printer->print("0x");
  for (byte b=0; b<8; b++) {
    printer->print(*a >> 4, HEX);
    printer->print(*a & 0xF, HEX);
    a++;
  }
}
