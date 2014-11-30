// Copyright 2014 by Thorsten von Eicken, see LICENSE in top-level directory

package database

// Omega: Alt+937

import (
	"fmt"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type testStruct struct {
	I int
	S string
}

//===== tests =====

var _ = Describe("Database store", func() {

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

	It("returns nil", func() {
		var v interface{}
		err := db.Get("test1", v)
		Ω(err).Should(MatchError(ErrNotFound))
		Ω(v).Should(BeNil())
	})

	It("returns an integer", func() {
		err := db.Put("test2", 35)
		Ω(err).ShouldNot(HaveOccurred())

		var v int
		err = db.Get("test2", &v)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(v).Should(Equal(35))
	})

	It("returns a struct", func() {
		v1 := testStruct{34, "hello"}
		err := db.Put("test3", &v1)
		Ω(err).ShouldNot(HaveOccurred())

		var v testStruct
		err = db.Get("test3", &v)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(v).Should(Equal(v1))
	})

	It("iterates", func() {
		for i := 0; i < 20; i += 1 {
			k := fmt.Sprintf("series/%02d", i)
			err := db.Put(k, i)
			Ω(err).ShouldNot(HaveOccurred())
		}

		var v int
		sum := 0
		err := db.Iterate("series/03", "series/13", &v, func(key string) error {
			//fmt.Printf("Got %s => %d\n", key, v)
			sum += v
			return nil
		})
		Ω(err).ShouldNot(HaveOccurred())
		Ω(sum).Should(Equal(3 + 4 + 5 + 6 + 7 + 8 + 9 + 10 + 11 + 12))
	})

	It("iterates through prefix", func() {
		for i := 0; i <= 20; i += 1 {
			k := fmt.Sprintf("series/%02d", i)
			err := db.Put(k, i)
			Ω(err).ShouldNot(HaveOccurred())
		}

		var v int
		sum := 0
		err := db.Iterate("series/", "", &v, func(key string) error {
			sum += v
			return nil
		})
		Ω(err).ShouldNot(HaveOccurred())
		Ω(sum).Should(Equal(210))
	})

	It("stops iterating on error", func() {
		for i := 0; i < 20; i += 1 {
			k := fmt.Sprintf("series/%02d", i)
			err := db.Put(k, i)
			Ω(err).ShouldNot(HaveOccurred())
		}

		var v int
		sum := 0
		err := db.Iterate("series/03", "series/13", &v, func(key string) error {
			if v == 8 {
				return fmt.Errorf("hello")
			}
			sum += v
			return nil
		})
		Ω(err).Should(MatchError("hello"))
		Ω(sum).Should(Equal(3 + 4 + 5 + 6 + 7))
	})
})
