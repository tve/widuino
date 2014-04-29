// Copyright (c) 2013-2014 Thorsten von Eicken
//
// Logging class
//

#include <JeeLib.h>
#include <Config.h>
#include <Net.h>
#include <Logger.h>

#define LCD    0    // support logging to the LCD (set to 0 to exclude that code)

Logger::Logger(void) {
  init();
#ifdef LOG_NORF12B
  this->defaults = (log_config){1, 0, 0, 0, 0};  // serial only
#else
  this->defaults = (log_config){1, 0, 1, 0, 0};  // serial and rf12b
#endif
}

Logger::Logger(log_config defaults) {
  init();
  this->defaults = defaults;
}

void Logger::init() {
  ix = 0;
  moduleId = LOG_MODULE;
  configSize = sizeof(log_config);
  memset(&config, 0, sizeof(log_config));
  config.serial = true;
}

void Logger::send(void) {
  buffer[ix] = 0;

  // Log to the serial port
  if (config.serial) {
    //Serial.print("LOG:");
    Serial.print((char *)buffer);
    if (ix > 0 && buffer[ix-1] == '\n')
      Serial.print('\r');
  }

#ifndef LOG_NORF12B
  // Log to the network
  if (config.rf12) {
    uint8_t *pkt = net.alloc();
    //while (!pkt) { (void)net.poll(); pkt = net.alloc(); }
    if (pkt) {
      *pkt = LOG_MODULE;
      memcpy(pkt+1, buffer, ix);
      net.send(ix+1, true); // +1 for module_id byte
    } else {
      //Serial.println(F("Log: out of rf12 buffers"));
    }
  }
#endif

#if LCD
  // Log to the LCD
  if (config.lcd) {
  }
#endif

  ix = 0;
}

// write a character to the buffer, used by Print but can also be called explicitly
// automatically sends the buffer when it's full or a \n is written
size_t Logger::write (uint8_t v) {
  if (ix >= LOG_MAX) { send(); }
  buffer[ix++] = v;
  if (v == 012) send();
  return 1;
}

// ===== Configuration =====

void Logger::receive(volatile uint8_t *pkt, uint8_t len) { return; } // this is never called :-)

void Logger::applyConfig(uint8_t *cf) {
  if (cf) {
    memcpy(&config, cf, sizeof(log_config));
    //Serial.print(F("Config Log: 0x"));
    //Serial.println(*cf, HEX);
  } else {
    memset(&config, 0, sizeof(log_config));
    config = defaults;
    config_write(LOG_MODULE, &config);
  }
  Serial.print(F("Config Log:"));
  if (config.serial) Serial.print(F(" serial"));
  if (config.lcd) Serial.print(F(" lcd"));
  if (config.rf12) Serial.print(F(" rf12"));
  if (*(uint8_t*)&config == 0) Serial.print(F(" !NONE! "));
  Serial.println();
}

