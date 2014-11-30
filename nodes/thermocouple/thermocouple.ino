// Copyright (c) 2013 by Thorsten von Eicken
//
// Node to display a thermocouple amplified via a AN623 instrumentation amp on
// the GLCD display in real-time

#define USE_GLCD

#include <Widuino.h>

#define BLINK_PIN     9 // PIN9=PB1=built-in LED
#define TH_PIN	     15 // PIN15=PORT2 A

#ifdef USE_GLCD
# include <GLCD_ST7565.h>
# include "utility/font_clR6x8.h"
  GLCD_ST7565 glcd;
  extern byte gLCDBuf[1024]; // requires removing "static" in GLCD_ST7565.cpp
#endif

Net net(10,0xD4);
//NetTime nettime;
Logger l((Logger::log_config){1, 0, 0}), *logger=&l;
MilliTimer notSet, lcdUpdate, xmit;
MilliTimer debugTimer;

//===== setup & loop =====

static Configured *(node_config[]) = {
  &net, logger, 0
};

void setup() {
  delay(100);
  Serial.begin(57600);
  delay(100);
  Serial.println(F("***** SETUP: " __FILE__));
  eeconf_reset(); // reset EEPROM
  eeconf_init(node_config);

  // Init analog pin
  pinMode(TH_PIN, INPUT);
  analogReference(DEFAULT);

# ifdef USE_GLCD
  glcd.begin(0x1a);
  glcd.clear();
  glcd.backLight(255);
  glcd.setFont(font_clR6x8);

  // draw a string at a location, use _p variant to reduce RAM use
  glcd.drawString_P(0,  0, PSTR(__FILE__));
  glcd.refresh();
# endif
  
  notSet.set(1000);
  logger->println(F("***** RUNNING: " __FILE__));
}

uint8_t avgRcvRssi, avgAckRssi = 0;

void loop() {
  while (net.flush()) eeconf_dispatch();

#if 0
  // If we don't know the time of day, just sit there and wait for it to be set
  if (timeStatus() != timeSet) {
    if (notSet.poll(60000)) {
#ifdef USE_LCD
      lcd.clear();
      lcd.setCursor(0,0); // first line
      lcd.print("Time not set");
#endif
#ifdef USE_GLCD
      glcd.clear();
      glcd.drawString_P(0, 0, PSTR("Time not set"));
      glcd.refresh();
#endif
    }
    return;
  }
#endif

  // Update the time display on the LCD
# ifdef USE_GLCD
  if (lcdUpdate.poll(1000)) {
    char buf[20];
    
    // read the analog thermocouple voltage and average
    delay(10);
    uint16_t meas[8];
    uint16_t analog = 0;
    for (byte i=0; i<8; i++) {
      analog += meas[i] = analogRead(TH_PIN);
      delay(1);
    }
    for (byte i=0; i<8; i++) {
      logger->print(meas[i]);
      logger->print(" ");
    }
    logger->println();

    analog = (analog+4)/8; // average with rounding
    float rmVolt = (float)analog * (3300/1024);
    float mVolt = rmVolt - 300;
    float temp = -0.01897 + 25.41881*mVolt - 0.42456*mVolt*mVolt + 0.04365*mVolt*mVolt*mVolt;
    int16_t mVolt_i = (int16_t)(mVolt+0.5);
    int16_t temp_i = (int16_t)(temp+0.5);

    // clear the upper part of the LCD where the text goes
    glcd.fillRect(0, 0, LCDWIDTH, 16, 0);

    // First line has volts and date/time
#if 0
    time_t t = now();
    char div = second(t) & 1 ? ':' : '-';
    snprintf(buf, 20, "%03d %2d/%2d %2d%c%02d",
        avgRcvRssi, month(t), day(t), hour(t), div, minute(t));
#else
    snprintf(buf, 20, "%04d %04d %04d %03d", analog, (int16_t)rmVolt, mVolt_i, temp_i);
#endif
    logger->println(buf);
    glcd.drawString(0, 0, buf);

#if 0
    // Second line has rcv and snd RSSI
    snprintf(buf, 20, "rcv:%03d snd:%03d", net.lastRcvRssi, net.lastAckRssi);
    glcd.drawString(0, 8, buf);
    net.lastRcvRssi = 0;
    net.lastAckRssi = 0;
#endif

    // Shift plot left one pixel and draw new RSSI as vertical line
    glcd.setUpdateArea(0,16,LCDWIDTH-1,LCDHEIGHT-1, false);
    for (byte l=2; l<8; l++) { // vertical blocks of 8 pix: do pix 16 thru 64
      byte *p = gLCDBuf + (l * 128);
      const byte x = 1; // shift by one pixel
      for (byte b = 0; b < LCDWIDTH-x; ++b)
        *(p+b) = *(p+b+x);
      for (byte b = LCDWIDTH-x; b < LCDWIDTH; ++b)
        *(p+b) = 0;
    }

    // Draw latest temp in right-most column
    if (temp_i > 0) {
      int16_t pix = temp_i>>1;
      glcd.drawLine(LCDWIDTH-1, LCDHEIGHT-1-pix, LCDWIDTH-1, LCDHEIGHT-1, 1);
    }

    glcd.refresh();
  }
# endif

}
