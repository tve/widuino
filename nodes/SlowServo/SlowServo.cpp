#include <JeeLib.h>
#include <Servo.h>
#include <SlowServo.h>

// number of usec steps to move per millisecond
#define SERVO_STEP 2

// Note: modify turnigy servo to 180 degree rotation by adding 2.7Kohm resistor on black
// lead of potentiometer

SlowServo::SlowServo(byte pin, int min, int max) {
  this->pin = pin;
  minPos = min;
  maxPos = max;
  servoPos = -1;
  servoTarget = 0;
}

void SlowServo::attach(int initial) {
  servoPos = initial;
  if (servoPos < minPos) servoPos = minPos;
  if (servoPos > maxPos) servoPos = maxPos;
  servoTarget = servoPos;
  servo.attach(pin, minPos, maxPos);
  servo.writeMicroseconds(servoPos);
  Serial.print("Servo init: ");
  Serial.println(servoPos);
}

void SlowServo::detach() {
  servo.detach();
}

uint8_t SlowServo::read() {
  return map(servo.readMicroseconds(), minPos, maxPos, 0, 180);
}

void SlowServo::step() {
  if (servoPos >= servoTarget+SERVO_STEP)
    servoPos -= SERVO_STEP;
  else if (servoPos > servoTarget)
    servoPos = servoTarget;
  else if (servoPos > servoTarget)
    servoPos = servoTarget;
  else if (servoPos <= servoTarget-SERVO_STEP)
    servoPos += SERVO_STEP;
  else if (servoPos < servoTarget)
    servoPos = servoTarget;
  Serial.print("step: ");
  Serial.println(servoPos);
  servo.writeMicroseconds(servoPos);
  if (servoPos == servoTarget) {
    timer.set(0);
    //EEPROM.write(SERVO_EEPROM,   servoPos & 0xFF);
    //EEPROM.write(SERVO_EEPROM+1, servoPos >> 8);
  }
}

void SlowServo::write(uint8_t pos) {
  if (pos > 180) pos = 180;
  servoTarget = map(pos, 0, 180, minPos, maxPos);
  if (servoPos == (uint16_t)-1) Serial.println("servo not initialized");
  if (servoPos != (uint16_t)-1) {
    timer.set(2);
    step();
  }
}

void SlowServo::writeNow(uint8_t pos) {
  if (pos > 180) pos = 180;
  servoTarget = map(pos, 0, 180, minPos, maxPos);
  if (servoPos == (uint16_t)-1) Serial.println("servo not initialized");
  if (servoPos != (uint16_t)-1) servo.writeMicroseconds(servoTarget);
  servoPos = servoTarget;
}

void SlowServo::loop() {
  if (!timer.idle() && timer.poll(2)) step();
}

/*

SlowServo p3servo(6);
int pos = 0;
MilliTimer t2;

void setup(void) {
  Serial.begin(57600);
  Serial.print("\n***** RUNNING: ");
	Serial.println(__FILE__);

  p3servo.attach();
  
  buttons.ledOn(1);
  delay(500);
  buttons.ledOff(1+2);

  //p3servo.writeMicroseconds(450);
  //Serial.println(p3servo.readMicroseconds());
  //delay(1000);
  //p3servo.writeMicroseconds(1800);
  //Serial.println(p3servo.readMicroseconds());
  //delay(1000);
  //p3servo.writeNow(0);
  //Serial.println(p3servo.readMicroseconds());
  //delay(1000);
  //p3servo.writeNow(180);
  //Serial.println(p3servo.readMicroseconds());
  //delay(1000);
  t2.set(5000);
}


void loop(void) {
  if (t2.poll(5000)) {
    Serial.print("*** write ");
    Serial.println(45+pos*90);
    p3servo.write(45+pos*90);
    pos = 1 - pos;
  }
  p3servo.loop();

}
*/
