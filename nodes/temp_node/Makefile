LIBRARYPATH = $(ARDUINODIR)/libraries ..
LIBRARIES = JeeLib Net OwTemp OwScan OneWire
include ../arduino.mk

run: upload size
	nc bbb 2001
