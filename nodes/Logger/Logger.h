// Copyright (c) 2013-2014 Thorsten von Eicken
//
// Logging class, the output can be directed to the serial port, an LCD display,
// and/or the network.

#ifndef LOGGER_H
#define LOGGER_H

#include <EEConf.h>

// Assumes JeeLib.h is included for rf12 constants
#define LOG_MAX (RF12_MAXDATA-1)    // max amount of chars that can be logged in one packet

class Logger : public Print, public Configured {
public:
  // Configuration structure stored in EEPROM
  typedef struct {
    bool  serial:1;  // log to serial port
    bool  lcd:1;     // log to LCD
    bool  rf12:1;    // log to the rf12 network
  } log_config;

private:
  log_config config, defaults;
  uint8_t buffer[LOG_MAX+1]; // +1 for null byte string termination
  uint8_t ix;

  void send(void);   // send accumulated buffer
  void init();

public:
  // constructor, uses default log initialization (serial + rf12b)
  Log(void);

  // constructor, explicit default log initialization (gets overridden by EEPROM)
  Log(log_config defaults);

  // write a character to the buffer, used by Print but can also be called explicitly
  // automatically prints/sends the buffer when it's full or a \n is written
  virtual size_t write (uint8_t v);

  // Configuration methods
  virtual void applyConfig(uint8_t *);
  virtual void receive(volatile uint8_t *pkt, uint8_t len);
};

extern Logger *logger;

#endif // LOGGER_H
