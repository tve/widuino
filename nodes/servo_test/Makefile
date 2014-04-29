LIBRARYPATH = $(ARDUINODIR)/libraries ..
LIBRARIES = JeeLib Servo SlowServo
include ../arduino.mk

run: upload size
	nc bbb 2001
