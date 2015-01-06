// Copyright (c) 2013 Thorsten von Eicken
//
// Network Time class

#include <JeeLib.h>
#include <Time.h>
#include <Logger.h>
#include <Modules.h>
#include <NTPTime.h>

void handleNTPTime(volatile uint8_t *pkt, uint8_t len) {
  if (pkt[0] != NTPTIME_MODULE) return;
  if (len >= 4) {
    bool wasSet = timeStatus();
    setTime(*(uint32_t *)(pkt+1));
    if (!wasSet) logger->println(F("NTP time initialized"));
  }
}

static void print2(Print *p, uint8_t v, char fill='0') {
		if (v < 10) p->print(fill);
		p->print(v);
}

void printNTPTime(Print *p, time_t t=0) {
  if (!timeStatus()) {
    p->print(F("????/??/?? ??:?? UTC"));
	return;
  }
  if (t == 0) t = now();
  p->print(year(t));     p->print('/');
  print2(p, month(t));   p->print('/');
  print2(p, day(t));     p->print(' ');
  print2(p, hour(t));    p->print(':');
  print2(p, minute(t));  p->print(':');
  print2(p, second());
  p->print(" UTC");
}
