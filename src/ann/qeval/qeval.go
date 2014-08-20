/*********************************************************************************
*     File Name           :     qeval.go
*     Created By          :     YIMMON, yimmon.zhuang@gmail.com
*     Creation Date       :     [2014-05-18 21:32]
*     Last Modified       :     [2014-06-13 22:08]
*     Description         :
**********************************************************************************/

package qeval

import (
    "algorithm/qboard"
    "ann"
    "encoding/json"
    "fmt"
    "io/ioutil"
    "os"
    "path"
    "regexp"
)

type Anns struct {
    Models []*ann.Ann
    Stats  []*Stats
}

type Stats struct {
    Wrate, Lrate float64
    Pwcor, Plcor float64
}

func (self *Anns) WinOddBoard(b *qboard.QBoard) float64 {
    var (
        wrate, lrate float64 = 0.05, -0.55
        pwcor, plcor float64 = 1, 1
        tmpinput     [32]ann.Type
        turn         = b.Turn
    )
    if self.Stats[turn] != nil {
        wrate, lrate = self.Stats[turn].Wrate, self.Stats[turn].Lrate
        pwcor, plcor = self.Stats[turn].Pwcor, self.Stats[turn].Plcor
    }

    input := NewAnnInputBoard(b, tmpinput[:])
    z := float64(self.Models[turn].Run(input)[0])
    if z > wrate {
        return pwcor
    } else if z < lrate {
        return 1 - plcor
    }
    return -1
}

func (self *Anns) WinOdd(h, v int32, s0, s1, turn int) float64 {
    b := qboard.NewQBoard(h, v, s0, s1, 0, turn)
    return self.WinOddBoard(b)
}

func GetAnnModels(dir string) (ret *Anns) {
    ret = new(Anns)
    ret.Models = make([]*ann.Ann, 60)
    fis, err := ioutil.ReadDir(dir)
    if err != nil {
        panic(err)
    }
    var turn, count int = 0, 0
    for _, f := range fis {
        if b, _ := regexp.MatchString("\\d\\d\\.ann", f.Name()); b {
            fmt.Sscanf(f.Name(), "%d.ann", &turn)
            if count++; count == 1 {
                turn = 0
            }
            for i := turn; i < 60; i++ {
                if ret.Models[i] != nil {
                    ret.Models[i].Destroy()
                }
                ret.Models[i] = ann.CreateFromFile(path.Join(dir, f.Name()))
            }
        }
    }
    if count == 0 {
        panic("No ann models found.")
    }
    ret.GetAnnsStats(dir)
    return
}

func (self *Anns) GetAnnsStats(dir string) {
    self.Stats = make([]*Stats, 60)
    filepath := path.Join(dir, "stats")
    if f, err := os.Open(filepath); err == nil {
        content, _ := ioutil.ReadAll(f)
        f.Close()
        statsmap := make(map[string]Stats)
        json.Unmarshal(content, &statsmap)
        for k, v := range statsmap {
            var idx int
            fmt.Sscanf(k, "%d", &idx)
            self.Stats[idx] = &v
        }
    }
}

func (self *Anns) DestroyAnnModels() {
    for i := 0; i < 60; i++ {
        if self.Models[i] != nil {
            self.Models[i].Destroy()
            self.Models[i] = nil
            self.Stats[i] = nil
        }
    }
}

func NewAnnInputBoard(b *qboard.QBoard, res []ann.Type) []ann.Type {
    res = res[:25]
    info := b.GetInfo()
    for i := 0; i < 5; i++ {
        res[i] = ann.Type(info.Point[i]) / 25
    }
    for i := 0; i < 11; i++ {
        res[i+5] = ann.Type(info.Link[i]) / 40
    }
    for i := 0; i < 5; i++ {
        res[16+i] = ann.Type(info.Loop[i]) / 5
    }
    res[21] = ann.Type(info.Halfheart) / 10
    res[22] = ann.Type(info.Fourlink) / 4
    res[23], res[24] = ann.Type(b.S[b.Now])/16, ann.Type(b.S[b.Now^1])/16

    return res

}

func NewAnnInput(h, v int32, s0, s1 int, res []ann.Type) []ann.Type {
    b := qboard.NewQBoard(h, v, s0, s1, 0, 0)
    return NewAnnInputBoard(b, res)
}

type IHash interface {
    Query(b *qboard.QBoard) (win int, ok bool)
    Add(b *qboard.QBoard, win int)
}

func WhoWillWin(b *qboard.QBoard, h interface{}, stop chan int) int {
    if len(stop) > 0 {
        return -8
    }
    if ie := b.IsEnd(); ie != 0 {
        if ie > 0 {
            return 1
        }
        return 0
    }

    hash, ok := h.(IHash)
    if ok {
        if win, qok := hash.Query(b); qok {
            return win
        }
    }

    var tmpms, tmppms [60]int
    ms := b.GetMove2(tmpms[:])
    if len(ms) == 0 {
        ms = b.GetMove(tmpms[:])
    }

    for _, m := range ms {
        b.Move(m)
        mm := b.Play(tmppms[:])
        w := WhoWillWin(b, h, stop)
        b.UnMove(mm...)
        b.UnMove(m)
        if w == -8 {
            return -8
        } else if w == b.Now {
            if ok {
                hash.Add(b, b.Now)
            }
            return b.Now
        }
    }
    if ok {
        hash.Add(b, b.Now^1)
    }
    return b.Now ^ 1
}
