package part

import (
	"net"
)

type port struct {}

var (
	port_map map[string]int = make(map[string]int)
	port_buf chan bool = make(chan bool,1)
)

func Port() (*port) {
	return &port{}
}

func (p *port) Get(key string) int {
	if p,ok := port_map[key]; ok {return p} 
	return p.New(key)
}

func (*port) Del(key string) {
	delete(port_map,key)
}

func (*port) Set(key string,l net.Listener) int {
	port_buf<-true
	defer func(){
		<-port_buf
	}()
	port_map[key] = l.Addr().(*net.TCPAddr).Port
	return l.Addr().(*net.TCPAddr).Port
}

func (*port) New(key string) int {
	port_buf<-true
	defer func(){
		<-port_buf
	}()
	Logf().I("New port with key:",key)
	if p := Sys().GetFreePort();p != 0{
		port_map[key] = p
		return p
	}

	Logf().E("cant get free port with key:",key)
	return 0
}