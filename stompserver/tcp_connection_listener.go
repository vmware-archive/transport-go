// Copyright 2019 VMware, Inc. All rights reserved. -- VMware Confidential

package stompserver

import (
    "time"
    "net"
    "github.com/go-stomp/stomp/frame"
)

type tcpStompConnection struct {
    tcpCon net.Conn
}

func (c *tcpStompConnection) ReadFrame() (*frame.Frame, error) {
    frameR := frame.NewReader(c.tcpCon)
    f, e := frameR.Read()
    return f,e
}

func (c *tcpStompConnection) WriteFrame(f *frame.Frame) error {
    frameWr := frame.NewWriter(c.tcpCon)
    err := frameWr.Write(f)
    return err
}

func (c *tcpStompConnection) SetReadDeadline(t time.Time) {
    c.tcpCon.SetReadDeadline(t)
}

func (c *tcpStompConnection) Close() error {
    return c.tcpCon.Close()
}

type tcpConnectionListener struct {
    listener net.Listener
}

func NewTcpConnectionListener(addr string) (RawConnectionListener, error) {
    tcpListener, err := net.Listen("tcp", addr)
    if err != nil {
        return nil, err
    }
    return &tcpConnectionListener{listener: tcpListener}, nil
}

func (l *tcpConnectionListener) Accept() (RawConnection, error) {
    conn, err := l.listener.Accept()

    if err != nil {
        return nil, err
    }

    return &tcpStompConnection{ tcpCon: conn }, nil
}

func (l *tcpConnectionListener) Close() error {
    return l.listener.Close()
}
