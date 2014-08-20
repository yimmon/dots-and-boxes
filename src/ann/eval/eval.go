/*********************************************************************************
*     File Name           :     eval.go
*     Created By          :     YIMMON, yimmon.zhuang@gmail.com
*     Creation Date       :     [2014-05-11 11:37]
*     Last Modified       :     [2014-05-18 21:35]
*     Description         :
**********************************************************************************/

package eval

import (
    "algorithm/board"
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
    Wrate, Lrate float32
    Pwcor, Plcor float32
}

func (self *Anns) WinOddBoard(b *board.Board) float32 {
    var wrate, lrate float32 = 0.05, -0.55
    var pwcor, plcor float32 = 1, 1
    turn := b.Turn
    if self.Stats[turn] != nil {
        wrate, lrate = self.Stats[turn].Wrate, self.Stats[turn].Lrate
        pwcor, plcor = self.Stats[turn].Pwcor, self.Stats[turn].Plcor
    }

    input := NewAnnInputBoard(b)
    z := float32(self.Models[turn].Run(input)[0])
    if z > wrate {
        return pwcor
    } else if z < lrate {
        return 1 - plcor
    }
    return -1
}

func (self *Anns) WinOdd(h, v int32, s0, s1, turn int8) float32 {
    b := board.NewBoard(h, v, s0, s1, 0, turn)
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

func NewAnnInputBoard(b *board.Board) []ann.Type {
    /*
       res := make([]ann.Type, 62, 64)
       for j, o := range [2]int32{h, v} {
           for k := uint(0); k < 30; k++ {
               if (1<<k)&o != 0 {
                   res[j*30+int(k)] = ann.Type(0)
               } else {
                   res[j*30+int(k)] = ann.Type(1)
               }
           }
       }
       res[60], res[61] = ann.Type(s0)/13.0, ann.Type(s1)/13.0
    */
    res := make([]ann.Type, 25, 32)
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
    res[23], res[24] = ann.Type(b.S[b.Now])/16.0, ann.Type(b.S[b.Now^1])/16.0

    return res

}

func NewAnnInput(h, v int32, s0, s1 int8) []ann.Type {
    b := board.NewBoard(h, v, s0, s1, 0, 0)
    return NewAnnInputBoard(b)
}

type IHash interface {
    Query(b *board.Board) (win int8, ok bool)
    Add(b *board.Board, win int8)
}

func WhoWillWin(b *board.Board, h interface{}, leafturn int, anns *Anns, stop chan int) int8 {
    if len(stop) > 0 {
        return -8
    }
    if ie := b.IsEnd(); ie != 0 {
        if ie > 0 {
            return 1
        }
        return 0
    }   /* else if int(b.Turn) >= leafturn && b.Turn <= 34 {
        y := anns.WinOddBoard(b)
        if y > 0.7 {
            return b.Now
        } else if y <= 0.3 && y >= 0 {
            return b.Now ^ 1
        }
        return -1
    }*/

    hash, ok := h.(IHash)
    if ok {
        if win, qok := hash.Query(b); qok {
            return win
        }
    }

    ms, _, _ := b.GetMove()
    for _, m := range ms {
        b.Move(m)
        mm, _ := b.Play()
        w := WhoWillWin(b, h, leafturn, anns, stop)
        b.UnMove(mm, m)
        if w == -8 {
            return -8
        }
        if w == b.Now {
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
