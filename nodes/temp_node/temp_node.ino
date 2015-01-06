// Copyright (c) 2013-2014 by Thorsten von Eicken
//
// Simple temperature node with One-Wire DS18B20 sensor on a port

#include <Widuino.h>
#include <NTPTime.h>'
#include <Time.h>
#include <OwScan.h>
#include <OwTemp.h>
#include <avr/wdt.h>

#define OW_PORT      2
#define MAX_DEV	     4
#define MAX_TEMP     4
#define TEMP_PERIOD  5      // how frequently to read sensors (in seconds)

MilliTimer tempTimer;
uint64_t temp_addr[MAX_TEMP];
OwTemp owTemp(OW_PORT+3, temp_addr, MAX_TEMP);
Port led(1);

// Standard module config and dispatch set-up
Net net(29); // use node_id=29 by default, which is going to raise red flags...
NTPTime ntptime;
Logger l, *logger=&l;
static Configured *(node_config[]) = {
  &net, logger, &ntptime, 0
};

extern void trace_flush(bool);

//===== setup & loop =====

void setup() {
  wdt_enable(WDTO_8S);
  jb_force();
  Serial.begin(57600);
  Serial.println(F("***** SETUP: " __FILE__));
  eeconf_init(node_config);
  logger->println(F("***** SETUP: " __FILE__));

  led.mode(OUTPUT);
  led.digiWrite(HIGH);
  led.mode2(OUTPUT);
  led.digiWrite2(HIGH);

  ow_scan(OW_PORT+3, temp_addr, MAX_TEMP, logger);
  tempTimer.set(TEMP_PERIOD);
  logger->println(F("***** RUNNING: " __FILE__));
}

int times = 0;

void loop() {
  while (net.flush()) {
    uint8_t m = rf12_data[0];
    eeconf_dispatch();
    logger->print("Recv: ");
    logger->println(m);
  }

  if (owTemp.loop(TEMP_PERIOD)) {
    logger->print("Time: ");
    logger->print(hour());
    logger->print(':');
    logger->print(minute());
    logger->println();
    // prep packet with temp values
    byte data[MAX_TEMP];
    byte *d = data;
    
    for (byte i=0; i<MAX_TEMP; i++) {
      if (!owTemp.isTemp(i)) continue;
      float t = owTemp.get(i);
      *d++ = (int8_t)(t+0.5);
      led.digiWrite(LOW);
      led.digiWrite2(LOW);
      logger->print("Temp ");
      logger->print(i);
      logger->print("=");
      logger->println(t);
      led.digiWrite2(HIGH);
    }

/*
    // send a packet with the data
    byte *pkt;
    while ((pkt = net.alloc()) == 0) {
      Serial.println("W..");
      if (net.poll())
        config_dispatch();
    }
    pkt[0] = OWTEMP_MODULE;
    memcpy(pkt+1, data, d-data);
    net.send(1+d-data, true);
*/

    if (++times == 5) {
      net.flush();
      delay(200);
      jb_upgrade(1);
    }
  }

}
