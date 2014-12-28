// Copyright 2014 by Thorsten von Eicken, see LICENSE in top-level directory

package main

import (
	"fmt"
	"strings"

	"github.com/tve/widuino/gears"
)

func RFFormat(m gears.RFMessage) string {
	switch m.Kind {
	//case 1: // Net
	case 2: // Log
		return fmt.Sprintf("Log: %s", strings.TrimSuffix(string(m.Data), "\n"))
	case 4: // Temp
		str := "Temp:"
		for _, t := range m.Data {
			str += fmt.Sprintf(" %dF", t)
		}
		return str
	case 7: // Water level
		if len(m.Data) != 4 {
			return fmt.Sprintf("Water level: %d bytes? % x", len(m.Data), m.Data)
		}
		v1 := float32(uint16(m.Data[1])<<8|uint16(m.Data[0])) * 3.3 / 1024
		v2 := float32(uint16(m.Data[3])<<8|uint16(m.Data[2])) * 3.3 / 1024
		return fmt.Sprintf("Water levels: %.3fV %.3fV", v1, v2)
	case 8: // GW RSSI
		if len(m.Data) < 4 {
			return fmt.Sprintf("RF RSSI %d bytes? % x", len(m.Data), m.Data)
		}
		str := fmt.Sprintf("RF: %ds/%dr Eth: %ds/%dr",
			m.Data[0], m.Data[1], m.Data[2], m.Data[3])
		for i := 4; i+1 < len(m.Data); i += 2 {
			if m.Data[i] != 0 || m.Data[i+1] != 0 {
				str += fmt.Sprintf(" i%d:%d/%d", i, m.Data[i], m.Data[i+1])
			}
		}
		return str
	}
	return fmt.Sprintf("unknown: % x", m.Data)
}
