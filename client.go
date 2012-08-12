package main

import (
    "bufio"
    "io"
    "log"
    "net"
    "os"
)

func main() {
    conn, err := net.Dial("tcp", "localhost:5000")
    if err != nil {
        log.Printf("connect failed")
        return
    }
    defer conn.Close()

    go sendMsg(conn)

    io.Copy(os.Stdout, conn)
}

func sendMsg(conn net.Conn) {
    input := bufio.NewReader(os.Stdin)
    for {
        msg,err := input.ReadBytes('\n')
        if err != nil {
            break
        }
        length := len(msg)
        if length >= 1<<16 {
            continue
        }
        low := byte(length&0xff)
        high := byte(length>>8&0xff)
        msg = append([]byte{low, high}, msg...)
        conn.Write(msg)
    }
}
