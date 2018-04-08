package main
import (
	"fmt"
	"net/rpc"
	"encoding/gob"
	"bufio"
	"time"
	"net"
	"io"
	"log"
)



type gobClientCodec struct {
	rwc    io.ReadWriteCloser
	dec    *gob.Decoder
	enc    *gob.Encoder
	encBuf *bufio.Writer
}

func (c *gobClientCodec) WriteRequest(r *rpc.Request, body interface{}) (err error) {
	if err = TimeoutCoder(c.enc.Encode, r, "client write request"); err != nil {
		return
	}
	if err = TimeoutCoder(c.enc.Encode, body, "client write request body"); err != nil {
		return
	}
	return c.encBuf.Flush()
}

func (c *gobClientCodec) ReadResponseHeader(r *rpc.Response) error {
	return c.dec.Decode(r)
}

func (c *gobClientCodec) ReadResponseBody(body interface{}) error {
	return c.dec.Decode(body)
}

func (c *gobClientCodec) Close() error {
	return c.rwc.Close()
}

func callclient(srv string, rpcname string, args interface{}, reply interface{}) error {
	conn, err := net.DialTimeout("tcp", srv+":4200", time.Second*10)
	if err != nil {
		return fmt.Errorf("ConnectError: %s", err.Error())
	}
	encBuf := bufio.NewWriter(conn)
	/*对消息编码*/
	codec := &gobClientCodec{conn, gob.NewDecoder(conn), gob.NewEncoder(encBuf), encBuf}
	/*创建了一个rpc客户端*/
	c := rpc.NewClientWithCodec(codec)
	/*rpc客户端可以执行远程调用，这个是要求服务器做应答的吧*/
	err = c.Call(rpcname, args, reply)
	/*panic: interface conversion: interface {} is *string, not string,使用了*(reply.(*string)) 才把内容取出来*/
	log.Println("call replay ",*(reply.(*string)))
	/*
	// Close calls the underlying codec's Close method. If the connection is already
// shutting down, ErrShutdown is returned.
	*/
	errc := c.Close()
	if err != nil && errc != nil {
		return fmt.Errorf("%s %s", err, errc)
	}
	if err != nil {
		return err
	} else {
		return errc
	}
}
