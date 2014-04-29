// C 2013 Thorsten von Eicken
//

#ifndef OwMisc_h
#define OwMisc_h

#include <OwScan.h>

class OwMisc {
public:
  // Create OwMisc object based on OwScan object
  OwMisc (OwScan *owScan);

	// Read one of the 4 counters of a ds2423
  uint32_t ds2423GetCount(uint8_t ix, uint8_t counter);

	// Read the Vad ADC of a DS2438
	int16_t ds2438GetVad(uint8_t ix);

	// Read the Vsense ADC of a DS2438
	int16_t ds2438GetVsense(uint8_t ix);

private:
  OwScan *os;
  void ds2438Config(OneWire ds, uint64_t addr);
  bool ds2438ReadPage(OneWire ds, uint64_t addr, uint8_t page, uint8_t data[9]);

	//void print(uint64_t addr);
};

#endif
