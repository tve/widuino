// Copyright 2014 by Thorsten von Eicken, see LICENSE in top-level directory

package database

// Omega: Alt+937

import (
	"fmt"
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

//===== tests =====

var _ = Describe("Database RFMessage", func() {

	var dir string
	var db *DB

	BeforeEach(func() {
		dir = fmt.Sprintf("/tmp/db-%d", os.Getpid())
		var err error
		db, err = Open(dir)
		Ω(err).ShouldNot(HaveOccurred())
	})

	AfterEach(func() {
		db.Close()
		os.RemoveAll(dir)
	})

	It("generates keys", func() {
		k, err := parseRFKey(genRFKey(123456789012))
		Ω(err).ShouldNot(HaveOccurred())
		Ω(k).Should(Equal(int64(123456789012)))
	})

	It("iterates", func() {
		now := time.Now().Unix()
		for i := 0; i < 20; i += 1 {
			m := RFMessage{At: now + int64(i), Group: byte(2 * i),
				Node: 13, Data: []byte(fmt.Sprintf("Hello %d", i))}
			err := db.PutRFMessage(m)
			Ω(err).ShouldNot(HaveOccurred())
		}

		cnt := 0
		err := db.RFIterate(now+3, now+16, func(m RFMessage) error {
			Ω(m.At).Should(Equal(now + int64(cnt+3)))
			Ω(m.Group).Should(Equal(byte(2 * (cnt + 3))))
			Ω(m.Node).Should(Equal(byte(13)))
			Ω(m.Data).Should(Equal([]byte(fmt.Sprintf("Hello %d", cnt+3))))
			cnt += 1
			return nil
		})
		Ω(err).ShouldNot(HaveOccurred())
		Ω(cnt).Should(Equal(13))
	})

	It("processes a channel", func() {
		c := make(chan RFMessage)
		NewProcessor(db)(c)

		now := time.Now().Unix()
		for i := 0; i < 20; i += 1 {
			m := RFMessage{At: now + int64(i), Group: byte(2 * i),
				Node: 13, Data: []byte(fmt.Sprintf("Hello %d", i))}
			c <- m
		}
		time.Sleep(time.Millisecond)

		cnt := 0
		err := db.RFIterate(now+3, now+16, func(m RFMessage) error {
			Ω(m.At).Should(Equal(now + int64(cnt+3)))
			Ω(m.Group).Should(Equal(byte(2 * (cnt + 3))))
			Ω(m.Node).Should(Equal(byte(13)))
			Ω(m.Data).Should(Equal([]byte(fmt.Sprintf("Hello %d", cnt+3))))
			cnt += 1
			return nil
		})
		Ω(err).ShouldNot(HaveOccurred())
		Ω(cnt).Should(Equal(13))
	})

})
