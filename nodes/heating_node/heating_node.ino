// Copyright (c) 2013 by Thorsten von Eicken
//
// Controller for sub-floor radint heating system: monitors temperature sensors and 
// controls zone circulation pumps. All this via 1-wire buses.

#include <Widuino.h>
#include <NTPTime.h>
#include <Time.h>
#include <OwScan.h>
#include <OwTemp.h>
#include <OwRelay.h>
#include <avr/wdt.h>

#define OWT_PORT       1    // Port for temperature sensors
#define OWR_PORT       2    // Port for relay switches

#define MAX_TEMP      16
#define TEMP_PERIOD   10    // how frequently to read sensors (in seconds)
#define MAX_RELAY      5
#define RELAY_PERIOD 120    // how frequently to read relays (in seconds)

OwScan owScanT(OWT_PORT+3, MAX_TEMP);
OwTemp owTemp(&owScanT, MAX_TEMP);
OwScan owScanR(OWR_PORT+3, MAX_RELAY);
OwRelay owRelay(&owScanR, MAX_RELAY);

Net net(29);  // use node_id=29 by default, which is going to raise red flags...
NTPTime ntptime;
Logger l, *logger=&l;
static Configured *(node_config[]) = {
  &net, logger, &ntptime, &owScanT, 0
};

// Relays
byte numRelay;
bool relayOut[MAX_RELAY];
MilliTimer notSet, toggle, debugTimer;

// Temperature names
byte numTemp;
char temp_name[MAX_TEMP][5] = { "Air ", "?" };

//===== setup & loop =====

void setup() {
  Serial.begin(57600);
  Serial.println(F("***** SETUP: " __FILE__));

  //eeprom_write_word((uint16_t *)0x20, 0xF00D);

  eeconf_init(node_config);

  numTemp = owScanT.scan(logger);
  numRelay = owScanR.scan(logger);
  //owRelay.swap(0, 1);
  logger->println(F("***** RUNNING: " __FILE__));
  wdt_enable(WDTO_4S);
}

int cnt = 0;
void loop() {
  wdt_reset();
  if (net.flush()) eeconf_dispatch();

  owTemp.loop(TEMP_PERIOD);
  owRelay.loop(RELAY_PERIOD);

  // Debug printout
  if (debugTimer.poll(7770)) {
    //logger->println("OwTimer debug...");
    //owTemp.printDebug((Print*)logger);
    logger->print("Temp: ");
    logger->print(owTemp.get(0));
    logger->print("F [");
    logger->print(owTemp.getMin(0));
    logger->print("..");
    logger->print(owTemp.getMax(0));
    logger->println("]");
  }

  // If we don't know the time of day complain about it
  if (timeStatus() != timeSet) {
    if (notSet.poll(60000)) {
      logger->println("Time not set");
    }
  }

  // Toggle both relays
  if (0 && toggle.poll(1000)) {
    cnt += 1;
    owRelay.loop(0);
    relayOut[0] = cnt & 1;
    logger->print("Relay 0 is ");
    logger->print(owRelay.get(0));
    logger->print(" setting to ");
    logger->print(relayOut[0]);
    if (owRelay.set(0, relayOut[0])) {
      logger->println(" OK");
    } else {
      logger->println(" FAILED");
    }

    relayOut[1] = (cnt>>1)&1;
    logger->print("Relay 1 is ");
    logger->print(owRelay.get(1));
    logger->print(" setting to ");
    logger->print(relayOut[1]);
    if (owRelay.set(1, relayOut[1])) {
      logger->println(" OK");
    } else {
      logger->println(" FAILED");
    }
  }

}
