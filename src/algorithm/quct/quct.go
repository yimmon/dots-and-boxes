/*********************************************************************************
*     File Name           :     quct.go
*     Created By          :     YIMMON, yimmon.zhuang@gmail.com
*     Creation Date       :     [2014-05-18 13:22]
*     Last Modified       :     [2014-06-20 15:58]
*     Description         :
**********************************************************************************/

package quct

import (
    "algorithm/qboard"
    "ann/qeval"
    "ann/qrcds"
    "fmt"
    "log"
    "math"
    "math/rand"
    "runtime"
    "sort"
    "sync"
    "time"
)

const INF float64 = 1e100

var (
    numThread                    int = 1
    msMarkTurn, searchTurn       int = 14, 24
    msMark1, msMark2, searchMove int
    ucb_C, ucb_FPU               [2]float64
    timelimit_l, timelimit_h     float64
    hash                         qrcds.Hash
)

func AdjustUCB(b *qboard.QBoard, timeout *uint) {
    if b.Turn < 10 {
        ucb_FPU, msMark1, msMark2 = [2]float64{0.61, 0.61}, 5, 7
        ucb_C[b.Now], ucb_C[b.Now^1] = 2.00, 2.00
    } else if b.Turn < 21 {
        ucb_FPU, msMark1, msMark2 = [2]float64{0.61, 0.61}, 6, 8
        ucb_C[b.Now], ucb_C[b.Now^1] = 1.00, 1.00
    } else {
        ucb_FPU, msMark1, msMark2 = [2]float64{0.61, 0.61}, 7, 8
        ucb_C[b.Now], ucb_C[b.Now^1] = 1.00, 1.00
    }
    ucb_FPU[b.Now] += 0.15

    switch {
    case b.Turn <= 2:
        searchMove, timelimit_l, timelimit_h = 6, 100, 300
    case b.Turn == 4 || b.Turn == 3:
        searchMove, timelimit_l, timelimit_h = 6, 100, 300
    case b.Turn == 6 || b.Turn == 5:
        searchMove, timelimit_l, timelimit_h = 8, 1200, 3000
    case b.Turn == 8 || b.Turn == 7:
        searchMove, timelimit_l, timelimit_h = 8, 1200, 3000
    case b.Turn == 10 || b.Turn == 9:
        searchMove, timelimit_l, timelimit_h = 9, 1200, 2800
    case b.Turn == 11:
        searchMove, timelimit_l, timelimit_h = 9, 1200, 2600
    case b.Turn == 12:
        searchMove, timelimit_l, timelimit_h = 9, 1200, 2600
    case b.Turn == 14 || b.Turn == 13 || b.Turn == 15:
        searchMove, timelimit_l, timelimit_h = 10, 1200, 2400
    case b.Turn == 16 || b.Turn == 17:
        searchMove, timelimit_l, timelimit_h = 10, 1200, 2400
    case b.Turn == 18:
        searchMove, timelimit_l, timelimit_h = 10, 1200, 2400
    case b.Turn == 19:
        searchMove, timelimit_l, timelimit_h = 11, 1200, 2400
    case b.Turn == 20 || b.Turn == 21:
        searchMove, timelimit_l, timelimit_h = 12, 1200, 2400
    case b.Turn == 22 || b.Turn == 23:
        searchMove, timelimit_l, timelimit_h = 13, 1200, 2400
    default:
        searchMove, timelimit_l, timelimit_h = 14, 800, 2000
    }

    if msMarkTurn = 16; b.Now == 1 {
        msMarkTurn = 17
    }

    if b.Turn > searchTurn && *timeout > 5000 {
        *timeout = 5000
    }
}

type QUCT int

type TNode struct {
    rwMutex    sync.RWMutex
    parent     *TNode
    child      []*TNode
    fromMoves  []int
    untry      []Untry
    h, v       int32
    s          [2]int
    visit, now int
    win        [2][2]float64
}

type Untry struct {
    m   int
    x   float64
}

type ByX []Untry

func (self ByX) Len() int           { return len(self) }
func (self ByX) Swap(i, j int)      { self[i], self[j] = self[j], self[i] }
func (self ByX) Less(i, j int) bool { return self[i].x > self[j].x }

func (self *QUCT) GetName() string {
    return string("QUCT")
}

func (self *QUCT) MakeMove(b *qboard.QBoard, timeout uint, verbose bool) (h, v int32, err error) {
    if b.Turn <= 1 {
        hash.ClearAll()
        runtime.GC()
    }

    enterTime := time.Now()
    sumSimulation, bestValue, visit, numChild, mxdepth := uint32(0), float64(-INF), 0, 0, -1
    var tmpms, tmppms [60]int
    if moves := b.Play(tmppms[:]); len(moves) != 0 {
        h, v = qboard.Moves2HV(moves...)
        b.UnMove(moves...)
        return
    } else if b.IsEnd() != 0 {
        if m := b.GetCMoves(tmpms[:]); len(m) != 0 {
            h, v = qboard.Moves2HV(m...)
            return
        }
        m := b.PlayRandomOne()
        h, v = qboard.Moves2HV(m)
        b.UnMove(m)
        return
    }

    if verbose {
        defer func() {
            log.Println("Turn:", b.Turn, ", Elapse:", time.Since(enterTime).String(), ", Sim:", sumSimulation,
                ", Average:", float64(time.Since(enterTime).Nanoseconds()/1000)/float64(sumSimulation),
                "us, WinRate:", bestValue, ", SimRate:", fmt.Sprintf("%.2f%%", float64(visit)/float64(sumSimulation)*100),
                ", Child:", numChild, ", MaxDepth:", mxdepth, ", ucb_FPU:", ucb_FPU, ", ucb_C:", ucb_C,
                ", searchMove:", searchMove,
                ", msMark1:", msMark1, ", msMark2:", msMark2, ", msMarkTurn:", msMarkTurn, ", searchTurn:", searchTurn,
                ", timelimit_l:", timelimit_l, ", timelimit_h:", timelimit_h)
        }()
    }

    AdjustUCB(b, &timeout)
    var (
        exit = make(chan int, numThread)
        stop = make(chan int, 128)
    )

    if b.Turn >= searchTurn {
        go func() {
            time.Sleep(150 * time.Second)
            stop <- 1
        }()
        now := b.Now
        ms := b.GetMove(tmpms[:])
        msChan := make(chan int, 60)
        for _, m := range ms {
            msChan <- m
        }
        for i := 0; i < numThread; i++ {
            bb := b.Copy()
            go func() {
                defer func() { exit <- 1 }()
                var tmppms [60]int
                for len(stop) == 0 && len(msChan) > 0 {
                    select {
                    case m := <-msChan:
                        bb.Move(m)
                        mm := bb.Play(tmppms[:])
                        if qeval.WhoWillWin(bb, &hash, stop) == now {
                            stop <- 1
                            h, v = qboard.Moves2HV(m)
                            return
                        }
                        bb.UnMove(mm...)
                        bb.UnMove(m)
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

    if b.Turn > searchTurn && timeout > 5000 {
        timeout = 5000
    }
    go func() {
        time.Sleep(time.Duration(timeout) * time.Millisecond)
        stop <- 1
    }()

    var root = new(TNode)
    root.Init(b)
    for t := 0; t < numThread; t++ {
        tb := b.Copy()
        go func() {
            defer func() { exit <- 1 }()
            bb := new(qboard.QBoard)
            for len(stop) == 0 {
                tb.CopyTo(bb)
                cur, rootturn := root, bb.Turn
                for next := cur.SelectBest(bb); next != nil; {
                    cur = next
                    next = cur.SelectBest(bb)
                }
                if bb.Turn-rootturn > mxdepth {
                    mxdepth = bb.Turn - rootturn
                }

                if newcur := cur.AddChild(bb, bb.Turn == rootturn); newcur == nil {
                    continue
                } else if newcur == cur {
                    goto WINRATE
                } else {
                    cur = newcur
                }
                for bb.IsEnd() == 0 {
                    if !Sim(bb) {
                        break
                    }
                }
            WINRATE:
                r0, r1 := WinRate(bb)
                for cur != nil {
                    cur.Update(r0, r1)
                    cur = cur.parent
                }
            }
        }()
    }

    go func() {
        defer func() { exit <- 1 }()
        const duration int = 300
        timePoint, simCount := time.Now(), root.visit
        time.Sleep(time.Duration(duration) * time.Millisecond)
        var resch *TNode = nil
        var rescnt, searchMove_b int = 0, searchMove
        for len(stop) == 0 {
            best, besti := float64(0), 0
            for i, ch := range root.child {
                tmp := float64(ch.win[0][b.Now]) / float64(ch.visit)
                if tmp > best {
                    best, besti = tmp, i
                }
            }
            if (best > 0.995 || best < 0.005) && root.child[besti].visit > 8000 {
                stop <- 1
                break
            } else if float64(root.child[besti].visit)/float64(root.visit) > 0.90 {
                if resch == root.child[besti] {
                    if rescnt++; rescnt >= 5000/duration {
                        stop <- 1
                        break
                    }
                } else {
                    resch, rescnt = root.child[besti], 1
                }
            } else {
                resch, rescnt = nil, 0
            }
            if t := float64(time.Since(timePoint).Nanoseconds()/1000) / float64(root.visit-simCount+1); t > timelimit_h &&
                searchMove-1 >= searchMove_b {
                searchMove--
            } else if t < timelimit_l && searchMove+1 <= searchMove_b+6+numThread && searchMove+1 <= searchMove_b+12 {
                searchMove++
            }
            timePoint, simCount = time.Now(), root.visit
            time.Sleep(time.Duration(duration) * time.Millisecond)
        }
    }()

    for i := 0; i < numThread; i++ {
        <-exit
    }
    stop <- 1
    <-exit

    resch := root.child[0]
    for _, ch := range root.child {
        if ch.visit > resch.visit {
            bestValue = float64(ch.win[0][b.Now]) / float64(ch.visit)
            resch, visit = ch, ch.visit
        }
    }
    h, v = qboard.Moves2HV(resch.fromMoves[0])
    bestValue, visit = float64(resch.win[0][b.Now])/float64(resch.visit), resch.visit
    numChild, sumSimulation = len(root.child), uint32(root.visit)
    if rand.Intn(8) == 0 {
        defer runtime.GC()
    }
    return
}

func Sim(b *qboard.QBoard) bool {
    if b.NumMoveAll() <= searchMove+rand.Intn(2) {
        return false
    }
    var tmpms, tmppms [60]int
    ms := b.GetMove2(tmpms[:])
    if len(ms) == 0 {
        if b.Turn < msMarkTurn {
            ms = b.GetMove12no44(tmpms[:])
            if len(ms) < msMark2 {
                ms = b.GetMove12(tmpms[:])
            }
        } else {
            ms = b.GetMove12(tmpms[:])
        }
        if len(ms) < msMark2 {
            ms = append(ms, b.GetMove3(ms[len(ms):cap(ms)])...)
        }
    }

    b.Move(ms[rand.Intn(len(ms))])
    b.Play(tmppms[:])
    return true
}

var invalidchan = make(chan int)

func WinRate(b *qboard.QBoard) (r0, r1 float64) {
    if ie := b.IsEnd(); ie != 0 {
        if ie < 0 {
            r0, r1 = 1, 0
        } else {
            r0, r1 = 0, 1
        }
    } else if b.NumMoveAll() <= searchMove+b.Now {
        if qeval.WhoWillWin(b, &hash, invalidchan) == 0 {
            r0, r1 = 1, 0
        } else {
            r0, r1 = 0, 1
        }
    } else {
        Sim(b)
        r0, r1 = WinRate(b)
    }

    return
}

func (self *TNode) Init(b *qboard.QBoard) {
    self.rwMutex.Lock()
    defer self.rwMutex.Unlock()
    if self.parent != nil {
        self.parent.rwMutex.Lock()
        defer self.parent.rwMutex.Unlock()
    }
    self.parent, self.child = nil, nil
    self.fromMoves = make([]int, 0, 1)
    self.h, self.v = b.H, b.V
    self.s[0], self.s[1] = b.S[0], b.S[1]
    self.visit, self.now = 0, b.Now
    self.win[0][0], self.win[0][1] = 0, 0
    self.win[1][0], self.win[1][1] = 0, 0
    self.untry = nil
}

func (self *TNode) AddChild(b *qboard.QBoard, force bool) *TNode {
    if b.IsEnd() != 0 {
        return self
    }
    self.rwMutex.Lock()
    defer self.rwMutex.Unlock()
    if self.parent != nil {
        self.parent.rwMutex.Lock()
        defer self.parent.rwMutex.Unlock()
    }

    var tmpms, tmppms [60]int
    if len(self.untry) == 0 && len(self.child) == 0 {
        if !force && b.NumMoveAll() <= searchMove {
            return self
        }
        ms := b.GetMove2(tmpms[:])
        if len(ms) == 0 {
            if b.Turn < msMarkTurn {
                ms = b.GetMove12no44(tmpms[:])
                if len(ms) < msMark1 {
                    ms = b.GetMove12(tmpms[:])
                } else if force {
                    if b.Turn < 13 && len(ms) > 20 {
                        ms = Shuffle(ms)[:18]
                    } else if b.Turn < 16 && len(ms) > 25 {
                        ms = Shuffle(ms)[:22]
                    }
                    tms, ltms := Shuffle(b.GetMove12only44(ms[len(ms):cap(ms)])), 5
                    if len(ms)+ltms < 22 {
                        ltms = 22 - len(ms)
                    }
                    if len(tms) < ltms {
                        ltms = len(tms)
                    }
                    ms = append(ms, tms[:ltms]...)
                }
            } else {
                ms = b.GetMove12(tmpms[:])
            }

            if b.Turn < 13 && len(ms) > 20 {
                ms = Shuffle(ms)[:18]
            } else if b.Turn < 16 && len(ms) > 25 {
                ms = Shuffle(ms)[:22]
            }
            if len(ms) < msMark1 || b.Turn >= 20 {
                ms = append(ms, b.GetMove3(ms[len(ms):cap(ms)])...)
            } else if b.Turn >= 16 {
                if ms3 := b.GetMove3(ms[len(ms):cap(ms)]); len(ms3) > 0 {
                    m3, l3 := ms3[0], 99
                    for _, tm := range ms3 {
                        if ll := b.LinkLength(tm); ll < l3 {
                            m3, l3 = tm, ll
                        }
                    }
                    ms = append(ms, m3)
                }
            }
        }
        self.untry = make([]Untry, len(ms))
        for i, m := range ms {
            self.untry[i].m = m
        }
    }

    if len(self.untry) == 0 {
        return nil
    }

    var rew [60]float64
    if self.parent != nil {
        for _, c := range self.parent.child {
            if c.visit > 0 {
                rew[c.fromMoves[0]] = c.win[0][self.now]/float64(c.visit) + 1e-10
            }
        }
    }
    for i, un := range self.untry {
        m := un.m
        if rew[m] > 0 {
            self.untry[i].x = rew[m]
        } else {
            self.untry[i].x = ucb_FPU[self.now] - (float64(b.EdgeDegree(m)) * 1e-5) + rand.Float64()*1e-8
        }
    }
    sort.Sort(ByX(self.untry))

    b.Move(self.untry[0].m)
    mm := b.Play(tmppms[:])
    np := new(TNode)
    np.Init(b)
    np.parent = self
    np.fromMoves = append(np.fromMoves, self.untry[0].m)
    np.fromMoves = append(np.fromMoves, mm...)
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

func (self *TNode) SelectBest(b *qboard.QBoard) *TNode {
    self.rwMutex.RLock()
    defer self.rwMutex.RUnlock()
    if self.parent != nil {
        self.parent.rwMutex.RLock()
        defer self.parent.rwMutex.RUnlock()
    }

    if len(self.child) == 0 || self.child[0].visit == 0 {
        return nil
    }

    var (
        val float64 = -INF
        res         = self.child[0]
    )
    logSum := math.Log(float64(self.visit))
    for _, ch := range self.child {
        if ch.visit > 0 {
            tmp := float64(ch.win[0][self.now]) / float64(ch.visit)
            vj := float64(ch.win[1][self.now])/float64(ch.visit) - tmp*tmp + math.Sqrt(2*logSum/float64(ch.visit))
            tmp += ucb_C[self.now] * math.Sqrt(math.Min(0.25, vj)*logSum/float64(ch.visit))
            if tmp > val {
                val, res = tmp, ch
            }
        }
    }
    if len(self.untry) > 0 &&
        val < float64(self.untry[0].x)+ucb_C[self.now]*math.Sqrt(logSum/(float64(self.visit)/float64(len(self.child)))*0.25) {
        return nil
    }
    b.Move(res.fromMoves...)
    return res
}

func (self *TNode) Update(r0, r1 float64) {
    self.rwMutex.Lock()
    defer self.rwMutex.Unlock()
    if self.parent != nil {
        self.parent.rwMutex.Lock()
        defer self.parent.rwMutex.Unlock()
    }

    self.win[0][0] += r0
    self.win[0][1] += r1
    self.win[1][0] += r0 * r0
    self.win[1][1] += r1 * r1
    self.visit++
}

func Shuffle(arr []int) []int {
    for i, j := range rand.Perm(len(arr)) {
        arr[i], arr[j] = arr[j], arr[i]
    }
    return arr
}

func SetNumThread(n int) {
    numThread = n
}

func init() {
    rand.Seed(time.Now().Unix())
    numThread = runtime.NumCPU() + 1

    hash.Init(func(k *qrcds.HashKey, v *qrcds.HashValue) {})
}
