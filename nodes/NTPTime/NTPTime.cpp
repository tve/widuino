// Copyright (c) 2013 Thorsten von Eicken
//
// Network Time class

#include <JeeLib.h>
#include <EEConf.h>
#include <Net.h>
#include <Time.h>
#include <Logger.h>
#include <NTPTime.h>

// constructor
NTPTime::NTPTime(void) {
  offset = 0; // UTC default
  moduleId = NTPTIME_MODULE;
  configSize = sizeof(ntptime_config);
}

// ===== Configuration =====

// Receive a time packet with UTC time
void NTPTime::receive(volatile uint8_t *pkt, uint8_t len) {
  if (len >= 4) {
    bool wasSet = timeStatus();
    setTime(*(uint32_t *)pkt);
    if (!wasSet) logger->println(F("Time initialized"));
  }
}

void NTPTime::applyConfig(uint8_t *cf) {
  if (cf)
    offset = ((ntptime_config *)cf)->offset;
  else
    config_write(NTPTIME_MODULE, &offset);
	Serial.println("NTPTime configured");
}
