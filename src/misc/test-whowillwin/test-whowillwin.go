/*********************************************************************************
*     File Name           :     test-whowillwin.go
*     Created By          :     YIMMON, yimmon.zhuang@gmail.com
*     Creation Date       :     [2014-05-07 12:47]
*     Last Modified       :     [2014-05-11 20:54]
*     Description         :
**********************************************************************************/

package main

import (
    "algorithm/board"
    "ann/eval"
    "ann/rcds"
    "flag"
    "fmt"
    "path"
)

var (
    pturn     = flag.Int("turn", -1, "Turn")
    pn        = flag.Int("n", 999999999, "N")
    pleafturn = flag.Int("leafturn", 29, "Leaf Turn")
    anns      *eval.Anns
    hash      rcds.Hash
)

func init() {
    flag.Parse()
    anns = eval.GetAnnModels("./AnnModels")
    hash.Init(func(k *rcds.HashKey, v *rcds.HashValue) {})
}

func main() {
    f := new(rcds.File)
    if err := f.Open(path.Join("./DataSets/ExamSets", fmt.Sprintf("%02d.exam", *pturn))); err != nil {
        panic(err)
    }
    defer f.Close()

    var sum, cor, incor, unknow int
    for sum < *pn {
        if rcd, err := f.ReadOneRecord(); err != nil {
            break
        } else {
            b := board.NewBoard(rcd.H, rcd.V, rcd.S0, rcd.S1, 0, int8(*pturn))
            z := eval.WhoWillWin(b, &hash, *pleafturn, anns, make(chan int, 1))
            sum++
            if (z == b.Now && rcd.Z > 0) || (z == b.Now^1 && rcd.Z < 0) {
                cor++
            } else if (z == b.Now^1 && rcd.Z > 0) || (z == b.Now && rcd.Z < 0) {
                incor++
            } else {
                unknow++
            }
        }
    }
    fmt.Printf("Sum: %d, Correct: %d, Incorrect: %d, Unknown: %d\n", sum, cor, incor, unknow)
    fmt.Printf("Correct: %.2f%%, Incorrect: %.2f%%, Unknown: %.2f%%\n", float32(cor)*100/float32(sum), float32(incor)*100/float32(sum),
        float32(unknow)*100/float32(sum))
}
