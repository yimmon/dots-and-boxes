/*********************************************************************************
*     File Name           :     test-qboard.go
*     Created By          :     YIMMON, yimmon.zhuang@gmail.com
*     Creation Date       :     [2014-05-19 17:26]
*     Last Modified       :     [2014-06-13 21:28]
*     Description         :
**********************************************************************************/

package main

import (
    "algorithm/qboard"
    "ann/qeval"
    "ann/qrcds"
    "flag"
    "fmt"
)

func main() {
    turn := flag.Int("turn", 32, "Turn")
    flag.Parse()
    var hash qrcds.Hash
    hash.Init(func(k *qrcds.HashKey, v *qrcds.HashValue) {})
    f := new(qrcds.File)
    f.Open(fmt.Sprintf("./DataSets/ExamSets/%02d.exam", *turn))
    defer f.Close()
    var i int
    var tmpms, tmppms [60]int
    for i = 1; i <= 1000; i++ {
        rcd, err := f.ReadOneRecord()
        if err != nil {
            break
        }
        fmt.Printf("No.%d:\n", i)
        b := qboard.NewQBoard(rcd.H, rcd.V, int(rcd.S0), int(rcd.S1), 0, *turn)
        ms := b.GetMove(tmpms[:])
        fmt.Println(ms)
        fmt.Println(b.Draw() + b.ShowLinks())
        w := qeval.WhoWillWin(b, &hash, make(chan int))
        if (rcd.Z > 0 && w != 0) || (rcd.Z < 0 && w != 1) {
            fmt.Println("Error")
            fmt.Println(b.Draw()+b.ShowLinks(), i, w)
            fmt.Println("\n\n")
            ms = b.GetMove(tmpms[:])
            fmt.Println(ms)
            for _, m := range ms {
                b.Move(m)
                mm := b.Play(tmppms[:])
                fmt.Println(b.Draw() + b.ShowLinks())
                if w = qeval.WhoWillWin(b, &hash, make(chan int)); w == 0 {
                    fmt.Println(b.Draw() + b.ShowLinks())
                    break
                } else {
                    fmt.Println(m, mm, w)
                }
                b.UnMove(mm...)
                b.UnMove(m)
            }
            panic(fmt.Sprintln(rcd))
        }
    }
    fmt.Println(i)
}
