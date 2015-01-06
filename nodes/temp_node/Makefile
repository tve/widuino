LIBRARYPATH = $(ARDUINODIR)/libraries ..
LIBRARIES = JeeLib Net EEConf Logger OwTemp2 OwScan2 OneWire NTPTime Time
LINKFLAGS = -Wl,--section-start,.data=0x800104
include ../arduino.mk

run: upload size
	nc bbb 2001
