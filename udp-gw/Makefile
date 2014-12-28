# Boards: 0=defaults, 1=antenna GW, 2=basement GW, 3=other GW
CPPFLAGS = -DBOARD=3

all: rf12-udp-gw.hex udpgw

HOST      = bbb
SERIALDEV = net:$(HOST):2000
#SERIALDEV = /dev/ttyUSB0
include ./arduino.mk

run: upload size
	nc $(HOST) 2001

udpgw: udpgw.go
	go build udpgw.go
