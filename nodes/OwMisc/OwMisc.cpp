#include <util/crc16.h>
#include <JeeLib.h>
#include <OwMisc.h>
//#include <Config.h>
#include <Log.h>

#define DEBUG 0

#define DS2438_CONFIG 0x01      // AD=Vad, EE=off, CA=off, IAD=on

// ===== Constructors =====

OwMisc::OwMisc(OwScan *owScan) {
	os = owScan;
}

// ===== DS2423 Dual Counter =====

uint32_t OwMisc::ds2423GetCount(uint8_t ix, uint8_t counter) {
	OneWire ds = os->getOneWire();
	uint64_t addr = os->getAddr(ix);
	uint16_t crc = 0, crc2;
	if ((uint8_t)addr != 0x1D) return NAN;

	// Read the counter
  ds.reset();
  ds.select((uint8_t *)&addr);
  ds.write(0xA5, 0);
	crc = _crc16_update(crc, 0xA5);
	uint8_t low_addr = (counter&1<<6)+0x9f;
  ds.write(low_addr, 0);             // low address, start with last byte of page
	crc = _crc16_update(crc, low_addr);
  ds.write(0x01, 0);                 // high address
	crc = _crc16_update(crc, 1);
	crc = _crc16_update(crc, ds.read()); // read last byte of page

	uint32_t value = 0;
	for (uint8_t i=0; i<4; i++) {
		uint8_t v = ds.read();
		value |= (uint32_t)v << (8*i);
		crc = _crc16_update(crc, v);
	}
	for (uint8_t i=0; i<4; i++) {
		uint8_t v = ds.read();  				 // read 32 bits of 0's
		crc = _crc16_update(crc, v);
	}
	crc2 = ds.read();
	crc2 |= (uint16_t)ds.read() << 8;
	ds.reset();
	bool crc_ok = crc == ~crc2;

#if DEBUG
	Serial.print("OW DS2423.");
	Serial.print(counter&1);
	Serial.print(" @");
	os->printAddr((Print *)&Serial, addr);
	Serial.print(" = ");
	Serial.print(value);
	Serial.print(" crc=0x");
	Serial.print(crc2, 16);
	if (crc_ok) {
		Serial.print(" OK");
	} else {
		Serial.print(" ERR: calc_crc=0x");
		Serial.print(~crc, 16);
	}
	Serial.println();
#endif
	return crc_ok ? value : -1;
}

// Check and configure the DS2438 so we can read its two ADC inputs
// Returns true if all is OK
void OwMisc::ds2438Config(OneWire ds, uint64_t addr) {
	// write the control register in the scratchpad
  ds.reset();
  ds.select((uint8_t *)&addr);
  ds.write(0x4E, 0);                 // write scratchpad
  ds.write(0x00, 0);                 // page 0
  ds.write(DS2438_CONFIG, 0);

	// copy scratchpad to memory
	ds.reset();												 // read last byte of page
  ds.select((uint8_t *)&addr);
  ds.write(0x48, 0);                 // copy to memory
  ds.write(0x00, 0);                 // page 0

	ds.reset();												 // read last byte of page
#if DEBUG
	Serial.print(F("DS2438 "));
	os->printAddr(&Serial, addr);
	Serial.println(F(" configured"));
#endif
}

// Read a page from the DS2438 and check the CRC
// return true if the CRC is OK
bool OwMisc::ds2438ReadPage(OneWire ds, uint64_t addr, uint8_t page, uint8_t data[9]) {
	// copy memory to scratchpad
	ds.reset();												 // read last byte of page
  ds.select((uint8_t *)&addr);
  ds.write(0xB8, 0);                 // copy from memory
  ds.write(0x00, 0);                 // page 0

	// Read the page
	ds.reset();
	ds.select((uint8_t *)&addr);
	ds.write(0xBE, 0);                 // read scratchpad
	ds.write(page, 0);
	for (uint8_t i=0; i<9; i++)
		data[i] = ds.read();
	ds.reset();

	// check CRC
  if (OneWire::crc8(data, 8) != data[8]) {
#if DEBUG
    Serial.print(F("DS2438: Bad CRC for "));
    os->printAddr(&Serial, addr);
    Serial.print(F(" -> "));
    os->printAddr(&Serial, *(uint64_t *)data);
    Serial.println();
#endif
    return false;
  }

#if DEBUG == 2
	Serial.print(F("DS2438 @"));
	os->printAddr(&Serial, addr);
	Serial.print(F(" -> "));
	os->printAddrRev(&Serial, *(uint64_t *)data);
	Serial.print(" ");
	Serial.print(data[8], 16);
	Serial.println();
#endif
	return true;
}

// Read the Vad ADC of a DS2438 in millivolt
int16_t OwMisc::ds2438GetVad(uint8_t ix) {
	OneWire ds = os->getOneWire();
	uint64_t addr = os->getAddr(ix);
	if ((uint8_t)addr != 0x26) return -1;

	// read page 0 
	uint8_t data[9];
	if (!ds2438ReadPage(ds, addr, 0, data)) return -1;
	// check whether we need to configure the beast
	if ((data[0] & 0x0f) != DS2438_CONFIG) {
		//Serial.print("R=0x");
		//Serial.println(data[0], 16);
		ds2438Config(ds, addr);
	}

	// start voltage conversion
  ds.reset();
  ds.select((uint8_t *)&addr);
  ds.write(0xB4, 0);                 // convert V command

	// now keep reading until it tells us that the conversion is done
	//Serial.print("RD=");
	//Serial.print(ds.read(), 16);
	//Serial.println();
	uint8_t n=0;
	while (--n > 0 && ds.read() != 0xff)
		delayMicroseconds(100);
	ds.reset();
	if (n == 0) return -1;
	// read ADC value
	if (!ds2438ReadPage(ds, addr, 0, data)) return -1;
	uint16_t v = ( ((uint16_t)data[4]<<8) | (uint16_t)data[3] ) * 10;

#if DEBUG
	// start temperature conversion for grins
  ds.reset();
  ds.select((uint8_t *)&addr);
  ds.write(0x44, 0);                 // convert T command
  ds.reset();
	delay(11);
	Serial.print("DS2438 @");
	os->printAddr(&Serial, addr);
	Serial.print(" T=");
	Serial.print((int8_t)data[2]);
	Serial.print("C Vad=");
	Serial.print(v);
	Serial.print("mV");
	Serial.println();
#endif
	return (int16_t)v;
}

// Read the Vsense ADC of a DS2438
int16_t OwMisc::ds2438GetVsense(uint8_t ix) {
	OneWire ds = os->getOneWire();
	uint64_t addr = os->getAddr(ix);
	if ((uint8_t)addr != 0x26) return -1;

	// read page 0 
	uint8_t data[9];
	if (!ds2438ReadPage(ds, addr, 0, data)) return -1;
	// check whether we need to configure the beast
	if ((data[0] & 0x0f) != DS2438_CONFIG) {
		//Serial.print("R=0x");
		//Serial.println(data[0], 16);
		ds2438Config(ds, addr);
		return -1; // we'll have to wait before it actually performs a measurement...
	}

	// return Vsense value
	int16_t v = ( ((int16_t)data[6]<<8) | (uint16_t)data[5] );

#if DEBUG
	Serial.print("DS2438 @");
	os->printAddr(&Serial, addr);
	Serial.print(" Vsense=");
	Serial.print((float)v * 0.2441);
	Serial.print("mV");
	Serial.println();
#endif
	return v;
}

// ===== One Wire utilities =====


#if 0
// Raw reading of temperature, returns INT16_MIN on failure
int16_t OwMisc::rawRead(uint64_t addr) {
	OneWire ds = os->getOneWire();
  byte data[12];
  ds.reset();
  ds.select((uint8_t *)&addr);    
  ds.write(0xBE);         // Read Scratchpad
  
  for (byte i = 0; i < 9; i++) {           // we need 9 bytes
    data[i] = ds.read();
  }
  if (OneWire::crc8(data, 8) != data[8]) {
#if DEBUG
    Serial.print(F("OWT: Bad CRC   for "));
    os->printAddr(&Serial, addr);
    Serial.print("->");
    os->printAddr(&Serial, *(uint64_t *)data);
    Serial.println();
#endif
    return INT16_MIN;
  }

  // handle missing sensor and double-check data
  if ((data[0] == 0 && data[1] == 0) || data[5] != 0xFF || data[7] != 0x10) {
#if DEBUG
    Serial.print(F("OWT: Bad data for "));
    os->printAddr(&Serial, addr);
    Serial.print("->");
    os->printAddr(&Serial, *(uint64_t *)data);
    Serial.println();
#endif
    return INT16_MIN;
  }

#if DEBUG
  Serial.print(F("OWT: Good data for "));
  os->printAddr(&Serial, addr);
  Serial.print("->");
  os->printAddr(&Serial, *(uint64_t *)data);
  Serial.println();
#endif
  
  // mask out bits according to precision of conversion
  int16_t raw = ((uint16_t)data[1] << 8) | data[0];
  int16_t t_mask[4] = {0x7, 0x3, 0x1, 0x0};
  byte cfg = (data[4] & 0x60) >> 5;
  raw &= ~t_mask[cfg];
  return raw;
}
#endif

