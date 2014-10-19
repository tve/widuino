// Copyright (c) 2014 Thorsten von Eicken
//
// Subscribe to an MQTT topic and print all messages

package main

import (
        "flag"
        proto "github.com/huin/mqtt"
        "github.com/jeffallen/mqtt"
        "log"
        "net"
)

var host = flag.String("host", "localhost:1883", "hostname of MQTT broker")
var user = flag.String("user", "", "username")
var pass = flag.String("pass", "", "password")

func main() {
        flag.Parse()

        if flag.NArg() < 1 {
                log.Fatal("usage: mqttlog topic [topic topic...]")
        }

        // Connect to MQTT broker
        conn, err := net.Dial("tcp", *host)
        if err != nil {
                log.Fatalf("MQTT broker: ", err)
        }
        cc := mqtt.NewClientConn(conn)

        // Send connect message
        if err := cc.Connect(*user, *pass); err != nil {
                log.Fatalf("connect: %s", err)
        }

        // Subscribe to requested topics
        tq := make([]proto.TopicQos, flag.NArg())
        for i := 0; i < flag.NArg(); i++ {
                tq[i].Topic = flag.Arg(i)
                tq[i].Qos = proto.QosAtMostOnce
        }
        cc.Subscribe(tq)

        // Print everything we receive
        log.Println("Connected with client id", cc.ClientId)
        for m := range cc.Incoming {
                payload := []byte(m.Payload.(proto.BytesPayload))
                log.Printf("%s (%d bytes)", m.TopicName, len(payload))
                printable := true
                for _, c := range payload {
                        printable = printable && c >= ' ' && c <= '~'
                }
                if printable {
                        log.Printf("  %s", payload)
                } else {
                        log.Printf("  %q", payload)
                }
        }
}
