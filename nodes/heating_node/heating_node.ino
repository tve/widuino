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

#define MAX_TEMP      20
#define TEMP_PERIOD   10    // how frequently to read sensors (in seconds)
#define MAX_RELAY      5
#define RELAY_PERIOD 120    // how frequently to read relays (in seconds)


Net net(29);  // use node_id=29 by default, which is going to raise red flags...
NTPTime ntptime;
Logger l, *logger=&l;
static Configured *(node_config[]) = {
  &net, logger, &ntptime, 0
};

MilliTimer notSet, toggle, debugTimer;

// Relays
bool relayOut[MAX_RELAY];
char relay_name[MAX_RELAY][8] = {
  "guest", "living", "bedroom", "polaris",
};
uint64_t relay_addr[MAX_RELAY] = {
  0x0564013200000054, 0x05ABE8310000004E, 0x050FE53100000022, 0x056EFA31000000C5
};
OwRelay owRelay(OWR_PORT+3, relay_addr, MAX_RELAY);

// Temperature names
char temp_name[MAX_TEMP][10] = {
  "panel", "htr in", "bdrm top", "livg top",
  "guest bot", "fauc cold", "fauc hot", "livg bot",
  "cold in", "fauc ret", "fauc out", "fauc c in",
  "guest top", "bdrm top",
};
// DS18B20 8E9331010000 : living zone top
// DS18B20 DE9731010000 : guest zone bot
// DS18B20 459931010000 : living zone bot
// DS18B20 1D9331010000 : water heater out  ! Missing
// DS18B20 179E31010000 : guest zone top
// DS18B20 9C9C31010000 : water heater in
// DS18B20 7AA131010000 : bedrm zone top
// DS18B20 9F8F31010000 : bedrm zone bot
// DS18B20 D09131010000 : wood surface
// DS18B20 218D31010000 : faucet mix cold
// DS18B20 A99931010000 : faucet mix hot
// DS18B20 D58731010000 : cold in
// DS18B20 5BA231010000 : faucet return
// DS18B20 5B9331010000 : faucet mix out
// DS18B20 FBA431010000 : faucet cold in
// DS18B20 7F8931010000 : water heater internal  ! Missing
uint64_t temp_addr[MAX_TEMP] = {
  0x28D09131010000CC, 0x289C9C3101000028, 0x287AA131010000DB, 0x288E933101000072,
  0x28DE973101000043, 0x28218D31010000D4, 0x28A99931010000FC, 0x28459931010000A8,
  0x28D58731010000C7, 0x285BA23101000014, 0x285B93310100005D, 0x28FBA431010000D4,
  0x28179E31010000B9, 0x289F8F3101000043,
};
OwTemp owTemp(OWT_PORT+3, temp_addr, MAX_TEMP);

void dumpMem(uint8_t *start) {
  for (uint8_t *a=start; a<start+0x200; a+=16) {
    logger->print((uint16_t)a, HEX);
    logger->print(": ");
    logger->print(((uint32_t*)a)[0], HEX);
    logger->print(' ');
    logger->print(((uint32_t*)a)[1], HEX);
    logger->print(' ');
    logger->print(((uint32_t*)a)[2], HEX);
    logger->print(' ');
    logger->print(((uint32_t*)a)[3], HEX);
    logger->print(' ');
    for (uint8_t i=0; i<32; i++) {
      if (a[i] >= 0x20 && a[i] < 0x7f) {
        logger->print((char)a[i]);
      } else {
        logger->print('.');
      }
    }
    logger->println();
  }
}

//===== setup & loop =====

void setup() {
  wdt_enable(WDTO_8S);
  jb_force();
  Serial.begin(57600);
  Serial.println(F("***** SETUP: " __FILE__));

  //eeprom_write_word((uint16_t *)0x20, 0xF00D);

  eeconf_init(node_config);
  logger->println(F("***** SETUP: " __FILE__));

  logger->println(F("Scanning temps"));
  ow_scan(OWT_PORT+3, temp_addr, MAX_TEMP, logger);
  wdt_reset();

  logger->println(F("Scanning relays"));
  ow_scan(OWR_PORT+3, relay_addr, MAX_RELAY, logger);
  wdt_reset();

  debugTimer.set(60000);

  logger->print("Free RAM: "); logger->println(jb_free_ram());
  logger->println(F("***** RUNNING: " __FILE__));
}

int cnt = 0;
void loop() {
  wdt_reset();
  if (net.flush()) eeconf_dispatch();

  if (debugTimer.poll()) {
      logger->println(F("Resetting..."));
      net.flush();
      delay(100);
      jb_upgrade(true);
  }

  if (owTemp.loop(TEMP_PERIOD)) {
    logger->print("Free RAM: "); logger->println(jb_free_ram());
    logger->print("Time: ");
    logger->print(hour());
    logger->print(':');
    logger->print(minute());
    logger->println();

    logger->println("Temp sensors: ");
    for (byte i=0; i<MAX_TEMP; i++) {
      if (!owTemp.isTemp(i)) continue;
      if (i < 10) logger->print(' ');
      logger->print(i);
      logger->print('=');
      float t = owTemp.get(i);
      if (t < 100.0) logger->print(' ');
      logger->print(t);
      logger->print("F ");
      logger->println(temp_name[i]);
    }
    //wdt_reset();
    //if (cnt % 10 == 0) owTemp.printDebug(logger);

    logger->println("Relays: ");
    for (byte i=0; i<MAX_RELAY; i++) {
      if (!owRelay.isRelay(i)) continue;
      logger->print(i);
      logger->print('=');
      logger->print(owRelay.get(i));
      logger->print(' ');
      logger->println(relay_name[i]);
    }

    if (++cnt >= 20) {
      logger->println(F("Resetting..."));
      delay(500);
      jb_upgrade(true);
    }
  }

  if (owRelay.loop(RELAY_PERIOD)) {
  }

  // If we don't know the time of day complain about it
  if (timeStatus() != timeSet) {
    if (notSet.poll(60000)) {
      logger->println("Time not set");
    }
  }

/*
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
  */

}
