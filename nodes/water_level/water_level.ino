// Copyright (c) 2014 by Thorsten von Eicken
//
// Water level sensor based on a Honeywell 24PC differential piezo pressure sensor
//
// The sensor used in a 24PCBFA6D sensor: differential up to 5psi whatstone bridge with
// a span of 100mV @5psi. The span is amplified using a AN623 instrumentation amplifier
// with a gain of ~300x for depths of ~36 inches and ~100x for depths of ~8 ft.

#include <Widuino.h>
#include <NTPTime.h>
#include <Time.h>
#include <OwScan.h>
#include <OwTemp.h>
#include <avr/wdt.h>

#define MAX_TEMP     4
#define PERIOD      10      // how frequently to read sensors (in seconds)
// #define PERIOD     60      // how frequently to read sensors (in seconds)

#define STATUS_LED   9      // B1
#define WATER1      16      // port3 A
#define WATER2      17      // port4 A
#define WATER_LED    6      // port3 D
#define WATER_OW     7      // port4 D
#define POWER_RELAY  4      // port1 D

// Temperature sensor names and addresses
char temp_name[MAX_TEMP][6] = { "upper", "lower" };
// DS18B20 8E9331010000 : living zone top
// DS18B20 DE9731010000 : guest zone bot
uint64_t temp_addr[MAX_TEMP] = {
};
OwTemp owTemp(WATER_OW, temp_addr, MAX_TEMP);

// Standard module config and dispatch set-up
Net net; // use node_id=29 by default, which is going to raise red flags...
Logger l(LOG_RF12), *logger=&l;

//===== setup & loop =====

void setup() {
  wdt_enable(WDTO_8S);
  jb_force();
  Serial.begin(57600);
  net.init(29);  // use node_id=29 by default, which is going to raise red flags...
  logger->println(F("***** SETUP: " __FILE__));

  // turn LEDs on for half a second
  digitalWrite(STATUS_LED, HIGH);
  digitalWrite(WATER_LED, HIGH);
  pinMode(STATUS_LED, OUTPUT);
  pinMode(WATER_LED, OUTPUT);
  delay(500);
  digitalWrite(STATUS_LED, LOW);
  digitalWrite(WATER_LED, LOW);

  // configure analog water pressure sensor ADCs
  pinMode(WATER1, INPUT);
  pinMode(WATER2, INPUT);
  analogReference(DEFAULT);

  // turn off pump relay
  digitalWrite(POWER_RELAY, LOW);
  pinMode(POWER_RELAY, OUTPUT);

  logger->println(F("Scanning temps"));
  ow_scan(WATER_OW, temp_addr, MAX_TEMP, logger);

  logger->print("Free RAM: "); logger->println(jb_free_ram());
  logger->println(F("***** RUNNING: " __FILE__));
}

int times = 0;

int readWater(uint8_t analogPin, uint8_t ledPin) {
  uint8_t n = 8;
  int raw = 0;
  while (n-- > 0) raw += analogRead(analogPin);
  raw = (raw + 4) >> 3;
  logger->print("Water #");
  logger->print(analogPin);
  logger->print(" = ");
  logger->print(raw);
  logger->print(" = ");
  logger->print((float)raw * 3.3 / 1024);
  logger->println("V");

  // set the water level led if the level is above a few inches
  if (raw > 300) digitalWrite(ledPin, HIGH);

  return raw;
}

void dispatch() {
  handleNTPTime(rf12_data, rf12_len);
}

void loop() {
  wdt_reset();
  while (net.flush()) dispatch();

  if (owTemp.loop(PERIOD)) {
    digitalWrite(STATUS_LED, HIGH);
    // prep packet with temp values
    byte data[MAX_TEMP];
    byte *d = data;

    for (byte i=0; i<MAX_TEMP; i++) {
      if (!owTemp.isTemp(i)) continue;
      float t = owTemp.get(i);
      *d++ = (int8_t)(t+0.5);
      logger->print("Temp ");
      logger->print(i);
      logger->print("=");
      logger->println(t);
    }

    // send a packet with the temperature data
    byte *pkt;
    while ((pkt = net.alloc()) == 0) {
      if (net.poll()) dispatch();
    }
    pkt[0] = OWTEMP_MODULE;
    memcpy(pkt+1, data, d-data);
    net.send(1+d-data, true);

    // read the water level sensors
    digitalWrite(WATER_LED, LOW);
    int raw1 = readWater(WATER1, WATER_LED);
    int raw2 = readWater(WATER2, WATER_LED);

    // send a packet with the water level data
    while ((pkt = net.alloc()) == 0) {
      if (net.poll()) dispatch();
    }
    pkt[0] = WATERLEVEL_MODULE;
    memcpy(pkt+1, &raw1, sizeof(raw1));
    memcpy(pkt+1+sizeof(raw1), &raw2, sizeof(raw2));
    net.send(1+2*sizeof(raw1), true);

    digitalWrite(STATUS_LED, LOW);

    logger->print("Time: ");
    printNTPTime(logger);
    logger->println();

    times++;
    if (times > 4) {
      logger->println(F("Resetting..."));
      net.flush();
      delay(100);
      jb_upgrade(true);
    }

    //digitalWrite(POWER_RELAY, (times++)&1);
  }
}
