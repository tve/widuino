// Copyright (c) 2013-2014 Thorsten von Eicken
//
// Logging class, the output can be directed to the serial port, an LCD display,
// and/or the network.

#ifndef LOGGER_H
#define LOGGER_H

// Assumes JeeLib.h is included for rf12 constants
#define LOG_MAX (RF12_MAXDATA-1)    // max amount of chars that can be logged in one packet

#define LOG_SERIAL 1
#define LOG_RF12   2

class Logger : public Print {
public:
  bool  serial:1;  // log to serial port
  bool  lcd:1;     // log to LCD
  bool  rf12:1;    // log to the rf12 network

private:
  uint8_t dest;              // log desitnations (bitmask)
  uint8_t buffer[LOG_MAX+1]; // +1 for null byte string termination
  uint8_t ix;
  uint8_t missed;            // number of log messages that couldn't be sent

  void send(void);   // send accumulated buffer
  void init();
  uint8_t* allocPkt(void);

public:
  // constructor, use LOG_XXX flags
  Logger(uint8_t destinations);

  // write a character to the buffer, used by Print but can also be called explicitly
  // automatically prints/sends the buffer when it's full or a \n is written
  virtual size_t write (uint8_t v);

};

extern Logger *logger;

#endif // LOGGER_H
