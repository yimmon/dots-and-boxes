/*********************************************************************************
*     File Name           :     alphabeta.go
*     Created By          :     YIMMON, yimmon.zhuang@gmail.com
*     Creation Date       :     [2014-04-22 11:00]
*     Last Modified       :     [2014-05-11 15:19]
*     Description         :
**********************************************************************************/

package alphabeta

import (
    "algorithm/board"
    "ann/eval"
    "log"
    "math/rand"
    "runtime"
    "sync/atomic"
    "time"
)

type AlphaBeta struct {
    stop      *int32
    leafCount int32
}

const (
    noCMark int     = 8
    INF     float64 = 1e100
)

var (
    numThread int = 1
    annsChan      = make(chan *eval.Anns, 1024)
)

func (self *AlphaBeta) GetName() string {
    return "AlphaBeta"
}

func (self *AlphaBeta) MakeMove(b *board.Board, timeout uint, verbose bool) (h, v int32, err error) {
    var (
        enterTime         = time.Now()
        bestValue float64 = -INF
        bv        [60]float64
        moves     [60]*board.Moves
        exit      = make(chan int, numThread)
    )
    self.leafCount, self.stop = 0, new(int32)
    if verbose {
        defer func() {
            log.Println("Turn:", b.Turn, ", Elapse:", time.Since(enterTime).String(), ", Leaves:", self.leafCount,
                ", BestValue:", bestValue)
            runtime.GC()
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

    mt, _ := b.PlayRandomOne()
    h, v = mt.Moves2HV()
    b.UnMove(mt)
    depthChan := make(chan int, 32)
    for i := 1; i < 30; i++ {
        depthChan <- i
    }

    for t := 0; t < numThread; t++ {
        go func() {
            var anns *eval.Anns
            select {
            case anns = <-annsChan:
            default:
                anns = eval.GetAnnModels("./AnnModels")
            }
            bb := board.NewBoard(b.H, b.V, b.S[0], b.S[1], b.Now, b.Turn)
        LOOP:
            for len(depthChan) > 0 {
                select {
                case dep := <-depthChan:
                    bv[dep], moves[dep] = self.AlphaBeta(anns, bb, bb.Now, 0.0, 2.0, dep)
                    if atomic.LoadInt32(self.stop) != 0 {
                        bv[dep] = -bv[dep]
                        break LOOP
                    }
                default:
                }
            }
            annsChan <- anns
            exit <- 1
        }()
    }

    go func(ptr *int32) {
        time.Sleep(time.Duration(timeout) * time.Millisecond)
        atomic.AddInt32(ptr, 1)
    }(self.stop)

    for i := 0; i < numThread; i++ {
        <-exit
    }
    for d, m := range moves {
        if m != nil {
            if bv[d] > 0 {
                bestValue = bv[d]
                h, v = m.Moves2HV()
            } else {
                if -bv[d] > bestValue {
                    bestValue = -bv[d]
                    h, v = m.Moves2HV()
                }
            }
        }
    }
    return
}

func (self *AlphaBeta) AlphaBeta(anns *eval.Anns, b *board.Board, root int8,
    alpha, beta float64, depth int) (val float64, moves *board.Moves) {
    if depth <= 0 || b.IsEnd() != 0 {
        if val = self.Evaluate(anns, b, root); alpha < 0 {
            val = -val
        }
        if val >= beta {
            val = beta
        } else if val < alpha {
            val = alpha
        }
        atomic.AddInt32(&self.leafCount, 1)
        return
    }

    now, tmp := b.Now, float64(0)
    ms, noC, _ := b.GetMove()
    if noC >= noCMark && len(ms)-noC > 0 { // **布局阶段未结束
        tms, ntms := make([]*board.Moves, 60), 0
        for _, m := range ms {
            if (b.HasNoCAfter(m) && !b.LinksTwo4(m)) || b.CanGetPointAfter(m) {
                tms[ntms] = m
                ntms++
            }
        }
        if ntms == 0 {
            for _, m := range ms {
                if b.HasNoCAfter(m) {
                    tms[ntms] = m
                    ntms++
                }
            }
        }
        if ntms == 0 {
            log.Fatal("AlphaBeta: ntms == 0.\n" + b.Draw())
        }

        ms = tms[:ntms]
    }

    for _, idx := range rand.Perm(len(ms)) {
        m := ms[idx]
        if err := b.Move(m); err != nil {
            log.Panic("AlphaBeta move fail.")
        }
        mm, _ := b.Play()

        if b.Now != now {
            tmp, _ = self.AlphaBeta(anns, b, root, -beta, -alpha, depth-1)
            tmp = -tmp
        } else {
            tmp, _ = self.AlphaBeta(anns, b, root, alpha, beta, depth-1)
        }

        if b.UnMove(mm, m); tmp > alpha || (tmp == alpha && moves == nil) {
            alpha, moves = tmp, m
        }
        if alpha >= beta {
            return beta, moves
        }
        if atomic.LoadInt32(self.stop) != 0 {
            break
        }
    }

    return alpha, moves
}
func (self *AlphaBeta) Evaluate(anns *eval.Anns, b *board.Board, root int8) (val float64) {
    if win := b.IsEnd(); win != 0 {
        if (win < 0 && root == 0) || (win > 0 && root == 1) {
            return INF
        } else {
            return 0
        }
    }

    input := eval.NewAnnInput(b.H, b.V, b.S[b.Now], b.S[b.Now^1])
    val = float64(anns.Models[b.Turn].Run(input)[0])
    if b.Now != root {
        val = -val
    }
    val += 1.0 // 0.0 ~ 2.0
    return
}

func SetNumThread(n int) {
    numThread = n
}

func init() {
    rand.Seed(time.Now().Unix())
    numThread = runtime.NumCPU()
}
