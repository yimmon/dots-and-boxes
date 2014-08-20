/*********************************************************************************
*     File Name           :     battle.go
*     Created By          :     YIMMON, yimmon.zhuang@gmail.com
*     Creation Date       :     [2014-04-22 13:15]
*     Last Modified       :     [2014-05-30 11:38]
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
    "flag"
    "fmt"
    "math/rand"
    "os"
    "runtime"
    "time"
)

var (
    n         int = 0
    timeout   int = 500
    agent     [2]interface{}
    winstat   [2]int
    nullfile  *os.File
    numThread int = 1
    nochange  bool
    ch        = make(chan int, 128)
)

func prepare() {
    if len(os.Args) < 3 {
        fmt.Println("Usage: battle -a algorithm1 -b algorithm2 -n number -timeout millisecond [-nochange]")
        os.Exit(0)
    }

    var alname [2]string
    flag.StringVar(&alname[0], "a", "alphabeta", "Algorithm 1")
    flag.StringVar(&alname[1], "b", "uct", "Algorithm 2")
    flag.IntVar(&n, "n", 0, "The number of battles.")
    flag.IntVar(&timeout, "timeout", 500, "Timeout of each step.")
    flag.BoolVar(&nochange, "nochange", false, "Not change.")
    flag.Parse()

    if n <= 0 {
        fmt.Printf("n cannot be %d\n", n)
        os.Exit(0)
    }
    for i, name := range alname {
        switch name {
        case "alphabeta":
            agent[i] = new(alphabeta.AlphaBeta)
        case "uct":
            agent[i] = new(uct.UCT)
        case "uctann":
            agent[i] = new(uctann.UCTANN)
        case "quct":
            agent[i] = new(quct.QUCT)
        case "quctann":
            agent[i] = new(quctann.QUCTANN)
        default:
            fmt.Println("Unknow algorithm name.")
            os.Exit(0)
        }
    }
}

func Battle(n int) {
    var name [2]string
    winstatp := [2]*int{&winstat[0], &winstat[1]}
    for ag := agent; n > 0; n-- {
        b := board.NewBoard(0, 0, 0, 0, 0, 0)
        t := uint(timeout)

        var h, v int32
        for b.IsEnd() == 0 {
            //tm, now := time.Now(), b.Now
            if b.Turn >= 26 {
                t = 1000
            }
            if age, ok := ag[b.Now].(board.IAlgorithm); ok {
                h, v, _ = age.MakeMove(b, t, true)
                name[b.Now] = age.GetName()
            } else if age, ok := ag[b.Now].(qboard.IAlgorithm); ok {
                qb := qboard.NewQBoard(b.H, b.V, int(b.S[0]), int(b.S[1]), int(b.Now), int(b.Turn))
                h, v, _ = age.MakeMove(qb, t, true)
                name[b.Now] = age.GetName()
            } else {
                panic("error")
            }
            if _, err := b.MoveHV(h, v); err != nil {
                panic(err)
            }
            //fmt.Println(b.Draw(), name[now], time.Since(tm), "Turn:", b.Turn, name[b.Now], "\n")
        }

        win := 0
        if b.IsEnd() > 0 {
            win = 1
        }
        (*winstatp[win])++
        fmt.Printf("<%d> Win:%d %s vs %s\n", *winstatp[0]+*winstatp[1], win, name[0], name[1])
        if !nochange {
            ag[0], ag[1] = ag[1], ag[0]
            winstatp[0], winstatp[1] = winstatp[1], winstatp[0]
        }
        runtime.GC()
    }
    ch <- 1
}

func init() {
    rand.Seed(time.Now().Unix())
    numThread = runtime.NumCPU()
    runtime.GOMAXPROCS(numThread)
    //nullfile, _ = os.Create("/dev/null")
    //log.SetOutput(nullfile)
}

func main() {
    prepare()
    winstat[0], winstat[1] = 0, 0

    /*
       for i := 1; i < numThread; i++ {
           go Battle(n / numThread)
       }
       Battle(n/numThread + n%numThread)
    */
    Battle(n)

    /*
       for i := 0; i < numThread; i++ {
           <-ch
       }
    */

    var name [2]string
    for i := 0; i < 2; i++ {
        if ag, ok := agent[i].(board.IAlgorithm); ok {
            name[i] = ag.GetName()
        } else if ag, ok := agent[i].(qboard.IAlgorithm); ok {
            name[i] = ag.GetName()
        }
    }

    fmt.Printf("%s vs %s\t%d:%d\t%.2f%%:%.2f%%\n",
        name[0], name[1], winstat[0], winstat[1],
        float32(winstat[0]*100)/float32(winstat[0]+winstat[1]),
        float32(winstat[1]*100)/float32(winstat[0]+winstat[1]))
}
