package main
import (
	"time"
	"fmt"
	"net/rpc"
	"log"
	"bufio"
	"encoding/gob"
	"io"
	"net"
	"strconv"
)


/*
https://github.com/daizuozhuo/rpc-example/blob/master/server.go
*/

type Worker struct {
	Name string
}

func NewWorker() *Worker {
	return &Worker{"test"}
}

/*果然是DoJob*/
func (w *Worker) DoJob(task string, reply *string) error {
	log.Println("Worker: DoJob", task)
	//time.Sleep(time.Second * 3)
	*reply = "OK"
	return nil
}

func TimeoutCoder(f func(interface{}) error, e interface{}, msg string) error {
	echan := make(chan error, 1)
	go func() { echan <- f(e) }() /*需要f来做处理	*/
	select {
	case e := <-echan:
		return e
	case <-time.After(time.Minute):
		return fmt.Errorf("Timeout %s", msg)
	}
}

type gobServerCodec struct {
	rwc    io.ReadWriteCloser
	dec    *gob.Decoder
	enc    *gob.Encoder
	encBuf *bufio.Writer
	closed bool
}

func (c *gobServerCodec) ReadRequestHeader(r *rpc.Request) error {
	return TimeoutCoder(c.dec.Decode, r, "server read request header")
}

func (c *gobServerCodec) ReadRequestBody(body interface{}) error {
	return TimeoutCoder(c.dec.Decode, body, "server read request body")
}

func (c *gobServerCodec) WriteResponse(r *rpc.Response, body interface{}) (err error) {
	if err = TimeoutCoder(c.enc.Encode, r, "server write response"); err != nil {
		if c.encBuf.Flush() == nil {
			log.Println("rpc: gob error encoding response:", err)
			c.Close()
		}
		return
	}
	if err = TimeoutCoder(c.enc.Encode, body, "server write response body"); err != nil {
		if c.encBuf.Flush() == nil {
			log.Println("rpc: gob error encoding body:", err)
			c.Close()
		}
		return
	}
	return c.encBuf.Flush()
}

func (c *gobServerCodec) Close() error {
	if c.closed {
		// Only call c.rwc.Close once; otherwise the semantics are undefined.
		return nil
	}
	c.closed = true
	return c.rwc.Close()
}

func ListenRPC() {
	rpc.Register(NewWorker())
	l, e := net.Listen("tcp", ":4200")
	if e != nil {
		log.Fatal("Error: listen 4200 error:", e)
	}
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				log.Print("Error: accept rpc connection", err.Error())
				continue
			}
			go func(conn net.Conn) {
				buf := bufio.NewWriter(conn)
				/*包装了连接，并对消息编解码*/
				srv := &gobServerCodec{
					rwc:    conn,
					dec:    gob.NewDecoder(conn),
					enc:    gob.NewEncoder(buf),
					encBuf: buf,
				}
				err = rpc.ServeRequest(srv)
				log.Println("...after ServeRequest...")
				if err != nil {
					log.Print("Error: server rpc request", err.Error())
				}
				/*自己实现的*/
				srv.Close()
			}(conn)
		}
	}()
}

func main() {
	go ListenRPC()
	N := 100
	mapChan := make(chan int, N)
	for i := 0; i < N; i++ {
		go func(i int) {
			/*客户端可以执行远程调用,服务端会执行DoJob*/
			callclient("localhost", "Worker.DoJob", strconv.Itoa(i), new(string))
			mapChan <- i
		}(i)
	}
	for i := 0; i<N; i++ {
		<-mapChan
	}


}
