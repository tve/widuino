// Copyright 2014 by Thorsten von Eicken, see LICENSE in top-level directory
package main

// Omega: Alt+937

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"net"
)

//===== tests =====

var _ = Describe("UDPGw GroupMap", func() {
	var gm *GroupMap
	var addr, addr2 *net.UDPAddr

	BeforeEach(func() {
		gm = &GroupMap{}
		addr = &net.UDPAddr{net.IP{1, 2, 3, 4}, 101, "foo"}
		addr2 = &net.UDPAddr{net.IP{1, 2, 3, 4}, 55, "foo"}
	})

	Describe("saveGroupToAddr", func() {
		It("saves a new group", func() {
			res := gm.saveGroupToAddr(1, addr)
			Ω(res).Should(Equal(true))
			Ω(gm.group).Should(HaveLen(1))
		})
		It("saves two groups", func() {
			res := gm.saveGroupToAddr(1, addr)
			res = gm.saveGroupToAddr(2, addr)
			Ω(res).Should(Equal(true))
			Ω(gm.group).Should(HaveLen(2))
		})
		It("handles duplicates", func() {
			res := gm.saveGroupToAddr(1, addr)
			res = gm.saveGroupToAddr(1, addr)
			Ω(res).Should(Equal(false))
			Ω(gm.group).Should(HaveLen(1))
		})
	})

	Describe("mapGroupToAddr", func() {
		It("retrieves a group", func() {
			_ = gm.saveGroupToAddr(1, addr)
			res := gm.mapGroupToAddr(1)
			Ω(res).Should(Equal(addr))
		})
		It("retrieves two groups", func() {
			_ = gm.saveGroupToAddr(1, addr)
			_ = gm.saveGroupToAddr(2, addr2)
			res := gm.mapGroupToAddr(1)
			Ω(res).Should(Equal(addr))
			Ω(res.Port).Should(Equal(101))
			res = gm.mapGroupToAddr(2)
			Ω(res).Should(Equal(addr2))
			Ω(res.Port).Should(Equal(55))
		})
		It("retrieves updated groups", func() {
			_ = gm.saveGroupToAddr(1, addr)
			res := gm.mapGroupToAddr(1)
			Ω(res).Should(Equal(addr))
			Ω(res.Port).Should(Equal(101))
			// now update
			_ = gm.saveGroupToAddr(1, addr2)
			res = gm.mapGroupToAddr(1)
			Ω(res).Should(Equal(addr2))
			Ω(res.Port).Should(Equal(55))
		})
	})

})
