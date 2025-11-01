package main

import (
	"fmt"
	"net"
	"sync"
	"time"
)

func dialToChatServerAndSendHI(wg *sync.WaitGroup) {
	defer wg.Done()
	conn, err := net.Dial("tcp", "0.0.0.0:9000")
	if err != nil {
		fmt.Println("error dialing:", err)
		return
	}
	msg := fmt.Sprintf("Hi from %s\n", conn.LocalAddr().String())
	conn.Write([]byte(msg))
	conn.Close()
}

func TestChatServer() {
	wg := &sync.WaitGroup{}

	for range 100000 {
		wg.Add(1)
		go dialToChatServerAndSendHI(wg)
		time.Sleep(1 * time.Millisecond)
	}
	wg.Wait()
}

func main() {
	TestChatServer()
}
