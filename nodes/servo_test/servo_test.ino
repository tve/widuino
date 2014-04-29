// Copyright (c) 2013-2014 by Thorsten von Eicken
//
// Simple servo test

#include <JeeLib.h>
#include <avr/eeprom.h>
#include <SlowServo.h>

#define SERVO_PORT   2
#define SERVO_OPEN   0
#define SERVO_CLOSED 180

SlowServo servo(SERVO_PORT+3, 450, 1800);

MilliTimer t;

//===== setup & loop =====

void setup() {
  Serial.begin(57600);
  Serial.println(F("***** SETUP: " __FILE__));

  servo.attach(1500);
  Serial.println(F("***** RUNNING: " __FILE__));
}

boolean on;

void loop() {
  if (on) {
    if (t.poll(2000)) {
      Serial.println("CLOSED=180");
      servo.write(SERVO_CLOSED);
      on = false;
    }
  } else {
    if (t.poll(2000)) {
      Serial.println("OPEN=0");
      servo.write(SERVO_OPEN);
      on = true;
    }
  }
  servo.loop();
}

