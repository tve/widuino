// Copyright 2014 by Thorsten von Eicken, see LICENSE in top-level directory
package main

// Omega: Alt+937

import (
	"fmt"
	"net"
	"os"
	"time"

	"./database"

	"github.com/docker/libchan"
	"github.com/docker/libchan/spdy"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

//===== tests =====

var _ = Describe("Libchan server", func() {

	var clientConn net.Conn
	var clientSender libchan.Sender
	var serverListener net.Listener

	var dbDir string

	BeforeEach(func() {
		dbDir = fmt.Sprintf("/tmp/db-%d", os.Getpid())
		db, _ = database.Open(dbDir) // db is global defined in main.go

		var err error
		serverListener, err = net.Listen("tcp", "localhost:9323")
		Ω(err).ShouldNot(HaveOccurred())
		go ServeChan(serverListener)
		time.Sleep(10 * time.Millisecond)

		clientConn, err = net.Dial("tcp", "127.0.0.1:9323")
		Ω(err).ShouldNot(HaveOccurred())

		transport, err := spdy.NewClientTransport(clientConn)
		Ω(err).ShouldNot(HaveOccurred())

		clientSender, err = transport.NewSendChannel()
		Ω(err).ShouldNot(HaveOccurred())
	})

	AfterEach(func() {
		if clientConn != nil {
			clientConn.Close()
		}
		if serverListener != nil {
			fmt.Printf("Closing serverListener\n")
			serverListener.Close()
		}
		time.Sleep(10 * time.Millisecond)

		db.Close()
		os.RemoveAll(dbDir)
	})

	It("responds to echo requests", func() {
		responseRecv, responseSend := libchan.Pipe()

		req := &EchoRequest{
			Cmd:   "echo",
			Text:  "Hello world!",
			Reply: responseSend,
		}

		err := clientSender.Send(req)
		Ω(err).ShouldNot(HaveOccurred())

		response := &EchoReply{}
		err = responseRecv.Receive(response)
		Ω(err).ShouldNot(HaveOccurred())

		//fmt.Println(response.Text)
		Ω(response.Text).Should(Equal(req.Text))
	})

	It("processes subscriptions", func() {
		// Create some initial messages
		now := time.Now().Unix()
		for i := 0; i < 10; i += 1 {
			m := database.RFMessage{At: now + int64(i), Group: byte(2 * i),
				Node: 13, Data: []byte(fmt.Sprintf("Hello %d", i))}
			err := db.PutRFMessage(m)
			Ω(err).ShouldNot(HaveOccurred())
		}

		// Send request to subscribe
		responseRecv, responseSend := libchan.Pipe()
		req := &SubRequest{
			Cmd:   "sub",
			Start: now + 4,
			Match: database.RFMessage{Node: 13},
			Reply: responseSend,
		}
		err := clientSender.Send(req)
		Ω(err).ShouldNot(HaveOccurred())

		// Create some more messages
		for i := 10; i < 20; i += 1 {
			m := database.RFMessage{At: now + int64(i), Group: byte(2 * i),
				Node: 13, Data: []byte(fmt.Sprintf("Hello %d", i))}
			err := db.PutRFMessage(m)
			Ω(err).ShouldNot(HaveOccurred())
		}

		// Receive responses
		cnt := 0
		for cnt < 16 {
			m := database.RFMessage{}
			err = responseRecv.Receive(&m)
			Ω(err).ShouldNot(HaveOccurred())

			//fmt.Printf("Reply: %+v\n", m)
			Ω(m.At).Should(Equal(now + 4 + int64(cnt)))
			cnt += 1
		}
	})

})
