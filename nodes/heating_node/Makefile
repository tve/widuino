LIBRARYPATH = $(ARDUINODIR)/libraries ..
LIBRARIES = JeeLib Net EEConf Logger NTPTime OwTemp OwRelay OwScan OneWire Time
SERIALDEV=net:bbb:2001
include ../arduino.mk

run: upload size
	nc bbb 2001
