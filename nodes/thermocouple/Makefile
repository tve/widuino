LIBRARYPATH = $(ARDUINODIR)/libraries ..
LIBRARIES = GLCD_ST7565 JeeLib Net EEConf Logger
SERIALDEV=net:bbb:2010
include ../arduino.mk

run: upload size
	nc bbb 2011
