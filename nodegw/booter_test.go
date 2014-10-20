// Copyright 2014 by Thorsten von Eicken, see LICENSE in top-level directory
package main

// Omega: Alt+937

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"time"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

//===== tests =====

var _ = Describe("Booter", func() {

	Describe("parses good config file", func() {

		var boo Booter

		BeforeEach(func() {
			boo = NewBooter("test_config1.json")
			time.Sleep(10 * time.Millisecond)
		})

		It("returns a Booter", func() {
			Ω(boo).ShouldNot(BeNil())
		})

		It("returns hardware ID mappings", func() {
			b := boo.(*booter)
			Ω(len(b.nodeType)).Should(Equal(2))
			Ω(b.nodeType["00000000000000000000000000000000"]).
				Should(Equal(pairingInfo{100, 252, 2}))
			Ω(b.nodeType["01020304000000000000000000000000"]).
				Should(Equal(pairingInfo{101, 252, 3}))
		})

		It("returns sketch mappings", func() {
			b := boo.(*booter)
			Eventually(
				func() map[uint16]string { return b.sketch }).
				Should(HaveLen(3))
			Ω(b.sketch[100]).Should(Equal("hex/default.hex"))
			Ω(b.sketch[101]).Should(Equal("hex/tempNode.hex"))
		})

	})

	It("calculates CRC-16", func() {
		crc := software{'a', 'b', 'c'}.Crc16()
		Ω(crc).Should(Equal(uint16(22345)))
	})

	Describe("hwIdIsZero", func() {
		It("returns true if zero", func() {
			hwId := [16]uint8{}
			Ω(hwIdIsZero(hwId)).Should(Equal(true))
		})
		It("returns false if non-zero", func() {
			hwId := [16]uint8{}
			hwId[4] = 1
			Ω(hwIdIsZero(hwId)).Should(Equal(false))
			hwId[4] = 0
			hwId[0] = 1
			Ω(hwIdIsZero(hwId)).Should(Equal(false))
			hwId[0] = 0
			hwId[15] = 1
			Ω(hwIdIsZero(hwId)).Should(Equal(false))
		})
	})

	Describe("Pair", func() {
		var boo Booter
		var req PairingRequest

		BeforeEach(func() {
			boo = NewBooter("test_config1.json")
			time.Sleep(10 * time.Millisecond)
			req = PairingRequest{101, 13, 14, 0, [16]uint8{}}
		})
		It("handles a null hwId", func() {
			rep := boo.Pair(req)
			Ω(rep.NodeType).Should(Equal(uint16(100)))
			Ω(rep.GroupId).Should(Equal(uint8(252)))
			Ω(rep.NodeId).Should(Equal(uint8(2)))
			Ω(rep.ShKey).ShouldNot(Equal([16]uint8{}))
		})
		It("handles a normal hwId", func() {
			req.HwId = [16]uint8{1, 2, 3, 4}
			rep := boo.Pair(req)
			Ω(rep.NodeType).Should(Equal(uint16(101)))
			Ω(rep.GroupId).Should(Equal(uint8(252)))
			Ω(rep.NodeId).Should(Equal(uint8(3)))
			Ω(rep.ShKey).Should(Equal([16]uint8{})) // reply has 0's => don't change
		})
	})

	Describe("readHex", func() {
		var hex, bad_hex string
		BeforeEach(func() {
			hex = ":100000000C9480000C94B9050C94E6050C94A8009F\n" +
				":100010000C94A8000C94A8000C94A8000C94A800C0\n" +
				":100020000C94A8000C94A8000C94A8000C945508FB\n"
			bad_hex = ":100000000C9480000C94B9050C94E6050C94A80090\n"
		})
		It("reads hex", func() {
			sw, err := readHex(strings.NewReader(hex))
			Ω(err).Should(BeNil())
			Ω(sw).ShouldNot(BeEmpty())
			//fmt.Printf("sw=%x", sw)
			Ω(sw).Should(HaveLen(48))
			Ω(fmt.Sprintf("%X", sw)).Should(Equal(
				"0C9480000C94B9050C94E6050C94A800" +
					"0C94A8000C94A8000C94A8000C94A800" +
					"0C94A8000C94A8000C94A8000C945508"))
		})
		It("returns errors", func() {
			sw, err := readHex(strings.NewReader(bad_hex))
			Ω(err).ShouldNot(BeNil())
			Ω(sw).Should(BeEmpty())
		})
	})

	Describe("findSoftware", func() {
		var boo, boo2 *booter

		BeforeEach(func() {
			boo = NewBooter("test_config1.json")
			boo2 = NewBooter("hex/test_config3.json")
			time.Sleep(10 * time.Millisecond)
		})
		It("finds existing software", func() {
			sw, err := boo.findSoftware(100)
			Ω(err).Should(BeNil())
			Ω(sw).ShouldNot(BeEmpty())
			Ω(boo.software).Should(HaveLen(1))
		})
		It("errors for non-existing software", func() {
			sw, err := boo.findSoftware(101)
			Ω(err).ShouldNot(BeNil())
			Ω(sw).Should(BeEmpty())
			Ω(boo.software).Should(HaveLen(0))
		})
		It("errors for non-existing software", func() {
			sw, err := boo.findSoftware(102)
			Ω(err).ShouldNot(BeNil())
			Ω(sw).Should(BeEmpty())
		})
		It("errors for bad sketch files", func() {
			sw, err := boo.findSoftware(113)
			Ω(err).Should(MatchError("line 9: bad checksum -5"))
			Ω(sw).Should(BeEmpty())
		})
		It("finds relative hex files", func() {
			glog.Warningf("Boo2: %#v", boo2)
			sw, err := boo2.findSoftware(100)
			Ω(err).Should(BeNil())
			Ω(sw).ShouldNot(BeEmpty())
			Ω(boo2.software).Should(HaveLen(1))
		})
	})

	Describe("Upgrade", func() {
		var boo Booter
		var req UpgradeRequest

		BeforeEach(func() {
			boo = NewBooter("test_config1.json")
			time.Sleep(10 * time.Millisecond)
			req = UpgradeRequest{100, 55, 1024, 0}
		})
		It("handles an existing software", func() {
			rep := boo.Upgrade(req)
			Ω(rep).ShouldNot(BeNil())
			Ω(rep.NodeType).Should(Equal(req.NodeType))
			Ω(rep.SwId).Should(Equal(req.NodeType))
			Ω(rep.SwSize).Should(Equal(uint16(5024 / 16)))
			Ω(rep.SwCheck).Should(Equal(uint16(61194)))
		})
		It("handles an missing software", func() {
			req.NodeType = 101
			rep := boo.Upgrade(req)
			Ω(rep).Should(BeNil())
		})
	})

	Describe("Download", func() {
		var boo Booter
		var req DownloadRequest

		BeforeEach(func() {
			boo = NewBooter("test_config1.json")
			time.Sleep(10 * time.Millisecond)
			req = DownloadRequest{100, 0}
		})
		It("handles the first chunk of an existing software", func() {
			rep := boo.Download(req)
			Ω(rep).ShouldNot(BeNil())
			Ω(rep.SwIdXorIx).Should(Equal(req.SwId ^ req.SwIndex))
			data := make([]byte, len(rep.Data))
			deWhiten(rep.Data[:], data)
			Ω(fmt.Sprintf("%X", data)).Should(Equal(
				"0C9480000C94B9050C94E6050C94A800" +
					"0C94A8000C94A8000C94A8000C94A800" +
					"0C94A8000C94A8000C94A8000C945508" +
					"0C94A8000C94A8000C94A8000C94A800"))
		})
		It("handles the third chunk of an existing software", func() {
			req.SwIndex = 2
			rep := boo.Download(req)
			Ω(rep).ShouldNot(BeNil())
			Ω(rep.SwIdXorIx).Should(Equal(req.SwId ^ req.SwIndex))
			data := make([]byte, len(rep.Data))
			deWhiten(rep.Data[:], data)
			Ω(fmt.Sprintf("%X", data)).Should(Equal(
				"742E696E6F002A2A2A2A2A2053455455" +
					"503A20736572766F5F746573742E696E" +
					"6F0000000000240027002A0000000000" +
					"250028002B0000000000230026002900"))
		})
		It("handles an missing software", func() {
			req.SwId = 105
			rep := boo.Download(req)
			Ω(rep).Should(BeNil())
		})

	})

	Describe("filesystem notifications", func() {

		var boo Booter
		var defaultPath, shortPath, configPath string
		var defaultName, shortName string
		var shortHex, config []byte

		BeforeEach(func() {
			rand := uint32(time.Now().UnixNano() + int64(os.Getpid()))
			// make a copy of the default sketch hex
			defaultHex, _ := ioutil.ReadFile("hex/default.hex")
			defaultName = fmt.Sprintf("default-%x.hex", rand)
			defaultPath = path.Join(os.TempDir(), defaultName)
			ioutil.WriteFile(defaultPath, defaultHex, 0600)
			// make a copy of the short sketch hex
			shortHex, _ = ioutil.ReadFile("hex/short.hex")
			shortName = fmt.Sprintf("short-%x.hex", rand)
			shortPath = path.Join(os.TempDir(), shortName)
			ioutil.WriteFile(shortPath, shortHex, 0600)
			// load test_config1.json
			config, _ = ioutil.ReadFile("test_config1.json")
			configName := fmt.Sprintf("config-%x.hex", rand)
			configPath = path.Join(os.TempDir(), configName)
			// replace the default sketch hex
			config = bytes.Replace(config, []byte("hex/default.hex"),
				[]byte(defaultName), -1)
			ioutil.WriteFile(configPath, config, 0600)
			// create booter
			boo = NewBooter(configPath)
			time.Sleep(10 * time.Millisecond)
		})

		AfterEach(func() {
			os.Remove(defaultPath)
			os.Remove(shortPath)
			os.Remove(configPath)
		})

		It("reloads when the config changes", func() {
			b := boo.(*booter)

			hasShortSketch := func() bool {
				for _, s := range b.sketch {
					if s == shortPath {
						return true
					}
				}
				return false
			}

			Ω(hasShortSketch()).Should(Equal(false))

			config = bytes.Replace(config, []byte(defaultName), []byte(shortPath), -1)
			ioutil.WriteFile(configPath, config, 0600)
			Eventually(hasShortSketch).Should(Equal(true))
		})

		It("reloads when a sketch changes", func() {
			b := boo.(*booter)

			Eventually(func() int {
				sw, _ := b.findSoftware(100)
				return len(sw)
			}).Should(Equal(5024))

			glog.Infof("===== writing %s =====", defaultPath)
			ioutil.WriteFile(defaultPath, shortHex, 0600)
			Eventually(func() int {
				sw, _ := b.findSoftware(100)
				return len(sw)
			}).Should(Equal(240))
		})

	})
})
