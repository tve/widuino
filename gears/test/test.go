package main

import (
	"fmt"
	"log"
	"reflect"
)
import "github.com/dmcgowan/go/codec"

type A struct {
	A1 int
	A2 string
}
type B struct {
	B1 string
	B2 int64
}
type MSG struct {
	A *A
	B *B
}

func main() {
	mh := codec.MsgpackHandle{}
	mh.MapType = reflect.TypeOf(map[string]interface{}(nil))
	buf := make([]byte, 10240)

	b := B{"hello", 3}
	msg_b := struct{ B B }{b}
	fmt.Printf("Send %#v\n", msg_b)

	enc := codec.NewEncoderBytes(&buf, &mh)
	err := enc.Encode(msg_b)
	if err != nil {
		log.Fatalf("Encode: %s", err)
	}

	var rcv interface{}
	dec := codec.NewDecoderBytes(buf, &mh)
	err = dec.Decode(&rcv)
	if err != nil {
		log.Fatalf("Decode: %s", err)
	}
	fmt.Printf("Raw Recv: %#v\n", rcv)

	var msg MSG
	dec = codec.NewDecoderBytes(buf, &mh)
	err = dec.Decode(&msg)
	if err != nil {
		log.Fatalf("Decode: %s", err)
	}
	fmt.Printf("Msg Recv: %#v\n", msg)
	fmt.Printf("Msg.A Recv: %#v\n", msg.A)
	fmt.Printf("Msg.B Recv: %#v\n", msg.B)
}
