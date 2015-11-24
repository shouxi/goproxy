package main

import (
	//	"bytes"
	//	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
)

func recv(buf []byte, m int, conn net.Conn) (n int, err error) {
	for nn := 0; n < m; {
		nn, err = conn.Read(buf[n:m])
		if nil != err && io.EOF != err {
			log.Println("err:", err)
			panic(err)
			return
		}
		n += nn
	}
	return
}

type reqHello struct {
	ver      uint8
	nmethods uint8
	methods  [255]uint8
}

func (msg *reqHello) read(conn net.Conn) (err error) {
	_, err = recv(msg.methods[:2], 2, conn)
	if nil != err {
		return
	}
	msg.ver, msg.nmethods = msg.methods[0], msg.methods[1]
	_, err = recv(msg.methods[:], int(msg.nmethods), conn)
	if nil != err {
		return
	}
	return
}
func (msg *reqHello) print() {
	log.Println("************")
	log.Println("get reqHello msg:")
	log.Println("ver:", msg.ver, " nmethods:", msg.nmethods, " methods:", msg.methods[:msg.nmethods])
	log.Println("************")
}

type ansEcho struct {
	ver    uint8
	method uint8
	buf    [2]uint8
}

func (msg *ansEcho) gen(t uint8) {
	msg.ver, msg.method = 5, t
	msg.buf[0], msg.buf[1] = 5, t
}
func (msg *ansEcho) write(conn net.Conn) {
	conn.Write(msg.buf[:])
}
func (msg *ansEcho) print() {
	log.Println("------------------")
	log.Println("send ansEcho msg:")
	log.Println("ver:", msg.ver, " method:", msg.method)
	log.Println("------------------")
}

type reqMsg struct {
	ver       uint8     // socks v5: 0x05
	cmd       uint8     // CONNECT: 0x01, BIND:0x02, UDP ASSOCIATE: 0x03
	rsv       uint8     //RESERVED
	atyp      uint8     //IP V4 addr: 0x01, DOMANNAME: 0x03, IP V6 addr: 0x04
	dst_addr  [255]byte //
	dst_port  [2]uint8  //
	dst_port2 uint16    //

	reqtype string
	url     string
}

func (msg *reqMsg) read(conn net.Conn) (err error) {
	buf := make([]byte, 4)
	_, err = recv(buf, 4, conn)
	if nil != err {
		return
	}

	msg.ver, msg.cmd, msg.rsv, msg.atyp = buf[0], buf[1], buf[2], buf[3]

	if 5 != msg.ver || 0 != msg.rsv {
		log.Println("Request Message VER or RSV error!")
		return
	}
	switch msg.atyp {
	case 1: //ip v4
		_, err = recv(msg.dst_addr[:], 4, conn)
	case 4:
		_, err = recv(msg.dst_addr[:], 16, conn)
	case 3:
		_, err = recv(msg.dst_addr[:1], 1, conn)
		_, err = recv(msg.dst_addr[1:], int(msg.dst_addr[0]), conn)
	}
	if nil != err {
		return
	}
	_, err = recv(msg.dst_port[:], 2, conn)
	if nil != err {
		return
	}
	//bbuf := bytes.NewBuffer(msg.dst_port[:])
	//err = binary.Read(bbuf, binary.BigEndian, msg.dst_port2)
	//if nil != err {
	//	log.Println(err)
	//	return
	//}
	msg.dst_port2 = (uint16(msg.dst_port[0]) << 8) + uint16(msg.dst_port[1])

	switch msg.cmd {
	case 1:
		msg.reqtype = "tcp"
	case 2:
		log.Println("BIND")
	case 3:
		msg.reqtype = "udp"
	}
	switch msg.atyp {
	case 1: // ipv4
		msg.url = fmt.Sprintf("%d.%d.%d.%d:%d", msg.dst_addr[0], msg.dst_addr[1], msg.dst_addr[2], msg.dst_addr[3], msg.dst_port2)
	case 3: //DOMANNAME
		msg.url = string(msg.dst_addr[1 : 1+msg.dst_addr[0]])
		msg.url += fmt.Sprintf(":%d", msg.dst_port2)
	case 4: //ipv6
		log.Println("IPV6")
	}
	return
}
func (msg *reqMsg) print() {
	log.Println("---***-----****----***---")
	log.Println("get reqMsg:")
	log.Println("ver:", msg.ver, " cmd:", msg.cmd, " rsv:", msg.rsv, " atyp", msg.atyp, " dst_addr:", msg.url)
	log.Println("---***-----****----***---")
}

type ansMsg struct {
	ver  uint8
	rep  uint8
	rsv  uint8
	atyp uint8
	//bnd_addr [255]uint8
	//bnd_port [2]uint8
	buf  [300]uint8
	mlen uint16
}

func (msg *ansMsg) gen(req *reqMsg, rep uint8) {
	msg.ver = 5
	msg.rep = rep //rfc1928
	msg.rsv = 0
	msg.atyp = 1 //req.atyp

	msg.buf[0], msg.buf[1], msg.buf[2], msg.buf[3] = msg.ver, msg.rep, msg.rsv, msg.atyp
	for i := 5; i < 11; i++ {
		msg.buf[i] = 0
	}
	msg.mlen = 10
	//i := 4
	//for ; i+4 <= int(req.dst_addr[0]); i++ {
	//	msg.buf[i] = req.dst_addr[i-4]
	//}
	//msg.buf[i], msg.buf[i+1] = req.dst_port[0], req.dst_port[1]
	//msg.mlen = uint16(i + 2)
}
func (msg *ansMsg) write(conn net.Conn) {
	conn.Write(msg.buf[:msg.mlen])
}
func (msg *ansMsg) print() {
	log.Println("***-----****----***---***")
	log.Println("send ans Msg:")
	log.Println(msg.buf[:msg.mlen])
	log.Println("***-----****----***---***")
}

func handleConn(conn net.Conn) {
	//defer conn.Close()
	log.Println("remote addr:", conn.RemoteAddr())

	var reqhello reqHello
	var ansecho ansEcho
	var reqmsg reqMsg
	var ansmsg ansMsg

	//recv hello
	var err error = nil
	err = reqhello.read(conn)
	if nil != err {
		return
	}
	reqhello.print()

	//send echo
	ansecho.gen(0)
	ansecho.write(conn)
	ansecho.print()

	//recv request
	err = reqmsg.read(conn)
	if nil != err {
		return
	}
	reqmsg.print()
	//connect
	var pconn net.Conn
	pconn, err = net.Dial(reqmsg.reqtype, reqmsg.url)
	//defer pconn.Close()

	//reply
	//error occur
	if nil != err {
		ansmsg.gen(&reqmsg, 4)
		ansmsg.write(conn)
		ansmsg.print()
		return
	}
	//success
	ansmsg.gen(&reqmsg, 0)
	ansmsg.write(conn)
	ansmsg.print()
	pipe(conn, pconn)
}

func resend(in net.Conn, out net.Conn) {
	buf := make([]byte, 10240)
	for {
		n, err := in.Read(buf)
		if io.EOF == err {
			log.Printf("io.EOF")
			return
		} else if nil != err {
			log.Printf("resend err\n", err)
			return
		}
		out.Write(buf[:n])
	}
}

func pipe(a net.Conn, b net.Conn) {
	go resend(a, b)
	go resend(b, a)
}

func socks5proxy() {
	ln, err := net.Listen("tcp", ":8000")
	if nil != err {
		fmt.Println("Bind Error!")
		return
	}

	for {
		conn, err := ln.Accept()
		if nil != err {
			fmt.Println("Accept Error!")
			continue
		}

		go handleConn(conn)
	}
}

func main() {
	log.Println()
	log.Println("----------------------starting---------------")
	socks5proxy()

}
