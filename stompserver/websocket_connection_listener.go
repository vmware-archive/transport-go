// Copyright 2019 VMware, Inc. All rights reserved. -- VMware Confidential

package stompserver

import (
    "github.com/go-stomp/stomp/frame"
    "github.com/gorilla/websocket"
    "net"
    "net/http"
    "net/url"
    "strings"
    "time"
)

type webSocketStompConnection struct {
    wsCon *websocket.Conn
}

func (c *webSocketStompConnection) ReadFrame() (*frame.Frame, error) {
    _, r, err := c.wsCon.NextReader()
    if err != nil {
        return nil, err
    }
    frameR := frame.NewReader(r)
    f, e := frameR.Read()
    return f,e
}

func (c *webSocketStompConnection) WriteFrame(f *frame.Frame) error {
    wr, err := c.wsCon.NextWriter(websocket.TextMessage)
    if err != nil {
        return err
    }
    frameWr := frame.NewWriter(wr)
    err = frameWr.Write(f)
    if err != nil {
        return err
    }
    err = wr.Close()
    return err
}

func (c *webSocketStompConnection) SetReadDeadline(t time.Time) {
    c.wsCon.SetReadDeadline(t)
}

func (c *webSocketStompConnection) Close() error {
    return c.wsCon.Close()
}

type webSocketConnectionListener struct {
    httpServer *http.Server
    requestHandler *http.ServeMux
    tcpConnectionListener net.Listener
    connectionsChannel chan rawConnResult
    allowedOrigins []string
}

type rawConnResult struct {
    conn RawConnection
    err error
}

func NewWebSocketConnectionListener(addr string, endpoint string, allowedOrigins []string) (RawConnectionListener, error) {
    rh := http.NewServeMux()
    l := &webSocketConnectionListener{
        requestHandler: rh,
        httpServer: &http.Server{
            Addr: addr,
            Handler: rh,
        },
        connectionsChannel: make(chan rawConnResult),
        allowedOrigins: allowedOrigins,
    }

    var upgrader = websocket.Upgrader{
        ReadBufferSize:  1024,
        WriteBufferSize: 1024,
    }

    upgrader.CheckOrigin = l.checkOrigin

    rh.HandleFunc(endpoint, func(writer http.ResponseWriter, request *http.Request) {
        conn, err := upgrader.Upgrade(writer, request, nil)
        if err != nil {
            l.connectionsChannel <- rawConnResult{ err: err}


        } else {
            l.connectionsChannel <- rawConnResult{
                conn: &webSocketStompConnection{
                    wsCon: conn,
                },
            }
        }
    })

    var err error
    l.tcpConnectionListener, err = net.Listen("tcp", addr)
    if err != nil {
        return nil, err
    }

    go l.httpServer.Serve(l.tcpConnectionListener)
    return l, nil
}

func (l *webSocketConnectionListener) checkOrigin(r *http.Request) bool {
    if len(l.allowedOrigins) == 0 {
        return true
    }

    origin := r.Header["Origin"]
    if len(origin) == 0 {
        return true
    }
    u, err := url.Parse(origin[0])
    if err != nil {
        return false
    }
    if strings.ToLower(u.Host) == strings.ToLower(r.Host) {
        return true
    }

    for _, allowedOrigin := range l.allowedOrigins {
        if strings.ToLower(u.Host) == strings.ToLower(allowedOrigin) {
            return true
        }
    }

    return false
}

func (l *webSocketConnectionListener) Accept() (RawConnection, error) {
    cr := <- l.connectionsChannel
    return cr.conn, cr.err
}

func (l *webSocketConnectionListener) Close() error {
    return l.httpServer.Close()
}
