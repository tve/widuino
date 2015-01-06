LIBRARYPATH = $(ARDUINODIR)/libraries ..
LIBRARIES = JeeLib Net2 Logger2 OwTemp2 OwScan2 OneWire NTPTime Time
SERIALDEV=net:bbb:2001
LINKFLAGS = -Wl,-Map=$(TARGET).map,--cref -Wl,--section-start,.data=0x800104
include ../arduino.mk

run: upload size
	nc bbb 2001
