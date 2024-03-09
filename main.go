package main

func main() {
	server := NewApiServer(":8090")
	server.Start()
}
