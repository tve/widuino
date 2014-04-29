// Copyright (c) 2013-2014 by Thorsten von Eicken
//
// Simple temperature node with One-Wire DS18B20 sensor on a port

#include <Widuino.h>
#include <OwScan.h>
#include <OwTemp.h>
#include <avr/wdt.h>

#define OW_PORT      2
#define MAX_DEV	     4
#define MAX_TEMP     4
#define TEMP_PERIOD 10      // how frequently to read sensors (in seconds)

MilliTimer tempTimer;
OwScan owScan(OW_PORT+3, MAX_DEV);
OwTemp owTemp(&owScan, MAX_TEMP);
byte numTemp;

// Standard module config and dispatch set-up
Net net(29); // use node_id=29 by default, which is goign to raise red flags...
Log l, *logger=&l;
static Configured *(node_config[]) = {
  &net, logger, &owScan, 0
};

//===== setup & loop =====

void setup() {
  delay(100);
  Serial.begin(57600);
  delay(100);
  Serial.println(F("***** SETUP: " __FILE__));
  config_init(node_config);

  numTemp = owScan.scan(logger);
  tempTimer.set(TEMP_PERIOD);
  logger->println(F("***** RUNNING: " __FILE__));
  wdt_enable(WDTO_2S);
}

int times = 0;

void loop() {
  wdt_reset();
  if (net.poll()) config_dispatch();

  if (owTemp.loop(TEMP_PERIOD)) {
    // prep packet with temp values
    byte data[MAX_TEMP];
    byte *d = data;
    
    for (byte i=0; i<numTemp; i++) {
      float t = owTemp.get(i);
      *d++ = (int8_t)(t+0.5);
      logger->print("Temp ");
      logger->print(i);
      logger->print("=");
      logger->println(t);
    }

    // send a packet with the data
    byte *pkt;
    while ((pkt = net.alloc()) == 0)
      if (net.poll())
        config_dispatch();
    pkt[0] = OWTEMP_MODULE;
    memcpy(pkt+1, data, d-data);
    net.send(1+d-data, true);
    delay(100);

    if (++times == 3) jb_upgrade();
  }

}
