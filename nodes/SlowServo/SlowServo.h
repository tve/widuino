// C 2013 Thorsten von Eicken

#ifndef SlowServo_h
#define SlowServo_h

#include <Servo.h>

// Note: modify turnigy servo to 180 degree rotation by adding 2.7Kohm resistor on black
// lead of potentiometer

class SlowServo {
public:
  // Create the servo interface. pin: arduino pin
  // min/max: min and max positions in usecs
  SlowServo(byte pin, int min=450, int max=2000);
  // Attach the servo: starts driving the servo output pin
  // initial: initial position in usecs
  void attach(int initial=450);
  // Detach the servo: stops driving the servo output pin
  void detach();
  // Move the servo to the position *slowly*
  void write(uint8_t pos);    // 0..180
  // Move the servo the the position immediately
  void writeNow(uint8_t pos); // 0..180 FAST
  // Read the current servo position
  uint8_t read();
  // Call in the main arduino loop to keep the servo moving
  void loop();

protected:
  Servo servo;
  MilliTimer timer;
  byte pin;
  uint16_t minPos, maxPos;
  uint16_t servoPos;
  uint16_t servoTarget;

  void step();
};

#endif
