// Copyright (c) 2013 Thorsten von Eicken
//
// Time class, receives time messages and keeps local time up-to-date

#ifndef NTPTIME_H
#define NTPTIME_H

#include <Time.h>

// Assumes JeeLib.h is included for rf12 constants

// handler for NTPTime messages received. The handler uses the timestamp in the message to set the
// local time clock (Time library). Use the Time library to read the current time.
void handleNTPTime(volatile uint8_t *pkt, uint8_t len);

// Pretty-print the time as YYYY/MM/DD HH:MM:SS UTC
void printNTPTime(Print *p, time_t t=0);

#endif // NTPTIME_H
