package main

import (
    "log"
    "net"
    "strconv"
    "sync"
)

type Msg struct {
    owner string
    data []byte
}

type ConnMap struct {
    conns map[string]net.Conn
    mutex sync.Mutex
}

func (cm *ConnMap) Insert(owner string, conn net.Conn) {
    cm.mutex.Lock()
    cm.conns[owner] = conn
    cm.mutex.Unlock()
}

func (cm *ConnMap) Exists(owner string) (bool) {
    cm.mutex.Lock()
    _, ok := cm.conns[owner]
    cm.mutex.Unlock()
    return ok
}

func (cm *ConnMap) Delete(owner string) {
    cm.mutex.Lock()
    delete(cm.conns, owner)
    cm.mutex.Unlock()
}

var read = make(chan Msg, 10)
var conns = ConnMap{make(map[string]net.Conn), sync.Mutex{}}

func main() {
    ln, err := net.Listen("tcp", ":5000")
    if err != nil {
        log.Println(err)
        return
    }

    go broadcastMsg()

    id := 0
    for {
        conn, err := ln.Accept()
        if err != nil {
            continue
        }
        id++
        owner := "p"+strconv.Itoa(id)
        go handleConnection(conn, owner)
    }
}

func handleConnection(conn net.Conn, owner string) {
    defer closeConnection(conn, owner)

    if conns.Exists(owner) {
        conn.Write([]byte("duplicate id\n"))
        return
    }
    conns.Insert(owner, conn)
    msg := Msg{"system", []byte(owner + " login!\n")}
    read <- msg
    log.Println(conn.RemoteAddr(), "connected!")

    readMsg(conn, owner)
}

func readMsg(conn net.Conn, owner string) {
    const lensize = 2
    tmp := make([]byte, 32)
    buf := make([]byte, 0)
    for {
        readlen, err := conn.Read(tmp)
        if err != nil {
            break
        }

        buf = append(buf, tmp[0 : readlen]...)
        if len(buf) < lensize {
            continue
        }

        msglen := int(buf[0]) + int(buf[1])<<8
        if len(buf) < msglen + lensize {
            continue
        }

        msg := Msg{owner, make([]byte, msglen)}
        copy(msg.data, buf[lensize : msglen + lensize])
        read <- msg
        buf = buf[lensize + msglen : ]
    }
}

func closeConnection(conn net.Conn, owner string) {
    log.Println(conn.RemoteAddr(), "disconnected!")
    conns.Delete(owner)
    conn.Close()
    msg := Msg{"system", []byte(owner + " logout!\n")}
    read <- msg
}

func broadcastMsg() {
    for {
        msg := <-read
        buf := make([]byte, 0, len(msg.data) + len(msg.owner) + 2)
        buf = append(buf, msg.owner...)
        buf = append(buf, ": "...)
        buf = append(buf, msg.data...)
        conns.mutex.Lock()
        for _,conn := range conns.conns {
            conn.Write(buf)
        }
        conns.mutex.Unlock()
    }
}
