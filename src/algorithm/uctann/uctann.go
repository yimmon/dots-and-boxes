/*********************************************************************************
*     File Name           :     uctann.go
*     Created By          :     YIMMON, yimmon.zhuang@gmail.com
*     Creation Date       :     [2014-05-07 22:31]
*     Last Modified       :     [2014-05-18 00:24]
*     Description         :
**********************************************************************************/

package uctann

import (
    "algorithm/board"
    "ann/eval"
    "ann/rcds"
    "fmt"
    "log"
    "math"
    "math/rand"
    "runtime"
    "sort"
    "sync"
    "time"
)

const (
    INF        float32 = 1e20
    noCMark1   int     = 5
    noCMark2   int     = 3
    searchTurn int8    = 22
    annTurn    int8    = 24
    leafturn   int8    = 32
)

var (
    ucb_C      float64 = 1.4142135623730951
    searchMove int     = 5
    numThread  int     = 1
    annsChan           = make(chan *eval.Anns, 1024)
    hash       rcds.Hash
)

type UCTANN int

type TNode struct {
    rwMutex   sync.RWMutex
    parent    *TNode
    child     []*TNode
    fromMoves [2]*Move
    untry     []*Move
    visit     int
    now       int8
    win       float32
}

type Move struct {
    H, V, M int32
}

func (self *UCTANN) GetName() string {
    return string("UCT-ANN")
}

func (self *UCTANN) MakeMove(b *board.Board, timeout uint, verbose bool) (h, v int32, err error) {
    enterTime := time.Now()
    sumSimulation, bestValue, visit, numChild := uint32(0), float32(-1e30), 0, 0
    var mx int8 = -1
    if verbose {
        defer func() {
            fmt.Printf("mx: %d\n", mx)
            log.Println("Turn:", b.Turn, ", Elapse:", time.Since(enterTime).String(), ", Sim:", sumSimulation,
                ", Average:", float32(time.Since(enterTime).Nanoseconds()/1000)/float32(sumSimulation),
                "us, WinRate:", bestValue,
                ", SimRate:", fmt.Sprintf("%.2f%%", float32(visit)/float32(sumSimulation)*100), ", Child:", numChild)
            defer runtime.GC()
        }()
    }

    if moves, _ := b.Play(); moves != nil && (moves.H != 0 || moves.V != 0 || moves.M != 0) {
        h, v = moves.Moves2HV()
        b.UnMove(moves)
        return
    } else if b.IsEnd() != 0 {
        if m, _ := b.GetCMoves(); m != nil {
            h, v = m.Moves2HV()
            return
        }
        m, _ := b.PlayRandomOne()
        h, v = m.Moves2HV()
        b.UnMove(m)
        return
    }

    var (
        exit = make(chan int, numThread)
        stop = make(chan int, 128)
        root = new(TNode)
    )

    if b.Turn >= searchTurn {
        go func() {
            time.Sleep(2 * time.Minute)
            stop <- 1
        }()
        now := b.Now
        ms, _, _ := b.GetMove()
        msChan := make(chan *board.Moves, 60)
        for _, m := range ms {
            msChan <- m
        }
        for i := 0; i < numThread; i++ {
            go func() {
                defer func() { exit <- 1 }()
                bb := board.NewBoard(b.H, b.V, b.S[0], b.S[1], b.Now, b.Turn)
                for len(stop) == 0 && len(msChan) > 0 {
                    select {
                    case m := <-msChan:
                        tm := bb.NewMoves(m.H, m.V, m.M)
                        bb.Move(tm)
                        mm, _ := bb.Play()
                        if eval.WhoWillWin(bb, &hash, 99, nil, stop) == now {
                            stop <- 1
                            h, v = m.Moves2HV()
                            return
                        }
                        bb.UnMove(mm, tm)
                    default:
                    }
                }
            }()
        }
        for i := 0; i < numThread; i++ {
            <-exit
        }
        if h != 0 || v != 0 {
            return
        }

        for len(stop) > 0 {
            <-stop
        }
    }

    go func() {
        time.Sleep(time.Duration(timeout) * time.Millisecond)
        stop <- 1
    }()

    AdjustUCB(b)
    root.Init(b)
    for t := 0; t < numThread; t++ {
        go func() {
            defer func() { exit <- 1 }()
            var anns *eval.Anns
            select {
            case anns = <-annsChan:
            default:
                anns = eval.GetAnnModels("./AnnModels")
            }
            for len(stop) == 0 {
                bb := board.NewBoard(b.H, b.V, b.S[0], b.S[1], b.Now, b.Turn)
                cur, rootturn := root, bb.Turn
                for next := cur.SelectBest(bb); next != nil; {
                    cur = next
                    next = cur.SelectBest(bb)
                }
                if bb.Turn-rootturn > mx {
                    mx = bb.Turn - rootturn
                }

                if cur = cur.AddChild(bb, anns, bb.Turn == b.Turn); cur == nil {
                    continue
                }
                rleafturn := leafturn + int8(rand.Intn(2))
                for bb.Turn < rleafturn && bb.IsEnd() == 0 {
                    if !Sim(bb, anns) {
                        break
                    }
                }
                r0, r1 := WinRate(bb, anns)
                for cur != nil {
                    cur.Update(r0, r1)
                    cur = cur.parent
                }
            }
            annsChan <- anns
        }()
    }

    go func() {
        defer func() { exit <- 1 }()
        time.Sleep(3 * time.Second)
        for len(stop) == 0 {
            best := float32(0)
            for _, ch := range root.child {
                tmp := float32(ch.win) / float32(ch.visit)
                if ch.now != root.now {
                    tmp = 1 - tmp
                }
                if tmp > best {
                    best = tmp
                }
            }
            if best > 0.99 || best < 0.01 {
                stop <- 1
                break
            }
            time.Sleep(200 * time.Millisecond)
        }
    }()

    for i := 0; i < numThread; i++ {
        <-exit
    }
    stop <- 1
    <-exit

    numChild = len(root.child)
    m := root.child[0].fromMoves[0]
    for _, ch := range root.child {
        tmp := float32(ch.win) / float32(ch.visit)
        if ch.now != root.now {
            tmp = 1 - tmp
        }
        if tmp > bestValue {
            bestValue, m, visit = tmp, ch.fromMoves[0], ch.visit
        }
    }
    h, v = b.NewMoves(m.H, m.V, m.M).Moves2HV()
    sumSimulation = uint32(root.visit)
    root = nil
    return
}

func Sim(b *board.Board, anns *eval.Anns) bool {
    ms, noC, _ := b.GetMove()
    if len(ms) <= searchMove {
        return false
    }
    if noC >= noCMark2 && len(ms)-noC > 0 { // **布局阶段未结束
        tms := make([]*board.Moves, 0, len(ms))
        for _, m := range ms {
            if b.HasNoCAfter(m) || b.CanGetPointAfter(m) || b.LoseOneAfter(m) {
                tms = append(tms, m)
            }
        }
        if len(tms) == 0 {
            log.Fatal("Sim: len(tms) == 0.\n" + b.Draw())
        }
        ms = tms
    }

    if b.Turn >= annTurn-4 {
        resm, best, now := ms[0], -INF, b.Now
        for _, m := range ms {
            b.Move(m)
            mm, _ := b.Play()
            input := eval.NewAnnInputBoard(b)
            z := float32(anns.Models[b.Turn].Run(input)[0])
            if b.Now != now {
                if b.Turn < annTurn {
                    z = -z
                } else {
                    z = anns.Stats[b.Turn].Wrate + anns.Stats[b.Turn].Lrate - z
                }
            }
            if z > best || (z == best && b.Now == now) {
                resm, best = m, z
            }
            b.UnMove(mm, m)
        }
        b.Move(resm)
    } else {
        b.Move(ms[rand.Intn(len(ms))])
    }
    b.Play()
    return true
}

var invalidchan = make(chan int)

func WinRate(b *board.Board, anns *eval.Anns) (r0, r1 float32) {
    if ie := b.IsEnd(); ie != 0 {
        if ie < 0 {
            r0, r1 = 1, 0
        } else {
            r0, r1 = 0, 1
        }
    } else if ms, _, _ := b.GetMove(); len(ms) <= searchMove {
        if eval.WhoWillWin(b, &hash, 99, nil, invalidchan) == 0 {
            r0, r1 = 1, 0
        } else {
            r0, r1 = 0, 1
        }
    } else if y := anns.WinOddBoard(b); y >= 0 {
        if b.Now == 0 {
            r0, r1 = y, 1-y
        } else {
            r0, r1 = 1-y, y
        }
    } else {
        Sim(b, anns)
        r0, r1 = WinRate(b, anns)
    }

    return
}

func (self *TNode) Init(b *board.Board) {
    self.rwMutex.Lock()
    defer self.rwMutex.Unlock()
    if self.parent != nil {
        self.parent.rwMutex.Lock()
        defer self.parent.rwMutex.Unlock()
    }
    self.parent, self.child = nil, nil
    self.fromMoves[0], self.fromMoves[1] = nil, nil
    self.visit, self.win, self.now = 0, 0, b.Now
    self.untry = nil
}

type SortMove struct {
    z   float32
    m   *board.Moves
}

type ByZ []SortMove

func (self ByZ) Len() int           { return len(self) }
func (self ByZ) Swap(i, j int)      { self[i], self[j] = self[j], self[i] }
func (self ByZ) Less(i, j int) bool { return self[i].z > self[j].z }

func (self *TNode) AddChild(b *board.Board, anns *eval.Anns, isRoot bool) *TNode {
    self.rwMutex.Lock()
    defer self.rwMutex.Unlock()
    if self.parent != nil {
        self.parent.rwMutex.Lock()
        defer self.parent.rwMutex.Unlock()
    }

    if b.IsEnd() != 0 {
        return self
    } else if len(self.untry) == 0 && len(self.child) == 0 {
        ms, noC, _ := b.GetMove()
        if noC >= noCMark1 && len(ms)-noC > 0 { // **布局阶段未结束
            tms := make([]*board.Moves, 0, len(ms))
            for _, m := range ms {
                if b.HasNoCAfter(m) || b.CanGetPointAfter(m) || b.LoseOneAfter(m) {
                    tms = append(tms, m)
                }
            }
            if len(tms) == 0 {
                log.Fatal("TNode Init: len(tms) == 0.\n" + b.Draw())
            }
            ms = tms
        }
        if len(ms) > 0 {
            self.untry = make([]*Move, len(ms))
            if b.Turn >= annTurn-4 {
                now := b.Now
                sm := make([]SortMove, len(ms))
                for i, m := range ms {
                    b.Move(m)
                    mm, _ := b.Play()
                    input := eval.NewAnnInputBoard(b)
                    z := float32(anns.Models[b.Turn].Run(input)[0])
                    if b.Now != now {
                        if b.Turn < annTurn {
                            z = -z
                        } else {
                            z = anns.Stats[b.Turn].Wrate + anns.Stats[b.Turn].Lrate - z
                        }
                    }
                    b.UnMove(mm, m)
                    sm[i].z, sm[i].m = z, m
                }
                sort.Sort(ByZ(sm))

                for i, s := range sm {
                    self.untry[i] = &Move{H: s.m.H, V: s.m.V, M: s.m.M}
                }
            } else {
                for i, j := range rand.Perm(len(ms)) {
                    self.untry[i] = &Move{H: ms[j].H, V: ms[j].V, M: ms[j].M}
                }
            }
        }
    }
    if len(self.untry) == 0 {
        return nil
    } else if !isRoot && len(self.untry)+len(self.child) <= searchMove {
        return self
    }

    b.Move(b.NewMoves(self.untry[0].H, self.untry[0].V, self.untry[0].M))
    mm, _ := b.Play()
    np := new(TNode)
    np.Init(b)
    np.parent = self
    np.fromMoves[0] = self.untry[0]
    if mm != nil {
        np.fromMoves[1] = &Move{H: mm.H, V: mm.V, M: mm.M}
    }
    if self.child == nil {
        self.child = make([]*TNode, 0, len(self.untry))
    }
    self.child = append(self.child, np)
    if len(self.untry) > 1 {
        self.untry = self.untry[1:]
    } else {
        self.untry = nil
    }
    return np
}

func (self *TNode) SelectBest(b *board.Board) *TNode {
    self.rwMutex.RLock()
    defer self.rwMutex.RUnlock()
    if self.parent != nil {
        self.parent.rwMutex.RLock()
        defer self.parent.rwMutex.RUnlock()
    }
    if len(self.untry) > 0 || len(self.child) == 0 {
        return nil
    }

    res := self.child[0]
    val := -1e99
    logSum := math.Log(float64(self.visit))
    for _, ch := range self.child {
        tmp := float64(ch.win) / float64(ch.visit)
        if ch.now != self.now {
            tmp = 1 - tmp
        }
        tmp += ucb_C * math.Sqrt(logSum/float64(ch.visit))
        if tmp > val {
            val, res = tmp, ch
        }
    }
    b.Move(b.NewMoves(res.fromMoves[0].H, res.fromMoves[0].V, res.fromMoves[0].M))
    if res.fromMoves[1] != nil {
        b.Move(b.NewMoves(res.fromMoves[1].H, res.fromMoves[1].V, res.fromMoves[1].M))
    }
    return res
}

func (self *TNode) Update(r0, r1 float32) {
    self.rwMutex.Lock()
    defer self.rwMutex.Unlock()
    if self.parent != nil {
        self.parent.rwMutex.Lock()
        defer self.parent.rwMutex.Unlock()
    }
    self.visit++
    if self.now == 0 {
        self.win += r0
    } else {
        self.win += r1
    }
}

func AdjustUCB(b *board.Board) {
    switch {
    case b.Turn <= 11:
        ucb_C = math.Sqrt(2.0) * 1.00
        searchMove = 5
    case b.Turn <= 13:
        ucb_C = math.Sqrt(2.0) * 0.80
        searchMove = 6
    case b.Turn <= 15:
        ucb_C = math.Sqrt(2.0) * 0.70
        searchMove = 7
    case b.Turn <= 17:
        ucb_C = math.Sqrt(2.0) * 0.60
        searchMove = 9
    case b.Turn <= 19:
        ucb_C = math.Sqrt(2.0) * 0.55
        searchMove = 11
    case b.Turn <= 23:
        ucb_C = math.Sqrt(2.0) * 0.50
        searchMove = 12
    case b.Turn <= 27:
        ucb_C = math.Sqrt(2.0) * 0.40
        searchMove = 13
    case b.Turn <= 31:
        ucb_C = math.Sqrt(2.0) * 0.30
        searchMove = 13
    default:
        ucb_C = math.Sqrt(2.0) * 0.20
        searchMove = 13
    }
}

func SetNumThread(n int) {
    numThread = n
}

func init() {
    rand.Seed(time.Now().Unix())
    numThread = runtime.NumCPU() + 1

    hash.Init(func(k *rcds.HashKey, v *rcds.HashValue) {})
}
