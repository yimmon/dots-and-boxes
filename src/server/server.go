/*********************************************************************************
*     File Name           :     server.go
*     Created By          :     YIMMON, yimmon.zhuang@gmail.com
*     Creation Date       :     [2014-04-09 20:42]
*     Last Modified       :     [2014-08-16 11:01]
*     Description         :
**********************************************************************************/

package main

import (
	"algorithm/alphabeta"
	"algorithm/board"
	"algorithm/qboard"
	"algorithm/quct"
	"algorithm/quctann"
	"algorithm/uct"
	"algorithm/uctann"
	"errors"
	"log"
	"math/rand"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"runtime"
	"time"
)

const (
	Address string = "0.0.0.0:12345"
)

type Server int

type MoveArg struct {
	Algorithm string
	Board     board.Board
	Timeout   uint
}

type MoveResult struct {
	H, V int32
}

func (self *Server) MakeMove(arg *MoveArg, result *MoveResult) (err error) {
	var agent board.IAlgorithm
	var qagent qboard.IAlgorithm

	switch arg.Algorithm {
	case "alphabeta":
		agent = new(alphabeta.AlphaBeta)
	case "uct":
		agent = new(uct.UCT)
	case "uctann":
		agent = new(uctann.UCTANN)
	case "quct":
		qagent = new(quct.QUCT)
	case "quctann":
		qagent = new(quctann.QUCTANN)
	default:
		err = errors.New("Unknown algorithm name.")
		return
	}

	if arg.Algorithm[0] == 'q' {
		b := qboard.NewQBoard(arg.Board.H, arg.Board.V, int(arg.Board.S[0]), int(arg.Board.S[1]), int(arg.Board.Now), int(arg.Board.Turn))
		result.H, result.V, err = qagent.MakeMove(b, arg.Timeout, true)
	} else {
		b := board.NewBoard(arg.Board.H, arg.Board.V, arg.Board.S[0], arg.Board.S[1], arg.Board.Now, arg.Board.Turn)
		result.H, result.V, err = agent.MakeMove(b, arg.Timeout, true)
	}
	//log.Println("Receive:", arg.Board.H, arg.Board.V, arg.Board.S[0], arg.Board.S[1], arg.Board.Now)
	//log.Println("Send:", result.H, result.V)
	return
}

func init() {
	rand.Seed(time.Now().Unix())
	runtime.GOMAXPROCS(runtime.NumCPU() + 1)
}

func main() {
	server := rpc.NewServer()
	server.Register(new(Server))
	server.HandleHTTP(rpc.DefaultRPCPath, rpc.DefaultDebugPath)

	l, err := net.Listen("tcp", Address)
	if err != nil {
		log.Fatal("Listen error:", err)
	}
	log.Println("Server is running at", Address)

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		log.Println("Accept new connection", conn.RemoteAddr())
		go server.ServeCodec(jsonrpc.NewServerCodec(conn))
	}
}
