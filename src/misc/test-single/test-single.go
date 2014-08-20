/*********************************************************************************
*     File Name           :     test-single.go
*     Created By          :     YIMMON, yimmon.zhuang@gmail.com
*     Creation Date       :     [2014-05-04 19:04]
*     Last Modified       :     [2014-06-06 16:17]
*     Description         :
**********************************************************************************/

package main

import (
    "algorithm/qboard"
    "ann"
    "ann/qeval"
    "ann/qrcds"
    "fmt"
    "sort"
)

type Output struct {
    str string
    z   float64
}

type ByZ []Output

func (self ByZ) Len() int           { return len(self) }
func (self ByZ) Swap(i, j int)      { self[i], self[j] = self[j], self[i] }
func (self ByZ) Less(i, j int) bool { return self[i].z > self[j].z }

func main() {
    for {
        var tmpms, tmppms [60]int
        var h, v, s0, s1 int32
        var count, incorrect, correct, unknow, turn int
        fmt.Scanf("0x%x 0x%x %d %d %d", &h, &v, &s0, &s1, &turn)
        b := qboard.NewQBoard(h, v, int(s0), int(s1), 0, int(turn))
        fmt.Print(b.Draw())
        c, p, z := Calc(b)
        fmt.Printf("(%d, %d, %f)\n\n", c, p, z)
        now := b.Now
        ms := b.GetMove12(tmpms[:])
        if len(ms) < 6 {
            ms = append(ms, b.GetMove3(ms[len(ms):cap(ms)])...)
        }

        output := make([]Output, len(ms))
        for i, m := range ms {
            ty := b.LinkType(m)
            b.Move(m)
            mm := b.Play(tmppms[:])
            c, p, z = Calc(b)
            if b.Now != now {
                z = anns.Stats[b.Turn].Wrate + anns.Stats[b.Turn].Lrate - z
            }
            output[i].str = fmt.Sprintf("(%v, %d, %d, %f, %d) %x\n", b.Now != now, c, p, z, ty, m)
            output[i].z = z
            if p == -1 {
                unknow++
            } else if c != p {
                incorrect++
            } else {
                correct++
            }
            count++
            b.UnMove(mm...)
            b.UnMove(m)
        }
        sort.Sort(ByZ(output))
        for _, v := range output {
            fmt.Print(v.str)
        }
        fmt.Printf("\ncorrect: %d, incorrect: %d, unknow: %d, count: %d\n", correct, incorrect, unknow, count)
    }
}

var anns = qeval.GetAnnModels("./AnnModels")
var hash = new(qrcds.Hash)

func init() {
    hash.Init(func(k *qrcds.HashKey, v *qrcds.HashValue) {})
}

func Calc(b *qboard.QBoard) (c, p int, z float64) {
    var tmpinput [32]ann.Type
    c = int(qeval.WhoWillWin(b, hash, make(chan int)))
    input := qeval.NewAnnInputBoard(b, tmpinput[:])
    z = float64(anns.Models[b.Turn].Run(input)[0])
    if z > anns.Stats[b.Turn].Wrate {
        p = int(b.Now)
    } else if z < anns.Stats[b.Turn].Lrate {
        p = int(b.Now ^ 1)
    } else {
        p = -1
    }
    return
}
