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

var _ = Describe("Database Subscribe", func() {

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

	It("functions correctly", func() {
		now := time.Now().Unix()
		for i := 0; i < 150; i += 1 {
			m := RFMessage{At: now + int64(i), Group: byte(2 * i),
				Node: 13, Data: []byte(fmt.Sprintf("Hello %d", i))}
			err := db.PutRFMessage(m)
			Ω(err).ShouldNot(HaveOccurred())
		}

		cnt := 0
		c := db.Subscribe(now + 4)
		go func() {
			for _ = range c {
				cnt += 1
				//fmt.Printf("Subscriber got %d=%d\n", m.At, m.At-now)
				if cnt >= 2 && cnt <= 4 {
					m1 := RFMessage{At: now + int64(1000+cnt)}
					err := db.PutRFMessage(m1)
					Ω(err).ShouldNot(HaveOccurred())
				}
			}
		}()

		time.Sleep(time.Millisecond)
		for i := 1020; i < 1025; i += 1 {
			m := RFMessage{At: now + int64(i), Group: byte(2 * i),
				Node: 13, Data: []byte(fmt.Sprintf("Hello %d", i))}
			err := db.PutRFMessage(m)
			Ω(err).ShouldNot(HaveOccurred())
		}

		time.Sleep(time.Millisecond)
		db.Unsubscribe(c)
		time.Sleep(time.Millisecond)

		for i := 1030; i < 1035; i += 1 {
			m := RFMessage{At: now + int64(i), Group: byte(2 * i),
				Node: 13, Data: []byte(fmt.Sprintf("Hello %d", i))}
			err := db.PutRFMessage(m)
			Ω(err).ShouldNot(HaveOccurred())
		}

		Eventually(func() int { return cnt }).Should(Equal(150 - 4 + 3 + 5))
	})

})
