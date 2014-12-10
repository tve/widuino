// Copyright (c) 2013 Thorsten von Eicken
//
// Time class, receives time messages and keeps local time up-to-date

#ifndef NTPTIME_H
#define NTPTIME_H

// Assumes JeeLib.h is included for rf12 constants

class NTPTime : public Configured {

  // Configuration structure stored in EEPROM
  typedef struct {
    int8_t	offset;	  // time zone offset
  } ntptime_config;

  int8_t offset;

public:
	// constructor
	NTPTime(void);

  // Configuration methods
	virtual void applyConfig(uint8_t *);
	virtual void receive(volatile uint8_t *pkt, uint8_t len);
};

#endif // NTPTIME_H
