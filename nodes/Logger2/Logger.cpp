// Copyright (c) 2013-2014 Thorsten von Eicken
//
// Logging class
//

#include <JeeLib.h>
#include <Net.h>
#include <Logger.h>

#define RETRY_MS 300   // for how many milliseconds to retry sending

Logger::Logger(uint8_t destinations) {
  dest = destinations;
  ix = 0;
}

uint8_t *Logger::allocPkt() {
  uint8_t *pkt = net.alloc();
  // if we didn't get a buffer we retry for some time
  if (!pkt) {
    unsigned long t0 = millis();
    while (!pkt && (millis()-t0) < RETRY_MS) {
      delay(1);
      (void)net.poll();
      pkt = net.alloc();
    }
  }
  return pkt;
}

void Logger::send(void) {
  buffer[ix] = 0;

  // Log to the serial port
  if (dest & LOG_SERIAL) {
    //Serial.print("LOG:");
    Serial.print((char *)buffer);
    if (ix > 0 && buffer[ix-1] == '\n')
      Serial.print('\r');
  }

#ifndef LOG_NORF12B
  // Log to the network
  if (dest & LOG_RF12) {
    // if we missed messages then say so first
    if (missed > 0) {
      uint8_t *pkt = allocPkt();
      uint8_t *pkt0 = pkt;
      if (pkt) {
        *pkt++ = LOG_MODULE;
	memcpy(pkt, "<lost ", 6); pkt += 6;
	if (missed > 10) *pkt++ = '0' + missed/10;
	*pkt++ = '0' + (missed%10);
	memcpy(pkt, " lines>", 7); pkt += 7;
	net.send(pkt-pkt0, true);
        missed = 0;
      } else {
	if (missed < 99) missed++;
	return;
      }
    }

    // Now send the actual message we have
    uint8_t *pkt = allocPkt();
    if (pkt) {
      *pkt = LOG_MODULE;
      memcpy(pkt+1, buffer, ix);
      net.send(ix+1, true); // +1 for module_id byte
      missed = 0;
    } else {
      if (missed < 255) missed++;
      //Serial.println(F("Log: out of rf12 buffers"));
    }
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
