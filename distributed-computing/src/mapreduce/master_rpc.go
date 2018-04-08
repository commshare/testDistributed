package mapreduce

import (
	"fmt"
	"log"
	"net"
	"net/rpc"
	"os"
)

/*这是用来关闭master的rpc服务器的*/
// Shutdown is an RPC method that shuts down the Master's RPC server.
func (mr *Master) Shutdown(_, _ *struct{}) error {
	debug("Shutdown: registration server\n")
	/*关闭一个channel*/
	close(mr.shutdown)
	mr.l.Close() // causes the Accept to fail
	return nil
}

/*只要worker活着，就能收到worker的rpc请求*/
// startRPCServer staarts the Master's RPC server. It continues accepting RPC
// calls (Register in particular) for as long as the worker is alive.
func (mr *Master) startRPCServer() {
	rpcs := rpc.NewServer()
	rpcs.Register(mr)
	os.Remove(mr.address) // only needed for "unix"
	l, e := net.Listen("unix", mr.address)
	if e != nil {
		log.Fatal("RegstrationServer", mr.address, " error: ", e)
	}
	mr.l = l

	// now that we are listening on the master address, can fork off
	// accepting connections to another thread.
	go func() {
	loop:
		for {
			select {
			case <-mr.shutdown: /*收到信号，close了也是信号*/
				break loop
			default:
			}
			conn, err := mr.l.Accept()
			/*使用rpc处理一个tcp连接*/
			if err == nil {
				go func() {
					rpcs.ServeConn(conn)
					conn.Close()
				}()
			} else {
				debug("RegistrationServer: accept error", err)
				break
			}
		}
		debug("RegistrationServer: done\n")
	}()
}
/*
关闭master的rcp 服务器
这个必须同一个rpc做，这样可以避免在rpc server 线程和当前线程的race 情况！！！！ TODO
*/
// stopRPCServer stops the master RPC server.
// This must be done through an RPC to avoid race conditions between the RPC
// server thread and the current thread.
func (mr *Master) stopRPCServer() {
	var reply ShutdownReply
	ok := call(mr.address, "Master.Shutdown", new(struct{}), &reply)
	if ok == false {
		fmt.Printf("Cleanup: RPC %s error\n", mr.address)
	}
	debug("cleanupRegistration: done\n")
}
