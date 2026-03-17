// 答案版本：在 TCP 之上的自定义应用层协议完整实现
//
// 协议格式（12 字节定长头部 + 变长 Body）：
//
//  +---------+---------+---------+---------+
//  | Magic   | Version |  Type   | Reserved|
//  | 2 bytes | 1 byte  | 1 byte  | 1 byte  |
//  +---------+---------+---------+---------+
//  | Seq             (4 bytes, big-endian) |
//  +---------------------------------------+
//  | BodyLen         (4 bytes, big-endian) |
//  +---------------------------------------+
//  | Body            (N bytes)             |
//  +---------------------------------------+
//
// 运行方式:
//   go run main.go -mode server
//   go run main.go -mode client -msg "hello"
package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"sync/atomic"
)

// -------- 协议常量 --------

const (
	Magic   uint16 = 0xCAFE
	Version uint8  = 0x01

	HeaderLen = 12 // 固定头部长度（字节）

	TypePing uint8 = 0x01
	TypePong uint8 = 0x02
	TypeData uint8 = 0x03
	TypeErr  uint8 = 0x04
)

// -------- 数据结构 --------

// Header 是协议固定头部。
type Header struct {
	Magic    uint16
	Version  uint8
	Type     uint8
	Reserved uint8
	Seq      uint32
	BodyLen  uint32
}

// Message 是一个完整的协议消息。
type Message struct {
	Header
	Body []byte
}

// -------- 编解码 --------

// Encode 将 Message 序列化为字节流（大端序）。
//
// 实现要点：
//  1. 先把 Header 各字段按大端序写入 buf
//  2. 追加 Body
//  3. 返回完整字节切片
func Encode(msg *Message) ([]byte, error) {
	msg.Magic = Magic
	msg.Version = Version
	msg.BodyLen = uint32(len(msg.Body))

	buf := new(bytes.Buffer)

	// 写头部
	if err := binary.Write(buf, binary.BigEndian, msg.Header); err != nil {
		return nil, fmt.Errorf("encode header: %w", err)
	}
	// 写 Body
	if _, err := buf.Write(msg.Body); err != nil {
		return nil, fmt.Errorf("encode body: %w", err)
	}
	return buf.Bytes(), nil
}

// Decode 从 reader 中读取并反序列化一个 Message。
//
// 实现要点（解决 TCP 粘包 / 拆包）：
//  1. io.ReadFull 读取固定 HeaderLen 字节 → 解析 Header
//  2. 校验 Magic / Version
//  3. 按 Header.BodyLen 再 io.ReadFull 读 Body
func Decode(r io.Reader) (*Message, error) {
	// 1. 读固定长度头部
	headerBuf := make([]byte, HeaderLen)
	if _, err := io.ReadFull(r, headerBuf); err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}

	// 2. 反序列化头部
	var hdr Header
	if err := binary.Read(bytes.NewReader(headerBuf), binary.BigEndian, &hdr); err != nil {
		return nil, fmt.Errorf("decode header: %w", err)
	}

	// 3. 校验魔数和版本
	if hdr.Magic != Magic {
		return nil, fmt.Errorf("invalid magic: 0x%X", hdr.Magic)
	}
	if hdr.Version != Version {
		return nil, fmt.Errorf("unsupported version: %d", hdr.Version)
	}

	// 4. 按 BodyLen 读 Body（精确读取，避免粘包）
	body := make([]byte, hdr.BodyLen)
	if hdr.BodyLen > 0 {
		if _, err := io.ReadFull(r, body); err != nil {
			return nil, fmt.Errorf("read body: %w", err)
		}
	}

	return &Message{Header: hdr, Body: body}, nil
}

// -------- 服务端 --------

// StartServer 在 addr 上监听 TCP 连接，对每条连接起一个 goroutine 处理。
func StartServer(addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Fprintln(os.Stderr, "accept error:", err)
			continue
		}
		go handleConn(conn)
	}
}

// handleConn 处理单个客户端连接：循环读取消息 → 分发 → 写回响应。
//
// 使用 bufio.Reader 减少系统调用次数（批量读取）。
func handleConn(conn net.Conn) {
	defer conn.Close()

	remote := conn.RemoteAddr()
	fmt.Printf("[server] new conn from %s\n", remote)
	defer fmt.Printf("[server] conn closed: %s\n", remote)

	reader := bufio.NewReader(conn)
	for {
		req, err := Decode(reader)
		if err != nil {
			if err != io.EOF {
				fmt.Fprintf(os.Stderr, "[server] decode error from %s: %v\n", remote, err)
			}
			return
		}
		fmt.Printf("[server] recv seq=%d type=0x%02X body=%q\n", req.Seq, req.Type, req.Body)

		resp := processMessage(req)
		data, err := Encode(resp)
		if err != nil {
			fmt.Fprintln(os.Stderr, "[server] encode error:", err)
			return
		}
		if _, err := conn.Write(data); err != nil {
			fmt.Fprintln(os.Stderr, "[server] write error:", err)
			return
		}
	}
}

// processMessage 根据消息类型返回响应消息。
//   - TypePing → TypePong（空 Body）
//   - TypeData → TypeData（Echo Body）
//   - 其他     → TypeErr（Body="unknown type"）
func processMessage(req *Message) *Message {
	resp := &Message{}
	resp.Seq = req.Seq // 保持序列号，方便客户端匹配

	switch req.Type {
	case TypePing:
		resp.Type = TypePong
	case TypeData:
		resp.Type = TypeData
		resp.Body = append([]byte(nil), req.Body...) // echo
	default:
		resp.Type = TypeErr
		resp.Body = []byte("unknown type")
	}
	return resp
}

// -------- 客户端 --------

// globalSeq 用于生成单调递增的序列号。
var globalSeq uint32

// SendMessage 连接 addr，发送一条 TypeData 消息，等待响应后打印并退出。
func SendMessage(addr, text string) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	defer conn.Close()

	seq := atomic.AddUint32(&globalSeq, 1)
	req := &Message{
		Header: Header{Type: TypeData, Seq: seq},
		Body:   []byte(text),
	}

	data, err := Encode(req)
	if err != nil {
		return fmt.Errorf("encode: %w", err)
	}
	if _, err := conn.Write(data); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	fmt.Printf("[client] sent seq=%d body=%q\n", seq, text)

	// 等待服务端响应
	reader := bufio.NewReader(conn)
	resp, err := Decode(reader)
	if err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	fmt.Printf("[client] recv seq=%d type=0x%02X body=%q\n", resp.Seq, resp.Type, resp.Body)
	return nil
}

// -------- main --------

func main() {
	mode := flag.String("mode", "server", "server | client")
	addr := flag.String("addr", "127.0.0.1:9000", "listen/connect address")
	msg := flag.String("msg", "hello", "message to send (client mode)")
	flag.Parse()

	var err error
	switch *mode {
	case "server":
		fmt.Printf("server listening on %s\n", *addr)
		err = StartServer(*addr)
	case "client":
		err = SendMessage(*addr, *msg)
	default:
		fmt.Fprintln(os.Stderr, "unknown mode:", *mode)
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
