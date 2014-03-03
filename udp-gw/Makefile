all: rf12-udp-gw.hex udpgw

HOST      = bbb
SERIALDEV = net:$(HOST):2000
include ./arduino.mk

run: upload size
	nc $(HOST) 2001

udpgw: udpgw.go
	go build udpgw.go
