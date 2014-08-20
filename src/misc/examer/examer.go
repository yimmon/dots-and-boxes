/*********************************************************************************
*     File Name           :     examer.go
*     Created By          :     YIMMON, yimmon.zhuang@gmail.com
*     Creation Date       :     [2014-05-05 14:48]
*     Last Modified       :     [2014-05-12 20:58]
*     Description         :
**********************************************************************************/

package main

import (
    "algorithm/board"
    "ann/eval"
    "ann/rcds"
    "flag"
    "fmt"
    "log"
    "math/rand"
    "os"
    "path"
    "runtime"
    "sync/atomic"
    "time"
)

var (
    numThread   int = 1
    pturn           = flag.Int("turn", -1, "Turn")
    pn              = flag.Int("n", 0, "N")
    phelp           = flag.Bool("help", false, "Help")
    HashManager rcds.Hash
)

func main() {
    dir := "./ExamSets"
    os.MkdirAll(dir, os.ModePerm)
    file := new(rcds.File)
    if err := file.Create(path.Join(dir, fmt.Sprintf("%02d.exam", *pturn))); err != nil {
        panic(err)
    }
    defer file.Close()
    fmt.Printf("Create %s\n", path.Join(dir, fmt.Sprintf("%02d.exam", *pturn)))

    ch := make(chan int, numThread)
    var count int32 = 0
    for t := 0; t < numThread; t++ {
        go func() {
            for {
                cnt := int(atomic.AddInt32(&count, 1))
                if cnt > *pn {
                    break
                }
                var b *board.Board
                var lms []*board.Moves
                for {
                    b, lms = board.GetBoard(*pturn)
                    if b.IsEnd() != 0 {
                        for i := len(lms) - 1; i >= 0; i-- {
                            b.UnMove(lms[i])
                        }
                        if b.IsEnd() != 0 {
                            panic("b is end.")
                        }
                    }
                    if int(b.Turn) == *pturn {
                        break
                    }
                }
                var z int8 = 0
                stop := make(chan int, 1)
                go func() {
                    time.Sleep(5 * time.Minute)
                    stop <- 1
                }()
                if r := eval.WhoWillWin(b, &HashManager, 99, nil, stop); r == b.Now {
                    z = 1
                } else if r == b.Now^1 {
                    z = -1
                } else {
                    fmt.Printf("%dth timeout.\n", cnt)
                }
                if z == 1 || z == -1 {
                    rcd := &rcds.Record{H: b.H, V: b.V, S0: b.S[b.Now], S1: b.S[b.Now^1], Z: int8(z)}
                    file.WriteOneRecord(rcd)
                    fmt.Printf("%dth finished.\n", cnt)
                }
            }
            ch <- 1
        }()
    }

    for i := 0; i < numThread; i++ {
        <-ch
    }
    fmt.Println("\nDone.")
}

func init() {
    rand.Seed(time.Now().Unix())
    numThread = runtime.NumCPU() + 1
    runtime.GOMAXPROCS(numThread)
    flag.Parse()
    if *phelp {
        fmt.Println("Usage: examer --turn Num -n Num")
        os.Exit(0)
    }
    HashManager.Init(func(k *rcds.HashKey, v *rcds.HashValue) {})
    nullfile, _ := os.Create("/dev/null")
    log.SetOutput(nullfile)
}
