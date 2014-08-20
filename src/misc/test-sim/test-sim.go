/*********************************************************************************
*     File Name           :     test-sim.go
*     Created By          :     YIMMON, yimmon.zhuang@gmail.com
*     Creation Date       :     [2014-05-22 15:35]
*     Last Modified       :     [2014-06-06 19:11]
*     Description         :
**********************************************************************************/

package main

import (
    "algorithm/qboard"
    "ann"
    "ann/qeval"
    "ann/qrcds"
    "flag"
    "fmt"
    "path"
)

func main() {
    var (
        pturn                  = flag.Int("turn", 28, "Turn")
        correct, incorrect int = 0, 0
        anns                   = qeval.GetAnnModels("./AnnModels")
        f                  qrcds.File
        tmpms, tmppms      [60]int
        tmpinput           [32]ann.Type
        hash               qrcds.Hash
    )
    flag.Parse()
    hash.Init(func(k *qrcds.HashKey, v *qrcds.HashValue) {})
    if err := f.Open(path.Join("./ExamSets", fmt.Sprintf("%02d.exam", *pturn))); err != nil {
        panic(err)
    }
    defer f.Close()
    for {
        if rcd, err := f.ReadOneRecord(); err != nil {
            break
        } else if rcd.Z > 0 {
            b := qboard.NewQBoard(rcd.H, rcd.V, int(rcd.S0), int(rcd.S1), 0, *pturn)
            ms := b.GetMove12(tmpms[:])
            if len(ms) < 10 {
                ms = append(ms, b.GetMove3(ms[len(ms):cap(ms)])...)
            }
            best, res, now := -1e100, -1, b.Now
            for _, m := range ms {
                b.Move(m)
                mm := b.Play(tmppms[:])
                input := qeval.NewAnnInputBoard(b, tmpinput[:])
                z := float64(anns.Models[b.Turn].Run(input)[0])
                if b.Now != now {
                    z = anns.Stats[b.Turn].Wrate + anns.Stats[b.Turn].Lrate - z
                }
                if z > best || (z == best && b.Now == now) {
                    best, res = z, m
                }
                b.UnMove(mm...)
                b.UnMove(m)
            }
            b.Move(res)
            b.Play(tmppms[:])
            w := qeval.WhoWillWin(b, &hash, make(chan int))
            if w == 0 {
                correct++
            } else {
                incorrect++
            }
        }
    }
    sum := correct + incorrect
    fmt.Printf("Turn: %d, Correct: %d(%.2f%%), Incorrect: %d(%.2f%%)\n", *pturn, correct, float64(correct)*100/float64(sum),
        incorrect, float64(incorrect)*100/float64(sum))
}
