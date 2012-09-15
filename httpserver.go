package main

import (
    "log"
    "net/http"
    "strconv"
    "strings"
    "sync"

    "code.google.com/p/go.net/websocket"
)

const listenAddr = "localhost:4000"

func main() {

    go broadcastMsg()

    http.HandleFunc("/", rootHandler)
    http.Handle("/socket", websocket.Handler(socketHandler))
    if err := http.ListenAndServe(listenAddr, nil); err != nil {
        log.Fatal(err)
    }
}

func rootHandler(w http.ResponseWriter, r *http.Request) {

    log.Println("rootHandler")
    rootTemplate.Execute(w, listenAddr)
}

var gid = 0
var read = make(chan Msg, 10)
var conns = ConnMap{make(map[string]*websocket.Conn), sync.RWMutex{}}

func socketHandler(conn *websocket.Conn) {

    log.Println("socketHandler")
    gid++
    owner := "p"+strconv.Itoa(gid)
    if conns.Exists(owner) {
        conn.Write([]byte("duplicate id\n"))
        return
    }

    conns.Insert(owner, conn)
    msg := Msg{"system", []byte(owner + " login!\n")}
    read <- msg
    log.Println(conn.RemoteAddr(), "connected!")

    readMsg(conn, owner)

    log.Println(conn.RemoteAddr(), "disconnected!")
    conns.Delete(owner)
    conn.Close()
    msg = Msg{"system", []byte(owner + " logout!\n")}
    read <- msg
}

type Msg struct {
    owner string
    data []byte
}

type ConnMap struct {
    conns map[string]*websocket.Conn
    mutex sync.RWMutex
}

func (cm *ConnMap) Insert(owner string, conn *websocket.Conn) {

    cm.mutex.Lock()
    defer cm.mutex.Unlock()

    cm.conns[owner] = conn
}

func (cm *ConnMap) Exists(owner string) (bool) {

    cm.mutex.RLock()
    defer cm.mutex.RUnlock()

    _, ok := cm.conns[owner]
    return ok
}

func (cm *ConnMap) Delete(owner string) {

    cm.mutex.Lock()
    defer cm.mutex.Unlock()

    delete(cm.conns, owner)
}

func readMsg(conn *websocket.Conn, owner string) {
    tmp := make([]byte, 32)
    buf := make([]byte, 0)
    for {
        readlen, err := conn.Read(tmp)
        if err != nil {
            break
        }

        buf = append(buf, tmp[0 : readlen]...)
        lensep := strings.Index(string(buf), ":")
        if lensep < 0 {
            continue
        }

        msglen, err := strconv.Atoi(string(buf[0:lensep]))
        if err != nil {
            log.Println("error: ", err)
            break
        }

        if len(buf) < msglen + lensep + 1 {
            continue
        }

        msg := Msg{owner, make([]byte, msglen)}
        copy(msg.data, buf[lensep + 1 : msglen + lensep + 1])
        read <- msg
        buf = buf[lensep + msglen + 1: ]
    }
}

func broadcastMsg() {
    for {
        msg := <-read
        buf := make([]byte, 0, len(msg.data) + len(msg.owner) + 2)
        buf = append(buf, msg.owner...)
        buf = append(buf, ": "...)
        buf = append(buf, msg.data...)
        conns.mutex.RLock()
        for _,conn := range conns.conns {
            conn.Write(buf)
        }
        conns.mutex.RUnlock()
    }
}
