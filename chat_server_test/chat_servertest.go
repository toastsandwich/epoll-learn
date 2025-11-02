package main

import (
	"fmt"
	"net"
	"sync"
	"time"
)

func dialToChatServerAndSendHI(wg *sync.WaitGroup) {
	defer wg.Done()
	conn, err := net.Dial("tcp", "192.168.1.4:9000")
	if err != nil {
		fmt.Println("error dialing:", err)
		return
	}
	for range 1000 {
		msg := fmt.Sprintf("Hi from %s\n", conn.LocalAddr().String())
		_, err := conn.Write([]byte(msg))
		if err != nil {
			fmt.Println("error writing message:", err)
		}
	}
	conn.Close()
}

func TestChatServer() {
	wg := &sync.WaitGroup{}

	for range 10_000 {
		wg.Add(1)
		go dialToChatServerAndSendHI(wg)
		time.Sleep(1 * time.Millisecond)
	}
	wg.Wait()
}

func main() {
	TestChatServer()
}
