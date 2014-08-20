/*********************************************************************************
*     File Name           :     uct.go
*     Created By          :     YIMMON, yimmon.zhuang@gmail.com
*     Creation Date       :     [2014-04-09 14:33]
*     Last Modified       :     [2014-05-05 21:08]
*     Description         :
**********************************************************************************/

package uct

import (
    "algorithm/board"
    "fmt"
    "log"
    "math"
    "math/rand"
    "runtime"
    "sync"
    "sync/atomic"
    "time"
)

const (
    ucb_C     float64 = 0.4142135623730951
    hashBlock uint    = 23 // prime
    hashSize  uint    = 13000000 / hashBlock
    hashClean uint    = hashSize / 100
    noCMark   int     = 8
)

var (
    numThread  int    = 1
    timeStamp  uint64 = 0
    hashExpire uint64 = uint64(hashSize*hashBlock) / 2 * 3
    hashTable  [hashBlock]map[HashKey]*HashValue
    rwMutex    [hashBlock]sync.RWMutex
)

type UCT int

type HashKey struct {
    h, v uint32
    o    uint16
}

type HashValue struct {
    visit, win uint32
    stamp      uint64
}

func (self *UCT) GetName() string {
    return string("UCT")
}

func (self *UCT) MakeMove(b *board.Board, timeout uint, verbose bool) (h, v int32, err error) {
    defer runtime.GC()
    enterTime := time.Now()
    var bestptr *HashValue
    sumSimulation, bestValue := uint32(0), float64(-1)
    if verbose {
        defer func() {
            visit := -1
            if bestptr != nil {
                visit = int(bestptr.visit)
            }
            log.Println("Elapse:", time.Since(enterTime).String(), ", Sim:", sumSimulation,
                ", Average:", float32(time.Since(enterTime).Nanoseconds()/1000)/float32(sumSimulation),
                "us, WinRate:", bestValue,
                ", SimRate:", fmt.Sprintf("%.2f%%", float32(visit)/float32(sumSimulation)*100))
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
        exit               = make(chan int, numThread)
        stop               = make(chan<- int, numThread+1)
        tip                = make(chan int, 1)
        lastM, count int32 = 0, 0
    )
    for i := 0; i < numThread; i++ {
        go func() {
            bb := board.NewBoard(b.H, b.V, b.S[0], b.S[1], b.Now, b.Turn)
            bptr := GetHashValue(NewHashKey(bb), true)
            for len(stop) == 0 {
                Simulation(bb, bptr)
                if bb.H != b.H || bb.V != b.V || bb.S != b.S || bb.Now != b.Now {
                    log.Panic("Simulation bug!!!")
                }

                if len(tip) > 0 {
                    select {
                    case <-tip:
                        if m, _, _, _, _ := SelectBest(bb, bptr, true); m.M == lastM {
                            if count++; count >= 50 {
                                stop <- 1
                            }
                        } else {
                            lastM, count = m.M, 0
                        }
                    default:
                    }
                }
            }
            exit <- 1
        }()
    }
    go func() {
        time.Sleep(time.Duration(timeout) * time.Millisecond)
        stop <- 1
    }()
    if timeout >= 1000 {
        go func() {
            for len(tip) == 0 {
                select {
                case tip <- 1:
                default:
                }
                time.Sleep(100 * time.Millisecond)
            }
        }()
    }

    for i := 0; i < numThread; i++ {
        <-exit
    }

    m, bestptr, sumSimulation, bestValue, err := SelectBest(b, nil, true)
    h, v = m.Moves2HV()
    return
}

func Simulation(b *board.Board, ptr *HashValue) (w int8) {
    if e := b.IsEnd(); e != 0 {
        if e == -1 {
            if b.Now == 0 {
                return 2
            }
            return 0
        } else {
            if b.Now == 1 {
                return 2
            }
            return 1
        }
    }

    var (
        now    = b.Now
        m1, m2 *board.Moves
        sptr   *HashValue
    )

    if ptr == nil {
        ptr = GetHashValue(NewHashKey(b), true)
    }
    if atomic.LoadUint32(&ptr.visit) == 0 {
        m1, _ = b.PlayRandomOne()
        m2, _ = b.Play()
        sptr = GetHashValue(NewHashKey(b), true)
    } else {
        bv := float64(0)
        m1, sptr, _, bv, _ = SelectBest(b, ptr, false)
        if bv > 90 {
            atomic.StoreUint32(&ptr.win, 1000000000)
            atomic.StoreUint32(&ptr.visit, 10000000)
            atomic.StoreUint64(&ptr.stamp, GetStamp())
            return 2
        }
        b.Move(m1)
        m2, _ = b.Play()
    }

    if w = Simulation(b, sptr); w == 2 {
        if b.Now == now {
            atomic.StoreUint32(&ptr.win, 1000000000)
            atomic.StoreUint32(&ptr.visit, 10000000)
            atomic.StoreUint64(&ptr.stamp, GetStamp())
            b.UnMove(m2, m1)
            return 2
        }
        w = b.Now
    }
    b.UnMove(m2, m1)
    if b.Now != now {
        log.Panic("Simulation: unmove fail.")
    }
    if w == b.Now {
        atomic.AddUint32(&ptr.win, 1)
    }
    atomic.AddUint32(&ptr.visit, 1)
    atomic.StoreUint64(&ptr.stamp, GetStamp())
    return w
}

func SelectBest(b *board.Board, bptr *HashValue, final bool) (moves *board.Moves, ptr *HashValue,
    sum uint32, bestValue float64, err error) {

    var (
        visit, win uint32
        tmp, now   = float64(0), b.Now
        tptr       *HashValue
    )

    if bptr == nil {
        bptr = GetHashValue(NewHashKey(b), true)
    }
    sum = atomic.LoadUint32(&bptr.visit)
    logSum := math.Log(float64(sum))
    bestValue = -1e10

    ms, noC, err := b.GetMove()
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
            log.Fatal("SelectBest: ntms == 0.\n" + b.Draw())
        }
        ms = tms[:ntms]
    }

    for _, m := range ms {
        if m == nil {
            log.Println("SelectBest: nil move.\n" + b.Draw())
            continue
        }
        if err = b.Move(m); err != nil {
            log.Panic("SelectBest move fail.")
        }
        mm, _ := b.Play()

        if e := b.IsEnd(); e != 0 {
            if (e == -1 && b.Now == 0) || (e == 1 && b.Now == 1) {
                visit, win = sum, sum
            } else {
                visit, win = sum, 0
            }
            if (b.Now == now && win == visit) || (b.Now != now && win == 0) {
                bestValue, moves, ptr = 1e3, m, nil
                /*
                   if final {
                       log.Println("bestValue:", sum, visit, win,
                           float64(win)/float64(visit), b.Now != now, "-", bestValue, m)
                   }
                */
                b.UnMove(mm, m)
                return
            }
            tptr = nil
        } else {
            if tptr = GetHashValue(NewHashKey(b), false); tptr == nil {
                visit, win = 0, 0
            } else {
                visit = atomic.LoadUint32(&tptr.visit)
                if visit != 0 {
                    win = atomic.LoadUint32(&tptr.win)
                }
            }
        }

        if visit == 0 {
            tmp = rand.Float64() + 1.0
            if final {
                tmp = -1e9
                //log.Println("bestValue:", tmp, bestValue, m)
            }
        } else {
            tmp = float64(win) / float64(visit)
            if b.Now != now {
                tmp = 1 - tmp
            }
            if !final {
                tmp += ucb_C * math.Sqrt(logSum/float64(visit))
            } else {
                //log.Println("bestValue:", sum, visit, win,
                //float64(win)/float64(visit), b.Now != now, tmp, bestValue, len(ms))
            }
        }
        if tmp >= bestValue {
            bestValue = tmp
            moves = m
            ptr = tptr
        }
        b.UnMove(mm, m)
        if bestValue > 90 {
            return
        }
    }

    if moves == nil {
        log.Fatal("SelectBest fail.\n" + b.Draw())
    }
    return
}

func NewHashKey(b *board.Board) *HashKey {
    return &HashKey{uint32(b.H), uint32(b.V),
        uint16(b.S[0])<<6 + uint16(b.S[1])<<1 + uint16(b.Now)}
}

func GetHashValue(k *HashKey, write bool) *HashValue {
    idx := uint8((k.h ^ k.v ^ uint32(k.o)) % uint32(hashBlock))
    rwMutex[idx].RLock()
    if v, ok := hashTable[idx][*k]; ok {
        rwMutex[idx].RUnlock()
        return v
    } else if write {
        rwMutex[idx].RUnlock()

        v = &HashValue{0, 0, GetStamp()}
        rwMutex[idx].Lock()
        if int(hashSize) <= len(hashTable[idx]) {
            CleanHashTable(idx)
        }
        hashTable[idx][*k] = v
        rwMutex[idx].Unlock()
        return v
    }
    rwMutex[idx].RUnlock()
    return nil
}

func CleanHashTable(idx uint8) uint {
    now, count := GetStamp(), uint(0)
    for k, v := range hashTable[idx] {
        if v.stamp+hashExpire < now {
            delete(hashTable[idx], k)
            count++
        }
    }
    if count < hashClean {
        hashExpire = uint64(float32(hashExpire) * 0.99)
        log.Println("**CleanHashTable:", idx, count, hashExpire, "-")
    } else {
        hashExpire = uint64(float32(hashExpire) * 1.01)
        log.Println("**CleanHashTable:", idx, count, hashExpire, "+")
    }
    return count
}

func GetStamp() uint64 {
    return atomic.AddUint64(&timeStamp, 1)
}

func SetNumThread(n int) {
    numThread = n
}

func init() {
    rand.Seed(time.Now().Unix())
    numThread = runtime.NumCPU()
    for i := uint(0); i < hashBlock; i++ {
        hashTable[i] = make(map[HashKey]*HashValue, hashSize)
    }
}
