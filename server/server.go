package server

import (
	"bufio"
	"fmt"
	"net"
)

func main() {
	ln, _ := net.Listen("tcp", ":8030")
	conn, _ := ln.Accept()
	reader := bufio.NewReader(conn)
	msg, _ := reader.ReadString('\n')
	fmt.Println(msg)
}
