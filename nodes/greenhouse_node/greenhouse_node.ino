// Copyright (c) 2013 by Thorsten von Eicken
// Greenhouse Node
//   4x20 LCD display, up/down/select buttons
//   heater and fan relays
//   inlet shutter servo
//   1-wire temperature input
//   buzzer 
//

#include <Widuino.h>
#include <PortsLCD.h>
#include <NTPTime.h>
#include <Time.h>
#include <OwScan.h>
#include <OwTemp.h>
#include <SlowServo.h>
#include <avr/wdt.h>

#define HEATER_PORT 1  // A pin
#define ONEW_PORT   1  // D pin
#define FAN_PORT    2  // A pin
#define SERVO_PORT  2  // D pin
#define BUTN_PORT   3
#define DISP_PORT   4

#define SERVO_OPEN   0
#define SERVO_CLOSED 180

#define N_TEMP      5  // number of temperature sensors installed

#define ENABLE_SERVO 1

// Define LCD here 'cause we use it soon
PortI2C myI2C (DISP_PORT);
LiquidCrystalI2C lcd (myI2C);

OwScan owScan(ONEW_PORT+3); // digital pin

Net net(29);  // use node_id=29 by default, which is going to raise red flags...
NTPTime ntptime;
Logger l, *logger=&l;
static Configured *(node_config[]) = {
  &net, logger, &ntptime, &owScan, 0
};

// Set points
int limit_heat = 60;  // when to turn heat on in F
int limit_fan  = 80;  // when to turn fan on in F
// Testing override
float force_air = NAN;      // used to force air temp for testing

//===== Heater relay =====

// The heater relay is driven using an open-collector style output where low=ON and
// tri-state=OFF. The heater plug drives the relay using the analog pin.

int8_t heaterAuto = 0;   // automatic says: 0=off, 1=on
int8_t heaterForce = 0;  // 0=auto, 1=OFF, 2=ON
Port heaterRelay(HEATER_PORT);   // port number, analog pin

void initHeater() {
  heaterRelay.mode2(INPUT);    // tri-state=OFF
  heaterRelay.digiWrite2(LOW); // this never changes
  heaterRelay.mode2(OUTPUT);   // turn on-off for 0.3 seconds
  delay(300);
  heaterRelay.mode2(INPUT);
}

MilliTimer heaterPrint;

void setHeater() {
  bool on = heaterForce == 2 || (!heaterForce && heaterAuto);
  heaterRelay.mode2(on ? OUTPUT : INPUT);
  //if (heaterPrint.poll(60000)) {
  //  Serial.print("Heater is ");
  //  Serial.println(on ? "ON" : "OFF");
  //}
}

//===== Fan relay =====

// The fan relay is driven using an open-collector style output where low=ON and
// tri-state=OFF. The fan plug drives the relay using the analog pin.

int8_t fanAuto = 0;   // automatic says: 0=off, 1=on
int8_t fanForce = 0;  // 0=auto, 1=OFF, 2=ON
Port fanRelay(FAN_PORT);   // port number, analog pin

void initFan() {
  fanRelay.mode2(INPUT);    // tri-state=OFF
  fanRelay.digiWrite2(LOW); // this never changes
  fanRelay.mode2(OUTPUT);   // turn on-off for 0.3 seconds
  delay(300);
  fanRelay.mode2(INPUT);
}

MilliTimer fanPrint;

void setFan() {
  boolean on = fanForce == 2 || (!fanForce && fanAuto);
  fanRelay.mode2(on ? OUTPUT : INPUT);
  //if (fanPrint.poll(60000)) {
  //  Serial.print("Fan is ");
  //  Serial.println(on ? "ON" : "OFF");
  //}
}

//===== Servo =====

int8_t servoAuto = 0;    // automatic says: 0=off, 1=on

SlowServo servo(SERVO_PORT+3, 450, 1800);

void initServo() {
#if ENABLE_SERVO
  servo.attach(1500); // need to save last position!
#else
  Serial.println("Servo disabled");
#endif
}

void setServo() {
  // the servo is coupled to the fanForce
  boolean on = fanForce == 2 || (!fanForce && servoAuto);
#if ENABLE_SERVO
  servo.write(on ? SERVO_OPEN : SERVO_CLOSED);
#endif
}

//===== Temperature Sensors =====

// Polls and keeps track of a number of one-wire temperature sensors. Reads the sensors
// every few seconds and keeps track of daily min/max by having an array of the min/max
// for every hour and shifting that. So at any point in time it has the min/max for the
// past 24 hours with a 1-hour granularity.

#define MAX_TEMP (N_TEMP+1) // max number of temperature sensors supported
OwTemp owt(&owScan, MAX_TEMP);
byte temp_num = 0;          // number of temp sensors

#define TEMP_PERIOD 20      // how frequently to read sensors (in seconds)
#define T_PLUG   0
#define T_AIR    1
#define T_SOIL   2
#define T_CEIL   3
#define T_IN     4

// Sensor 1-wire addresses
uint64_t temp_addr[MAX_TEMP] = {
  0xBD00000131802628LL, // plug
  0x5C00000131850128LL, // air -- orange wire
  0xF80000013180D728LL, // soil -- long green wire
  0xFA000001318E5A28LL, // ceiling -- blue wire
  0x19000001318A7728LL, // air inlet -- short green wire
  0x0000000000000028LL, // spare
};

// Temperature names
char temp_name[MAX_TEMP][5] = { "Plug", "Air ", "Soil", "Ceil", "Inlt", "?" };

// Temperatures
float  temp_now[MAX_TEMP];      // current temperatures
int8_t temp_min[MAX_TEMP][24];  // per-hour minimum temperatures for past 24 hours
int8_t temp_max[MAX_TEMP][24];  // per-hour maximum temperatures for past 24 hours

// Timer/counter to shift min/max temps every hour
MilliTimer minMaxTimer;  // 60 second timer
byte minMaxCount = 0;    // count 60 minutes

// Find temperature sensors and print what we found
void findTemp() {
  temp_num = owt.setup(N_TEMP, temp_addr);

  lcd.setCursor(0, 1);
  lcd.print("Found ");
  lcd.print(temp_num);
  lcd.print("/");
  lcd.print(N_TEMP);
  lcd.println(" temps ");

  Serial.print("Found ");
  Serial.print(temp_num);
  lcd.print(" of ");
  lcd.print(N_TEMP);
  Serial.println(" sensors");
}

// Update temperature sensors and keep track on min/max
// Keeps it own timer to know when to read temp sensors, force=true overrides
// that and forces a read
void pollTemp(boolean force=false) {
  if (owt.poll(force ? 0 : 20)) {
    Serial.print("Temperatures: ");
    for (int s=0; s<N_TEMP; s++) {
      Serial.print(temp_name[s]); Serial.print(":");
      float t = owt.get(temp_addr[s]);
      if (isnan(t)) t = owt.get(temp_addr[s]);
      if (!isnan(t)) {
        temp_now[s] = t;
        int8_t rt = (int8_t)(temp_now[s] + 0.5);
        if  (rt < temp_min[s][0]) temp_min[s][0] = rt;
        if  (rt > temp_max[s][0]) temp_max[s][0] = rt;
        Serial.print(temp_now[s]); Serial.print(" ");
      } else {
        Serial.print("NaN   ");
      }
    }
    Serial.println();

    // rotate min/max temp every hour
    if (minMaxTimer.poll(60000)) {
      minMaxCount++;
      if (minMaxCount == 60) {
        minMaxCount = 0;
        // rotate min/max temps
        for (int s=0; s<N_TEMP; s++) {
          for (int i=23; i>0; i--) {
            temp_min[s][i] = temp_min[s][i-1];
            temp_max[s][i] = temp_max[s][i-1];
          }
        }
      }
    }
  }
}

// Initialize temperature sensor "module"
void initTemp() {
  findTemp();
  pollTemp(true);

  // init min/max arrays
  for (int i=0; i<MAX_TEMP; i++) {
    for (int j=0; j<24; j++) {
      temp_min[i][j] = (int8_t)(temp_now[i]+0.5);
      temp_max[i][j] = (int8_t)(temp_now[i]+0.5);
    }
  }
}

//===== 16x2 LCD display =====

// 1234567890123456
// Soil:00F Heat:00
// 00F..00F Up/Down

MilliTimer lcdTimer;
byte lcd_pos = 0; // Air/Ceil/Soil/...
// menu
enum { M_HTR, M_FAN, M_ALARM, M_TEST };
int8_t menu_pos = M_HTR;
// sub-menu
enum { M_TOP, M_ONOFF, M_UPDN, M_LAST };
int8_t sub_pos = M_TOP;

#define screen_width 16
#define screen_height 2

// Custom characters, these are used to display temperatures above 99F using two characters
// Also, a raised&smaller F for Farenheit, and an up-down arrow
byte ch0[8] = { 0x0A, 0x1D, 0x0D, 0x0D, 0x0D, 0x0D, 0x1A, 0 }; // 10
byte ch1[8] = { 0x09, 0x1B, 0x09, 0x09, 0x09, 0x09, 0x09, 0 }; // 11
byte ch2[8] = { 0x0A, 0x1D, 0x09, 0x0A, 0x0C, 0x0C, 0x1F, 0 }; // 12
byte chF[8] = { 0x1E, 0x10, 0x1C, 0x10, 0x10, 0x00, 0x00, 0x00 }; // F
byte chU[8] = { 0x04, 0x0E, 0x1F, 0x04, 0x04, 0x1F, 0x0E, 0x04 }; // up/down

void initLCD() {
  lcd.begin(screen_width, screen_height);
  lcd.print(__FILE__);
  lcd.createChar(0, ch0);
  lcd.createChar(1, ch1);
  lcd.createChar(2, ch2);
  lcd.createChar(6, chF);
  lcd.createChar(7, chU);
  lcdTimer.set(10);
}

// Print temperature on LCD, use special characters for >99F to fit in 2 positions
void tempPrintLCD(const char *label, int8_t t) {
  if (label && strlen(label)) lcd.print(label);
  if (isnan(t)) {
    lcd.print("---");
  } else {
    int8_t t1 = t / 10;
    if (t1 >= 10) lcd.print((char)(t1-10));
    else          lcd.print(t1);
    lcd.print(t % 10);
    lcd.print((char)6);
  }
}

// calculate the minimum temperature
int8_t calcMin(int8_t t_min[24]) {
  int8_t t = t_min[0];
  for (byte i=1; i<24; i++)
    if (t_min[i] < t) t = t_min[i];
  return t;
}

// calculate the maximum temperature
int8_t calcMax(int8_t t_max[24]) {
  int8_t t = t_max[0];
  for (byte i=1; i<24; i++)
    if (t_max[i] > t) t = t_max[i];
  return t;
}

// helper to show the temperatures
void showTempRaw(const char *label, float temp, int8_t min, int8_t max) {
  lcd.setCursor(0, 0);
  tempPrintLCD(label, (int8_t)(temp+0.5));
  lcd.setCursor(0, 1);
  tempPrintLCD(0, min);
  tempPrintLCD("..", max);
  //lcd.print(">");
}

// Update the temperatures
void showTemp() {
  if (lcdTimer.poll(1600)) {
    // display temp
    char str[6];
    strcpy(str, temp_name[lcd_pos]);
    str[4] = ':';
    str[5] = 0;
    showTempRaw(str, temp_now[lcd_pos], calcMin(temp_min[lcd_pos]), calcMax(temp_max[lcd_pos]));
    lcd_pos++;
    if (lcd_pos == N_TEMP) lcd_pos = 0;
  }
}

const char *forceString[] = { "~~~", "OFF", " ON" };

// Update the menus
void showMenu() {
  lcd.setCursor(8, 0);
  lcd.print(" ");
  switch (menu_pos) {
  default:
    menu_pos = M_HTR;
    // fall-thru
  case M_HTR:
    lcd.print("Htr:");
    if (sub_pos == M_ONOFF) lcd.print(forceString[heaterForce]);
    else                    tempPrintLCD(0, limit_heat);
    break;
  case M_FAN:
    lcd.print("Fan:");
    if (sub_pos == M_ONOFF) lcd.print(forceString[fanForce]);
    else                    tempPrintLCD(0, limit_fan);
    break;
  case M_ALARM:
    lcd.print("Alarm: ");
    break;
  case -1:
    menu_pos = M_TEST;
    // fall-thru
  case M_TEST:
    lcd.print("Test:  ");
    break;
  }

  lcd.setCursor(8, 1);
  lcd.print(" ");
  switch (sub_pos) {
  default:
    sub_pos = M_TOP;
    // fall-thru
  case M_TOP:
    lcd.print("  \x07SEL\x07");
    break;
  case M_ONOFF:
    lcd.print(" On/Off");
    break;
  case M_UPDN:
    lcd.print("Up/Down");
    break;
  }
}

/*

//===== Beeper =====

#define BEEPER_PIN (16)
#define BEEPER_FREQ (2500)
MilliTimer beeperTimer;
byte beeperState = 0;

void startBeeper(byte num) {
  beeperState = 2*num-1;
  beeperTimer.set(800);
  tone(BEEPER_PIN, BEEPER_FREQ);
}

void loopBeeper() {
  if (beeperState == 0) return;
  // wait for timer before doing next state transition
  if (beeperTimer.poll()) {
    if (beeperState & 1) {
      // beeping - switch to pause (or be done)
      noTone(BEEPER_PIN);
      beeperState--;
      if (beeperState > 0) beeperTimer.set(200);
    } else {
      // not beeping - start beeping
      tone(BEEPER_PIN, BEEPER_FREQ);
      beeperState--;
      beeperTimer.set(800);
    }
  }
}
*/

//===== Buttons =====

MilliTimer debounce;
Port btn(BUTN_PORT);
byte lastState = 0;
byte checkFlags = 0;
enum { NOTHING, TOGGLEDN, TOGGLEUP, SELECT }; // for buttonCheck

byte buttonState() {
  int v = btn.anaRead();
  byte state = v < 300 ? 1 : (v > 700 ? 2 : 0);
  return state | (!btn.digiRead() << 2);
}

byte buttonCheck() {
  if (debounce.idle() || debounce.poll()) {
    byte newState = buttonState();
    if (newState != lastState) {
      debounce.set(100); // don't check again for at least 100 ms
      if ((lastState ^ newState) & 1 && newState & 1)
        bitSet(checkFlags, TOGGLEDN);
      if ((lastState ^ newState) & 2 && newState & 2)
        bitSet(checkFlags, TOGGLEUP);
      if ((lastState ^ newState) & 4 && newState & 4)
        bitSet(checkFlags, SELECT);
      lastState = newState;
    }
  }
  // note that simultaneous button events will be returned in successive calls
  if (checkFlags) {
    for (byte i = 1; i <= SELECT; ++i) {
      if (bitRead(checkFlags, i)) {
          bitClear(checkFlags, i);
        return i;
      }
    }
  }
  // if there are no button events, return the overall current button state
  return NOTHING;
}

//===== setup & loop =====

void setup() {
  initLCD();
  Serial.begin(57600);
  Serial.println(F("***** SETUP: " __FILE__));

  numTemp = owScanT.scan(logger);

  btn.mode(INPUT_PULLUP);
  delay(500);
  initTemp();
  initHeater();
  initFan();
  initServo();
  delay(1000);
  showMenu();

  wdt_enable(WDTO_4S);
}

MilliTimer bp;
MilliTimer tchk; // re-check temp sensors if not all found

void loop() {
  wdt_reset();
  byte b = buttonCheck();

  if (b == TOGGLEUP || b == TOGGLEDN) {
    int8_t dir = (b == TOGGLEUP ? 1 : -1);
    //Serial.print("Dir:");
    //Serial.println(dir);
    switch (sub_pos) {
    default:
      sub_pos = M_TOP;
    case M_TOP: // scroll
      menu_pos += -dir;
      break;
    case M_ONOFF: // on/off
      switch (menu_pos) {
      case M_HTR:
        heaterForce += dir;
        if (heaterForce > 2) heaterForce = 0;
        if (heaterForce < 0) heaterForce = 2;
        break;
      case M_FAN:
        fanForce += dir;
        if (fanForce > 2) fanForce = 0;
        if (fanForce < 0) fanForce = 2;
        break;
      case M_ALARM:
        break;
      case M_TEST:
        force_air = (dir == 1 ? 80.0 : NAN);
        pollTemp(true);
        Serial.print("TEST ON/OFF: ");
        Serial.println(force_air);
        break;
      }
      break;
    case M_UPDN: // up/down
      switch (menu_pos) {
      case M_HTR:
        limit_heat += dir*5;
        break;
      case M_FAN:
        limit_fan  += dir*5;
        break;
      case M_ALARM:
        break;
      case M_TEST:
        force_air += dir*5;
        pollTemp(true);
        Serial.print("TEST UP/DN: ");
        Serial.println(force_air);
        break;
      }
      break;
    }
    showMenu();
  } else if (b == SELECT) {
    sub_pos++;
    if (sub_pos >= M_LAST) sub_pos = 0;
    showMenu();
  }

  pollTemp();
  // override air temp for testing
  if (!isnan(force_air)) temp_now[T_AIR]  = force_air;
  showTemp();

  // send & receive rf12 packets
  //pollPacket();

  // Fan temp control with 2 degree hysteresis
  bool oldFanAuto = fanAuto;
  if (fanAuto)
    fanAuto    = !isnan(temp_now[T_AIR]) && temp_now[T_AIR] > limit_fan-1;
  else
    fanAuto    = !isnan(temp_now[T_AIR]) && temp_now[T_AIR] > limit_fan+1;
  if (oldFanAuto != fanAuto) Serial.println(fanAuto ? "Fan off->ON" : "Fan on->OFF");

  // Shutter servo work in conjunction with fan, but turns on 2 degrees lower
  bool oldServoAuto = servoAuto;
  if (fanAuto)
    servoAuto = true;
  else if (servoAuto)
    servoAuto  = !isnan(temp_now[T_AIR]) && temp_now[T_AIR] > limit_fan-3;
  else
    servoAuto  = !isnan(temp_now[T_AIR]) && temp_now[T_AIR] > limit_fan+1;
  if (oldServoAuto != servoAuto) Serial.println(servoAuto ? "Servo off->ON" : "Servo on->OFF");

  // Heater temp control with 2 degree hysteresis
  bool oldHeaterAuto = heaterAuto;
  if (heaterAuto)
    heaterAuto = !isnan(temp_now[T_AIR]) && temp_now[T_AIR] < limit_heat+1;
  else
    heaterAuto = !isnan(temp_now[T_AIR]) && temp_now[T_AIR] < limit_heat-1;
  if (oldHeaterAuto != heaterAuto) Serial.println(heaterAuto ? "Heater off->ON" : "Heater on->OFF");

  setHeater();
  setFan();
  setServo();
#if ENABLE_SERVO
  servo.loop();
#endif
  
}

