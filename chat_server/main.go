package main

func main() {
	ch := NewChatServer(0, 0, 0, 0, 9000)
	ch.Serve()
	ch.Close()
}
