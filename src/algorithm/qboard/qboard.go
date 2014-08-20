/*********************************************************************************
*     File Name           :     qboard.go
*     Created By          :     YIMMON, yimmon.zhuang@gmail.com
*     Creation Date       :     [2014-05-18 10:54]
*     Last Modified       :     [2014-06-13 22:06]
*     Description         :
**********************************************************************************/

package qboard

import (
    "fmt"
    "math/rand"
    "time"
)

type IAlgorithm interface {
    GetName() string
    MakeMove(b *QBoard, timeout uint, verbose bool) (h, v int32, err error)
}

type QBoard struct {
    H, V      int32
    S         [2]int
    Now, Turn int

    edge    [60]Edge
    node    [7][7]Node
    linkSt  [60]Stack
    moveCh  [4]chan int
    info    Info
    history Stack
    parent  [60]int

    recordM  []int
    recordH  []int
    linkFunc []func(int, int) int
}

type Edge struct {
    idx  int
    node [2]*Node
    code int32
    st   Stack
}

type Node struct {
    x, y, degree int
    ground       bool
    edge         [4]*Edge
}

type LinkState struct {
    ty, length int
    edge, pe   [2]int
}

type EdgeState struct {
    cut, selected bool
    link, pos     int
}

type Info struct {
    Link                [11]int
    Loop                [5]int
    Halfheart, Fourlink int
    Point               [5]int
}

type Stack struct {
    data []interface{}
}

type History struct {
    split, union, change bool
    a, b                 int
    e                    []int
}

func (self *QBoard) Init() {
    self.H, self.V, self.Now, self.Turn = 0, 0, 0, 0
    self.S[0], self.S[1] = 0, 0
    for i := 0; i < 7; i++ {
        for j := 0; j < 7; j++ {
            self.node[i][j].x, self.node[i][j].y = i, j
            self.node[i][j].degree = 0
            if i == 0 || i == 6 || j == 0 || j == 6 {
                self.node[i][j].ground = true
            }
        }
    }
    for i := 0; i < 4; i++ {
        self.moveCh[i] = make(chan int, 300)
    }

    for _, i := range rand.Perm(60) {
        self.edge[i].idx, self.edge[i].code = i, 1<<uint(i%30)
        x, y := (i%30)/6+1, (i%30)%6+1
        if i < 30 {
            self.edge[i].node[0] = &self.node[x][y-1]
            self.edge[i].node[1] = &self.node[x][y]
            self.edge[i].node[0].edge[1] = &self.edge[i]
            self.edge[i].node[1].edge[3] = &self.edge[i]
        } else {
            self.edge[i].node[0] = &self.node[y-1][x]
            self.edge[i].node[1] = &self.node[y][x]
            self.edge[i].node[0].edge[2] = &self.edge[i]
            self.edge[i].node[1].edge[0] = &self.edge[i]
        }
        self.edge[i].node[0].degree++
        self.edge[i].node[1].degree++

        self.edge[i].st.Clear()
        self.linkSt[i].Clear()
        self.addHistory()
        self.newLinkState(i, 8, 0, i, i, i, -1)
        self.parent[i] = i
    }
    for _, i := range [2]int{0, 6} {
        for j := 1; j <= 5; j++ {
            self.node[i][j].degree += 3
            self.node[j][i].degree += 3
        }
    }
    self.info.Init()
    self.history.Clear()
    self.recordM, self.recordH = make([]int, 0, 60), make([]int, 0, 60)

    self.linkFuncInit()
}

func (self *QBoard) getMoveCheck(m int) bool {
    switch {
    case m == 0 && self.edge[30].hasSelected():
        return false
    case m == 5 && self.edge[54].hasSelected():
        return false
    case m == 24 && self.edge[35].hasSelected():
        return false
    case m == 29 && self.edge[59].hasSelected():
        return false
    }
    return true
}

func (self *QBoard) GetMove(ms []int) []int {
    if self.NumMove(0) != 0 {
        fmt.Println(self.Draw() + self.ShowLinks())
        panic("Call Play() before GetMove()!")
    }

    var vst [60]bool
    ms = ms[:0]
    for k := 1; k <= 3; k++ {
        for i := len(self.moveCh[k]); i > 0; i-- {
            m := <-self.moveCh[k]
            if !vst[m] && self.chanType(m) == k {
                vst[m] = true
                self.moveCh[k] <- m
                if k != 1 || self.getMoveCheck(m) {
                    ms = append(ms, m)
                }
            }
        }
    }
    return ms
}

func (self *QBoard) GetMove12(ms []int) []int {
    if self.NumMove(0) != 0 {
        panic("Call Play() before GetMove12()!")
    }

    var vst [60]bool
    ms = ms[:0]
    for k := 1; k <= 2; k++ {
        for i := len(self.moveCh[k]); i > 0; i-- {
            m := <-self.moveCh[k]
            if !vst[m] && self.chanType(m) == k {
                vst[m] = true
                self.moveCh[k] <- m
                if k != 1 || self.getMoveCheck(m) {
                    ms = append(ms, m)
                }
            }
        }
    }
    return ms
}

func (self *QBoard) GetMove12no44(ms []int) []int {
    if self.NumMove(0) != 0 {
        panic("Call Play() before GetMove12no44()!")
    }

    var vst [60]bool
    ms = ms[:0]
    for k := 1; k <= 2; k++ {
        for i := len(self.moveCh[k]); i > 0; i-- {
            m := <-self.moveCh[k]
            if !vst[m] && self.chanType(m) == k {
                vst[m] = true
                if self.edge[m].node[0].degree != 4 || self.edge[m].node[1].degree != 4 {
                    if k != 1 || self.getMoveCheck(m) {
                        ms = append(ms, m)
                    }
                }
                self.moveCh[k] <- m
            }
        }
    }
    return ms
}

func (self *QBoard) GetMove12only44(ms []int) []int {
    if self.NumMove(0) != 0 {
        panic("Call Play() before GetMove12only44()!")
    }

    var vst [60]bool
    ms = ms[:0]
    for i := len(self.moveCh[1]); i > 0; i-- {
        m := <-self.moveCh[1]
        if !vst[m] && self.chanType(m) == 1 {
            vst[m] = true
            if self.edge[m].node[0].degree == 4 && self.edge[m].node[1].degree == 4 {
                if self.getMoveCheck(m) {
                    ms = append(ms, m)
                }
            }
            self.moveCh[1] <- m
        }
    }
    return ms
}

func (self *QBoard) GetMove1(ms []int) []int {
    if self.NumMove(0) != 0 {
        panic("Call Play() before GetMove1()!")
    }

    var vst [60]bool
    ms = ms[:0]
    for i := len(self.moveCh[1]); i > 0; i-- {
        m := <-self.moveCh[1]
        if !vst[m] && self.chanType(m) == 1 {
            vst[m] = true
            self.moveCh[1] <- m
            if self.getMoveCheck(m) {
                ms = append(ms, m)
            }
        }
    }
    return ms
}

func (self *QBoard) GetMove2(ms []int) []int {
    if self.NumMove(0) != 0 {
        panic("Call Play() before GetMove2()!")
    }

    var vst [60]bool
    ms = ms[:0]
    for i := len(self.moveCh[2]); i > 0; i-- {
        m := <-self.moveCh[2]
        if !vst[m] && self.chanType(m) == 2 {
            vst[m] = true
            ms = append(ms, m)
            self.moveCh[2] <- m
        }
    }
    return ms
}

func (self *QBoard) GetMove3(ms []int) []int {
    if self.NumMove(0) != 0 {
        panic("Call Play() before GetMove3()!")
    }

    var vst [60]bool
    ms = ms[:0]
    for i := len(self.moveCh[3]); i > 0; i-- {
        m := <-self.moveCh[3]
        if !vst[m] && self.chanType(m) == 3 {
            vst[m] = true
            ms = append(ms, m)
            self.moveCh[3] <- m
        }
    }
    return ms
}

func (self *QBoard) NumMove(k int) (ret int) {
    var vst [60]bool
    for i := len(self.moveCh[k]); i > 0; i-- {
        m := <-self.moveCh[k]
        if !vst[m] && self.chanType(m) == k {
            vst[m] = true
            self.moveCh[k] <- m
            if k != 1 || self.getMoveCheck(m) {
                ret++
            }
        }
    }
    return
}

func (self *QBoard) NumMoveAll() (ret int) {
    var vst [60]bool
    for k := 0; k < 4; k++ {
        for i := len(self.moveCh[k]); i > 0; i-- {
            m := <-self.moveCh[k]
            if !vst[m] && self.chanType(m) == k {
                vst[m] = true
                self.moveCh[k] <- m
                if k != 1 || self.getMoveCheck(m) {
                    ret++
                }
            }
        }
    }
    return
}

func (self *QBoard) ImportantEdge(ms []int) []int {
    var vst [60]bool
    for _, m := range ms {
        vst[m] = true
    }
    for _, m := range [4]int{14, 15, 44, 45} {
        if !vst[m] && self.edge[m].hasSelected() {
            ms = append(ms, m)
        }
    }
    return ms
}

func (self *QBoard) EdgeDegree(m int) int {
    return self.edge[m].node[0].degree + self.edge[m].node[1].degree
}

func (self *QBoard) GetCMoves(ms []int) []int {
    ms = ms[:0]
    for k := 0; k <= 3; k++ {
        for i := len(self.moveCh[k]); i > 0; i-- {
            m := <-self.moveCh[k]
            if self.chanType(m) == k {
                if self.CanGetPointAfter(m) {
                    ms = append(ms, m)
                }
                self.moveCh[k] <- m
            }
        }
    }
    return ms

}

func (self *QBoard) Play(ms []int) []int {
    ms = ms[:0]
    for len(self.moveCh[0]) != 0 {
        m := <-self.moveCh[0]
        if self.chanType(m) == 0 {
            self.singleMove(m)
            ms = append(ms, m)
        }
    }
    return ms
}

func (self *QBoard) PlayRandomOne() (idx int) {
    if self.S[0]+self.S[1] == 25 {
        panic("PlayRandomOne: Game is over.")
    }

    ms := make([]int, 0, 60)
    ms = self.GetMove(ms)
    idx = ms[rand.Intn(len(ms))]
    self.singleMove(idx)
    return
}

func (self *QBoard) IsEnd() int {
    switch {
    case self.S[0] >= 13:
        return -1
    case self.S[1] >= 13:
        return 1
    default:
        return 0
    }
}

func NewQBoard(h, v int32, s0, s1, now, turn int) (b *QBoard) {
    b = new(QBoard)
    b.H, b.V = h, v
    b.S[0], b.S[1], b.Now, b.Turn = s0, s1, now, turn
    for i := 0; i < 7; i++ {
        for j := 0; j < 7; j++ {
            b.node[i][j].x, b.node[i][j].y = i, j
            b.node[i][j].degree = 0
            if i == 0 || i == 6 || j == 0 || j == 6 {
                b.node[i][j].ground = true
            }
        }
    }
    for i := 0; i < 4; i++ {
        b.moveCh[i] = make(chan int, 300)
    }

    for _, i := range rand.Perm(60) {
        b.parent[i] = i
        b.linkSt[i].Clear()
        b.edge[i].st.Clear()
        b.edge[i].idx, b.edge[i].code = i, 1<<uint(i%30)
        x, y := (i%30)/6+1, (i%30)%6+1
        if i < 30 {
            b.edge[i].node[0] = &b.node[x][y-1]
            b.edge[i].node[1] = &b.node[x][y]
            b.edge[i].node[0].edge[1] = &b.edge[i]
            b.edge[i].node[1].edge[3] = &b.edge[i]
        } else {
            b.edge[i].node[0] = &b.node[y-1][x]
            b.edge[i].node[1] = &b.node[y][x]
            b.edge[i].node[0].edge[2] = &b.edge[i]
            b.edge[i].node[1].edge[0] = &b.edge[i]
        }
        if (i < 30 && h&(1<<uint(i)) == 0) || (i >= 30 && v&(1<<uint(i-30)) == 0) {
            b.edge[i].node[0].degree++
            b.edge[i].node[1].degree++
            b.edge[i].addState(false, false, -1, -1)
        } else {
            b.edge[i].addState(true, false, -1, -1)
        }
    }
    for _, i := range [2]int{0, 6} {
        for j := 1; j <= 5; j++ {
            b.node[i][j].degree += 3
            b.node[j][i].degree += 3
        }
    }

    b.info.Init()
    b.info.Link[0] = 0

    var (
        vst [7][7]bool
        u   = func(a int) int {
            for b.parent[a] != a {
                a = b.parent[a]
            }
            return a
        }
        w   = func(aa, bb int) int {
            ua, ub := u(aa), u(bb)
            if ua != ub {
                b.parent[ub] = ua
            }
            return ua
        }
        o   = func(buf []int) int {
            if len(buf) == 1 {
                return buf[0]
            }
            p := buf[0]
            for _, e := range buf[1:] {
                p = w(p, e)
            }
            return p
        }
    )
    for i := 1; i <= 5; i++ {
        for j := 1; j <= 5; j++ {
            if b.node[i][j].degree == 1 {
                buf, end := b.walkToEnd(&(b.node[i][j]), &vst)
                if len(buf) == 1 {
                    r := o(buf[0])
                    if end[0].degree == 1 {
                        switch len(buf[0]) {
                        case 1: // O-O
                            if i*7+j > end[0].x*7+end[0].y {
                                b.addHistory()
                                b.newLinkState(r, 4, 0, buf[0][0], buf[0][0], buf[0][0], -1)
                            }
                        case 2: // O-O-O
                            b.addHistory()
                            b.newLinkState(r, 5, 1, buf[0][0], buf[0][1], MaxInt(buf[0][0], buf[0][1]), -1)
                        case 3: // O-O-O-O
                            b.addHistory()
                            b.newLinkState(r, 7, 2, buf[0][0], buf[0][2], MaxInt(buf[0][0], buf[0][2]), buf[0][1])
                        default: // O-O-...-O-O
                            b.addHistory()
                            b.newLinkState(r, 6, len(buf[0])-1, buf[0][0], buf[0][len(buf[0])-1],
                                MaxInt(buf[0][0], buf[0][len(buf[0])-1]), -1)
                        }
                    } else {
                        switch len(buf[0]) {
                        case 1: // O-X
                            b.addHistory()
                            b.newLinkState(r, 1, 0, buf[0][0], buf[0][0], buf[0][0], -1)
                        case 2: // O-O-X
                            b.addHistory()
                            b.newLinkState(r, 3, 1, buf[0][0], buf[0][1], buf[0][0], buf[0][1])
                        default: // O-O-...-O-X
                            b.addHistory()
                            b.newLinkState(r, 2, len(buf[0])-1, buf[0][0], buf[0][len(buf[0])-1], buf[0][0], -1)
                        }
                    }
                }
            }
        }
    }

    for i := 0; i < 7; i++ {
        for j := 0; j < 7; j++ {
            if b.node[i][j].degree >= 3 {
                buf, end := b.walkToEnd(&(b.node[i][j]), &vst)
                for k, e := range buf {
                    r := o(e)
                    if end[k].degree == 1 {
                        switch len(e) {
                        case 1: // X-O
                            // pass
                        case 2: // X-O-O
                            b.addHistory()
                            b.newLinkState(r, 3, 1, e[1], e[0], e[1], e[0])
                        default: // X-O-O-...-O
                            b.addHistory()
                            b.newLinkState(r, 2, len(e)-1, e[len(e)-1], e[0], e[len(e)-1], -1)
                        }
                    } else {
                        switch len(e) {
                        case 1: // X-X
                            if i*7+j > end[k].x*7+end[k].y {
                                b.addHistory()
                                b.newLinkState(r, 8, 0, e[0], e[0], e[0], -1)
                            }
                        case 2: // X-O-X
                            b.addHistory()
                            b.newLinkState(r, 9, 1, e[0], e[1], MaxInt(e[0], e[1]), -1)
                        case 3: // X-O-O-X
                            b.addHistory()
                            b.newLinkState(r, 10, 2, e[0], e[2], MaxInt(e[0], e[2]), e[1])
                        default: // X-O-...-O-X
                            b.addHistory()
                            b.newLinkState(r, 11, len(e)-1, e[0], e[len(e)-1], MaxInt(e[0], e[len(e)-1]), -1)
                        }
                    }
                }
            }
        }
    }

    for i := 1; i <= 5; i++ {
        for j := 1; j <= 5; j++ {
            if b.node[i][j].degree == 2 && !vst[i][j] {
                e := b.walkLoop(&(b.node[i][j]), &vst)
                r := o(e)
                b.addHistory()
                b.newLinkState(r, 12, len(e), e[0], e[len(e)-1], b.maxEdgeInLoop(e[0], e[len(e)-1]), -1)
            }
        }
    }

    for i, _ := range b.info.Point {
        b.info.Point[i] = 0
    }
    for i := 1; i <= 5; i++ {
        for j := 1; j <= 5; j++ {
            b.info.Point[b.node[i][j].degree]++
        }
    }
    b.info.Point[0] = 0
    for _, i := range [2]int{0, 6} {
        for j := 1; j <= 5; j++ {
            if b.node[i][j].degree == 4 {
                b.info.Point[0]++
            }
            if b.node[j][i].degree == 4 {
                b.info.Point[0]++
            }
        }
    }

    for i := 0; i < 4; i++ {
        b.NumMove(i)
    }
    b.linkFuncInit()
    b.history.Clear()
    b.recordM, b.recordH = make([]int, 0, 60), make([]int, 0, 60)

    return
}

func (self *QBoard) Copy() *QBoard {
    b := new(QBoard)
    self.CopyTo(b)
    return b
}

func (self *QBoard) CopyTo(b *QBoard) {
    b.H, b.V = self.H, self.V
    b.S[0], b.S[1], b.Now, b.Turn = self.S[0], self.S[1], self.Now, self.Turn

    for i := 0; i < 60; i++ {
        b.edge[i].idx, b.edge[i].code = self.edge[i].idx, self.edge[i].code
        b.edge[i].st.CopyFrom(&self.edge[i].st)
        x, y := (i%30)/6+1, (i%30)%6+1
        if i < 30 {
            b.edge[i].node[0] = &b.node[x][y-1]
            b.edge[i].node[1] = &b.node[x][y]
            b.edge[i].node[0].edge[1] = &b.edge[i]
            b.edge[i].node[1].edge[3] = &b.edge[i]
        } else {
            b.edge[i].node[0] = &b.node[y-1][x]
            b.edge[i].node[1] = &b.node[y][x]
            b.edge[i].node[0].edge[2] = &b.edge[i]
            b.edge[i].node[1].edge[0] = &b.edge[i]
        }
        b.linkSt[i].CopyFrom(&self.linkSt[i])
        b.parent[i] = self.parent[i]
    }
    for i := 0; i < 7; i++ {
        for j := 0; j < 7; j++ {
            b.node[i][j].x, b.node[i][j].y = self.node[i][j].x, self.node[i][j].y
            b.node[i][j].degree, b.node[i][j].ground = self.node[i][j].degree, self.node[i][j].ground
        }
    }

    for k := 0; k < 4; k++ {
        if b.moveCh[k] == nil {
            b.moveCh[k] = make(chan int, 300)
        }
        for len(b.moveCh[k]) != 0 {
            <-b.moveCh[k]
        }
        for i := len(self.moveCh[k]); i > 0; i-- {
            m := <-self.moveCh[k]
            if self.chanType(m) == k {
                b.moveCh[k] <- m
                self.moveCh[k] <- m
            }
        }
    }
    b.info = self.info
    b.history.CopyFrom(&self.history)
    if b.recordM == nil {
        b.recordM, b.recordH = make([]int, len(self.recordM), 60), make([]int, len(self.recordH), 60)
    } else {
        b.recordM = b.recordM[:len(self.recordM)]
        b.recordH = b.recordH[:len(self.recordH)]
    }
    for i, _ := range self.recordM {
        b.recordM[i], b.recordH[i] = self.recordM[i], self.recordH[i]
    }
    if b.linkFunc == nil {
        b.linkFuncInit()
    }

    return
}

/* 入口s的度不能为2 */
func (self *QBoard) walkToEnd(s *Node, vst *[7][7]bool) (buf [][]int, end []*Node) {
    buf, end = make([][]int, 0, s.degree), make([]*Node, 0, s.degree)
    for _, e := range s.edge {
        if e != nil && e.hasCut() == false {
            p := 0
            if e.node[p] == s {
                p = 1
            }
            if !vst[e.node[p].x][e.node[p].y] {
                ee, pp, cur := e, e.node[p], len(buf)
                buf = append(buf, make([]int, 1, 8))
                buf[cur][0] = e.idx
                for pp.degree == 2 {
                    vst[pp.x][pp.y] = true
                    for _, eee := range pp.edge {
                        if eee.hasCut() == false && eee.idx != ee.idx {
                            ee = eee
                            if eee.node[0] == pp {
                                pp = eee.node[1]
                            } else {
                                pp = eee.node[0]
                            }
                            break
                        }
                    }
                    buf[cur] = append(buf[cur], ee.idx)
                }
                end = append(end, pp)
            }
        }
    }
    return
}

/* 入口s必是环中一点 */
func (self *QBoard) walkLoop(s *Node, vst *[7][7]bool) (es []int) {
    es = make([]int, 0, 12)
    for _, e := range s.edge {
        if e.hasCut() == false {
            ee, pp := e, s
            for !vst[pp.x][pp.y] {
                es = append(es, ee.idx)
                vst[pp.x][pp.y] = true
                for _, eee := range pp.edge {
                    if eee.hasCut() == false && eee.idx != ee.idx {
                        ee = eee
                        if eee.node[0] == pp {
                            pp = eee.node[1]
                        } else {
                            pp = eee.node[0]
                        }
                        break
                    }
                }
            }
            break
        }
    }
    return
}

func (self *QBoard) CanGetPointAfter(idx int) bool {
    if self.edge[idx].node[0].degree == 1 || self.edge[idx].node[1].degree == 1 {
        return true
    }
    return false
}

func (self *QBoard) MoveHV(h, v int32) (moves []int) {
    ms := make([]int, 0, 16)
    for i, o := range [2]int32{h, v} {
        for j := uint(0); (1 << j) <= o; j++ {
            if (1<<j)&o != 0 {
                ms = append(ms, i*30+int(j))
            }
        }
    }
    moves = make([]int, 0, len(ms))
    for {
        for i, m := range ms {
            if m != -1 && self.CanGetPointAfter(m) {
                self.Move(m)
                moves = append(moves, m)
                ms[i] = -1
            }
        }
        tmpNo, tmpYes := 0, 0
        for _, m := range ms {
            if m != -1 {
                if self.CanGetPointAfter(m) {
                    tmpYes++
                } else {
                    tmpNo++
                }
            }
        }
        if tmpYes == 0 && tmpNo == 0 {
            return
        } else if tmpYes == 0 && tmpNo == 1 {
            for _, m := range ms {
                if m != -1 {
                    self.Move(m)
                    moves = append(moves, m)
                    return
                }
            }
        } else if tmpYes == 0 {
            fmt.Println(self.Draw())
            moves = nil
            return
        }
    }
}

func (self *QBoard) ShowLinks() (ret string) {
    for _, l := range self.linkSt {
        if !l.Empty() {
            ls, _ := l.Top().(*LinkState)
            if ls.ty != -1 {
                ret += fmt.Sprintf("ty: %d, length: %d, edge: %d %d, pe: %d %d\n",
                    ls.ty, ls.length, ls.edge[0], ls.edge[1], ls.pe[0], ls.pe[1])
            }
        }
    }
    return
}

func (self *QBoard) Draw() string {
    var layout [11][12]byte

    for i := 0; i < 11; i++ {
        for j := 0; j < 11; j++ {
            layout[i][j] = byte(' ')
        }
        layout[i][11] = byte('\n')
    }

    for i := 0; i < 5; i++ {
        for j := 0; j < 6; j++ {
            if self.edge[i*6+j].hasCut() {
                layout[i*2+1][j*2] = byte('|')
            }
        }
    }
    for i := 0; i < 5; i++ {
        for j := 0; j < 6; j++ {
            layout[j*2][i*2] = byte('.')
            layout[j*2][i*2+2] = byte('.')
            if self.edge[30+i*6+j].hasCut() {
                layout[j*2][i*2+1] = byte('_')
            }
        }
    }

    str := ""
    for i := 0; i < 11; i++ {
        str += " " + string(layout[i][:])
    }
    str = fmt.Sprintf("%sTurn=%d, S[0]=%d, S[1]=%d, Now=%d\nH=0x%08x, V=0x%08x\n",
        str, self.Turn, self.S[0], self.S[1], self.Now, self.H, self.V)
    return str
}

func (self *QBoard) LinkType(e int) int {
    for e != self.parent[e] {
        e = self.parent[e]
    }
    lst, _ := self.linkSt[e].Top().(*LinkState)
    return lst.ty
}

func (self *QBoard) LinkLength(e int) int {
    for e != self.parent[e] {
        e = self.parent[e]
    }
    lst, _ := self.linkSt[e].Top().(*LinkState)
    return lst.length
}

func Moves2HV(idxs ...int) (h, v int32) {
    for _, idx := range idxs {
        if idx < 30 {
            h |= 1 << uint(idx)
        } else {
            v |= 1 << uint(idx-30)
        }
    }
    return
}
func (self *QBoard) GetInfo() *Info {
    return &(self.info)
}

func (self *QBoard) singleMove(idx int) {
    est, _ := self.edge[idx].st.Top().(*EdgeState)
    if est.cut || !est.selected {
        fmt.Println(self.Draw() + self.ShowLinks())
        fmt.Println("idx:", idx, "est:", est)
        panic("Move unselected.")
    }
    self.cut(idx)
    self.recordM = append(self.recordM, idx)
    lst := self.linkSt[est.link].Top().(*LinkState)
    self.recordH = append(self.recordH, self.linkFunc[lst.ty](est.link, est.pos))
}

func (self *QBoard) Move(ms ...int) {
    for _, m := range ms {
        self.singleMove(m)
    }
}

func (self *QBoard) singleUnMove(idx int) {
    if idx != self.recordM[len(self.recordM)-1] {
        panic("Invalid unmove.")
    }
    for i := self.recordH[len(self.recordH)-1]; i > 0; i-- {
        h, _ := self.history.Pop().(*History)
        switch {
        case h.change:
            self.removeLinkState(h.a)
        case h.union:
            self.parent[h.a], self.parent[h.b] = h.a, h.b
            self.removeLinkState(h.a)
            self.removeLinkState(h.b)
        case h.split:
            self.parent[h.a] = h.b
            self.removeLinkState(h.a)
        default:
            panic("Invalid history.")
        }
        for _, e := range h.e {
            self.edge[e].removeState()
        }
    }

    e := &self.edge[idx]
    if (idx < 30 && self.H&e.code == 0) || (idx >= 30 && self.V&e.code == 0) {
        panic("Uncut invalid edge")
    }
    if idx < 30 {
        self.H &= ^e.code
    } else {
        self.V &= ^e.code
    }
    for i := 0; i < 2; i++ {
        if e.node[i].degree++; e.node[i].degree == 1 {
            self.S[self.Now]--
        }
    }
    if e.node[0].degree != 1 && e.node[1].degree != 1 {
        self.Now ^= 1
        self.Turn--
    }
    e.removeState()
    for i := 0; i < 2; i++ {
        if e.node[i].x >= 1 && e.node[i].x <= 5 && e.node[i].y >= 1 && e.node[i].y <= 5 {
            if e.node[i].degree != 1 {
                self.info.Point[e.node[i].degree-1]--
            }
            self.info.Point[e.node[i].degree]++
        } else {
            self.info.Point[0]++
        }
    }

    self.recordH = self.recordH[:len(self.recordH)-1]
    self.recordM = self.recordM[:len(self.recordM)-1]
}

func (self *QBoard) UnMove(ms ...int) {
    for i := len(ms) - 1; i >= 0; i-- {
        self.singleUnMove(ms[i])
    }
}

func (self *Stack) Push(v interface{}) {
    if self.data == nil {
        self.data = make([]interface{}, 0, 128)
    }
    self.data = append(self.data, v)
}

func (self *Stack) Top() interface{} {
    return self.data[len(self.data)-1]
}

func (self *Stack) Pop() interface{} {
    if len(self.data) == 0 {
        panic("Empty stack cannot pop.")
    }
    ret := self.data[len(self.data)-1]
    self.data = self.data[:len(self.data)-1]
    return ret
}

func (self *Stack) Empty() bool {
    return len(self.data) == 0
}

func (self *Stack) Clear() {
    self.data = self.data[:0]
}

func (self *Stack) TopTop() interface{} {
    if len(self.data) == cap(self.data) {
        return nil
    }
    return self.data[len(self.data) : len(self.data)+1][0]
}

func (self *Stack) Expand() {
    self.data = self.data[:len(self.data)+1]
}

func (self *Stack) CopyFrom(st *Stack) {
    if cap(self.data) < len(st.data) {
        self.data = make([]interface{}, len(st.data), cap(st.data))
    } else {
        self.data = self.data[:len(st.data)]
    }
    for i, v := range st.data {
        switch v.(type) {
        case *LinkState:
            ls, ok := self.data[i].(*LinkState)
            if !ok {
                ls = new(LinkState)
            }
            ols, _ := v.(*LinkState)
            *ls = *ols
            self.data[i] = ls
        case *EdgeState:
            es, ok := self.data[i].(*EdgeState)
            if !ok {
                es = new(EdgeState)
            }
            oes, _ := v.(*EdgeState)
            *es = *oes
            self.data[i] = es
        case *History:
            h, ok := self.data[i].(*History)
            if !ok {
                h = new(History)
            }
            oh, _ := v.(*History)
            *h = *oh
            self.data[i] = h
        default:
            panic("CopyFrom error.")
        }
    }
}

func (self *Info) Init() {
    self.Halfheart, self.Fourlink = 0, 0
    for i := 0; i < 11; i++ {
        self.Link[i] = 0
    }
    for i := 0; i < 5; i++ {
        self.Loop[i] = 0
        self.Point[i] = 0
    }
    self.Link[0] = 60
    self.Point[0], self.Point[4] = 20, 25
}

func (self *Edge) addState(cut, selected bool, link, pos int) {
    es, ok := self.st.TopTop().(*EdgeState)
    if ok {
        es.cut, es.selected = cut, selected
        es.link, es.pos = link, pos
        self.st.Expand()
    } else {
        es = &EdgeState{cut: cut, selected: selected, link: link, pos: pos}
        self.st.Push(es)
    }
}

func (self *Edge) removeState() {
    self.st.Pop()
}

func (self *QBoard) cut(idx int) {
    e := &self.edge[idx]
    if (idx < 30 && self.H&e.code != 0) || (idx >= 30 && self.V&e.code != 0) {
        fmt.Println(self.Draw(), self.H&e.code, self.V&e.code, e, idx)
        es, _ := e.st.Top().(*EdgeState)
        fmt.Println(es)
        ls, _ := self.linkSt[es.link].Top().(*LinkState)
        fmt.Println(ls)
        fmt.Println(self.recordM)
        panic("Cut invalid edge")
    }
    if idx < 30 {
        self.H |= e.code
    } else {
        self.V |= e.code
    }
    for i := 0; i < 2; i++ {
        if e.node[i].degree--; e.node[i].degree == 0 {
            self.S[self.Now]++
        }
    }
    if e.node[0].degree != 0 && e.node[1].degree != 0 {
        self.Now ^= 1
        self.Turn++
    }
    e.addState(true, false, -1, -1)
    for i := 0; i < 2; i++ {
        if e.node[i].x >= 1 && e.node[i].x <= 5 && e.node[i].y >= 1 && e.node[i].y <= 5 {
            self.info.Point[e.node[i].degree+1]--
            if e.node[i].degree != 0 {
                self.info.Point[e.node[i].degree]++
            }
        } else {
            self.info.Point[0]--
        }
    }
}

func (self *QBoard) chanType(idx int) int {
    es, _ := self.edge[idx].st.Top().(*EdgeState)
    if !es.cut && es.selected {
        ls, _ := self.linkSt[es.link].Top().(*LinkState)
        switch {
        case ls.ty <= 2 || (ls.ty >= 4 && ls.ty <= 6):
            return 0
        case ls.ty == 8 || ls.ty == 9 || (ls.ty == 10 && es.pos == 1):
            return 1
        case ls.ty == 3 || ls.ty == 7:
            return 2
        default:
            return 3
        }
    }
    return -1
}

func (self *QBoard) addHistory() *History {
    h, ok := self.history.TopTop().(*History)
    if ok {
        h.split, h.union, h.change = false, false, false
        if h.e != nil {
            h.e = h.e[:0]
        }
        self.history.Expand()
    } else {
        h = new(History)
        self.history.Push(h)
    }
    return h
}

func (self *QBoard) addLinkState(link, ty, length, e0, e1, p0, p1 int) {
    ols, _ := self.linkSt[link].Top().(*LinkState)
    self.info.add(ols, -1)
    ls, ok := self.linkSt[link].TopTop().(*LinkState)
    if ok {
        ls.ty, ls.length = ty, length
        ls.edge[0], ls.edge[1] = e0, e1
        ls.pe[0], ls.pe[1] = p0, p1
        self.linkSt[link].Expand()
    } else {
        ls = &LinkState{ty: ty, length: length, edge: [2]int{e0, e1}, pe: [2]int{p0, p1}}
        self.linkSt[link].Push(ls)
    }
    if ls.ty != -1 {
        self.info.add(ls, 1)
    }
    h, _ := self.history.Top().(*History)
    if h.e == nil {
        h.e = make([]int, 0, 6)
    }
    for i := 0; i < 2; i++ {
        if ols.pe[i] >= 0 && self.edge[ols.pe[i]].hasCut() == false {
            h.e = append(h.e, ols.pe[i])
            self.edge[ols.pe[i]].addState(false, false, -1, -1)
        }
    }
    for i := 0; i < 2; i++ {
        if ls.pe[i] >= 0 {
            h.e = append(h.e, ls.pe[i])
            self.edge[ls.pe[i]].addState(false, true, link, i)
            self.addMove(ls.pe[i], ls.ty, i)
        }
    }
}

func (self *QBoard) newLinkState(link, ty, length, e0, e1, p0, p1 int) {
    ls, ok := self.linkSt[link].TopTop().(*LinkState)
    if ok {
        ls.ty, ls.length = ty, length
        ls.edge[0], ls.edge[1] = e0, e1
        ls.pe[0], ls.pe[1] = p0, p1
        self.linkSt[link].Expand()
    } else {
        ls = &LinkState{ty: ty, length: length, edge: [2]int{e0, e1}, pe: [2]int{p0, p1}}
        self.linkSt[link].Push(ls)
    }

    if ls.ty != -1 {
        self.info.add(ls, 1)
    }
    h, _ := self.history.Top().(*History)
    if h.e == nil {
        h.e = make([]int, 0, 6)
    }
    for i := 0; i < 2; i++ {
        if ls.pe[i] >= 0 {
            h.e = append(h.e, ls.pe[i])
            self.edge[ls.pe[i]].addState(false, true, link, i)
            self.addMove(ls.pe[i], ls.ty, i)
        }
    }
}

func (self *QBoard) removeLinkState(link int) {
    ols, _ := self.linkSt[link].Pop().(*LinkState)
    if ols.ty != -1 {
        self.info.add(ols, -1)
    }
    if !self.linkSt[link].Empty() {
        ls, _ := self.linkSt[link].Top().(*LinkState)
        self.info.add(ls, 1)
        for i := 0; i < 2; i++ {
            if ls.pe[i] >= 0 {
                self.addMove(ls.pe[i], ls.ty, i)
            }
        }
    }
}

func (self *Edge) getMaxEnd() *Node {
    if self.node[0].degree >= self.node[1].degree {
        return self.node[0]
    } else {
        return self.node[1]
    }
}

// node 必须是2度点
func (self *QBoard) getTowEdge(node *Node) (a, b int) {
    if node.degree != 2 {
        panic("getTowEdge: node.degree != 2")
    }
    for i := 0; i < 4; i++ {
        if node.edge[i].hasCut() == false {
            a, b = node.edge[i].idx, a
        }
    }
    return
}

func (self *QBoard) getLinkIdx(eidx int) int {
    if self.edge[eidx].hasSelected() {
        return self.edge[eidx].getLink()
    }
    for self.parent[eidx] != eidx {
        eidx = self.parent[eidx]
    }
    return eidx
}

func (self *Edge) hasCut() bool {
    es, _ := self.st.Top().(*EdgeState)
    return es.cut
}

func (self *Edge) hasSelected() bool {
    es, _ := self.st.Top().(*EdgeState)
    return !es.cut && es.selected
}

// 必须是selected的边
func (self *Edge) getLink() int {
    es, _ := self.st.Top().(*EdgeState)
    return es.link
}

// self 需为1度点
func (self *Node) getEdge() int {
    for i := 0; i < 4; i++ {
        if self.edge[i].hasCut() == false {
            return self.edge[i].idx
        }
    }
    return -1
}

func getAnotherEnd(x int, es []int) int {
    for _, e := range es {
        if e != x {
            return e
        }
    }
    return x
}

func (self *QBoard) union(ea, eb, a, b int) {
    if a == b {
        panic("a == b")
    }
    var tmpls LinkState
    lsa, _ := self.linkSt[a].Top().(*LinkState)
    lsb, _ := self.linkSt[b].Top().(*LinkState)
    if lsa.ty > lsb.ty {
        a, b = b, a
        ea, eb = eb, ea
        lsa, lsb = lsb, lsa
    }
    ls := &tmpls
    ls.edge[0], ls.edge[1] = getAnotherEnd(ea, lsa.edge[:]), getAnotherEnd(eb, lsb.edge[:])
    switch {
    case lsa.ty == 1 && lsb.ty == 1:
        ls.ty, ls.length = 5, 1
    case lsa.ty == 1 && lsb.ty == 2:
        ls.ty, ls.length = 6, lsb.length+1
    case lsa.ty == 1 && lsb.ty == 3:
        ls.ty, ls.length = 7, 2
    case lsa.ty == 1 && lsb.ty == 8:
        ls.ty, ls.length = 3, 1
    case lsa.ty == 1:
        ls.ty, ls.length = 2, lsb.length+1
    case lsa.ty == 2 && lsb.ty <= 3:
        ls.ty, ls.length = 6, lsa.length+lsb.length+1
    case lsa.ty == 2:
        ls.ty, ls.length = 2, lsa.length+lsb.length+1
    case lsa.ty == 3 && lsb.ty == 3:
        ls.ty, ls.length = 6, 3
    case lsa.ty == 3:
        ls.ty, ls.length = 2, lsa.length+lsb.length+1
    case lsa.ty == 8 && lsb.ty == 8:
        ls.ty, ls.length = 9, 1
    case lsa.ty == 8 && lsb.ty == 9:
        ls.ty, ls.length = 10, 2
    case lsa.ty == 8:
        ls.ty, ls.length = 11, lsb.length+1
    default:
        ls.ty, ls.length = 11, lsa.length+lsb.length+1
    }
    self.setPE(ls, ea, eb)
    if a < b {
        a, b = b, a
        ea, eb = eb, ea
        lsa, lsb = lsb, lsa
    }
    self.parent[b] = a
    self.addLinkState(b, -1, -1, -1, -1, -1, -1)
    self.addLinkState(a, ls.ty, ls.length, ls.edge[0], ls.edge[1], ls.pe[0], ls.pe[1])
}

func (self *QBoard) addMove(m, ty, pos int) {
    switch {
    case ty <= 2 || (ty >= 4 && ty <= 6):
        self.moveCh[0] <- m
    case ty == 8 || ty == 9 || (ty == 10 && pos == 1):
        self.moveCh[1] <- m
    case ty == 3 || ty == 7:
        self.moveCh[2] <- m
    default:
        self.moveCh[3] <- m
    }
}

func (self *QBoard) setPE(ls *LinkState, ea, eb int) {
    switch {
    case ls.ty == 3:
        ls.pe[0], ls.pe[1] = ls.edge[0], ls.edge[1]
        if self.edge[ls.pe[0]].node[0].degree >= 3 || self.edge[ls.pe[0]].node[1].degree >= 3 {
            ls.pe[0], ls.pe[1] = ls.edge[1], ls.edge[0]
        }
    case ls.ty == 7 || ls.ty == 10:
        x, y := ls.edge[0], ls.edge[1]
        if x > y {
            x, y = y, x
        }
        if x == ea {
            ls.pe[0], ls.pe[1] = y, eb
        } else if x == eb {
            ls.pe[0], ls.pe[1] = y, ea
        } else {
            if y == eb {
                ls.pe[0], ls.pe[1] = eb, ea
            } else {
                ls.pe[0], ls.pe[1] = ea, eb
            }
        }
    case ls.ty == 2:
        ls.pe[0], ls.pe[1] = ls.edge[0], -1
        if self.edge[ls.pe[0]].node[0].degree >= 3 || self.edge[ls.pe[0]].node[1].degree >= 3 {
            ls.pe[0] = ls.edge[1]
        }
    default:
        ls.pe[0], ls.pe[1] = ls.edge[0], -1
        if ls.pe[0] < ls.edge[1] {
            ls.pe[0] = ls.edge[1]
        }
    }
}

func (self *QBoard) maxEdgeInLoop(e0, e1 int) int {
    var ret = MaxInt(e0, e1)
    e, fx, p := e0, 0, 0
    for {
        if fx == 1 || fx == 2 {
            p = 1
        } else {
            p = 0
        }
        for i, ee := range self.edge[e].node[p].edge {
            if ee.hasCut() == false && ee.idx != e {
                e, fx = ee.idx, i
                break
            }
        }
        if e == e0 {
            break
        }
        ret = MaxInt(ret, e)
    }
    return ret
}

func MaxInt(a, b int) int {
    if a > b {
        return a
    }
    return b
}

func (self *Info) add(ls *LinkState, inc int) {
    switch {
    case ls.ty >= 8 && ls.ty <= 11:
        ll := ls.length
        if ll > 10 {
            ll = 10
        }
        self.Link[ll] += inc
    case ls.ty == 3:
        self.Halfheart += inc
    case ls.ty == 7:
        self.Fourlink += inc
    case ls.ty == 12:
        ll := (ls.length - 4) >> 1
        if ll > 4 {
            ll = 4
        }
        self.Loop[ll] += inc
    }
}

// 返回被切断边的邻边
func (self *QBoard) nextEdge(idx int) int {
    p := 0
    if self.edge[idx].node[p].degree != 1 {
        p ^= 1
    }
    for _, e := range self.edge[idx].node[p].edge {
        if e.hasCut() == false {
            return e.idx
        }
    }
    return -1
}

// 返回未被切断边的邻边
func (self *QBoard) neightborEdge(idx int) int {
    p := 0
    if self.edge[idx].node[p].degree == 1 {
        p ^= 1
    }
    for _, e := range self.edge[idx].node[p].edge {
        if e.hasCut() == false && e.idx != idx {
            return e.idx
        }
    }
    return -1
}

func (self *QBoard) linkFuncInit() {
    self.linkFunc = make([]func(int, int) int, 13)

    self.linkFunc[1] = func(link, pos int) (nh int) {
        if pos != 0 {
            panic("linkFunc 1: pos != 0")
        }
        lst, _ := self.linkSt[link].Top().(*LinkState)
        h := self.addHistory()
        nh++
        h.change, h.a = true, link
        self.addLinkState(link, -1, -1, -1, -1, -1, -1)

        np := self.edge[lst.pe[pos]].getMaxEnd()
        if np.degree == 2 {
            h = self.addHistory()
            nh++
            ea, eb := self.getTowEdge(np)
            h.a, h.b = self.getLinkIdx(ea), self.getLinkIdx(eb)
            if h.a == h.b {
                h.change = true
                lsa, _ := self.linkSt[h.a].Top().(*LinkState)
                self.addLinkState(h.a, 12, lsa.length+1, lsa.edge[0], lsa.edge[1],
                    self.maxEdgeInLoop(lsa.edge[0], lsa.edge[1]), -1)
            } else {
                h.union = true
                self.union(ea, eb, h.a, h.b)
            }
        }
        return
    }

    self.linkFunc[2] = func(link, pos int) (nh int) {
        if pos != 0 {
            panic("linkFunc 2: pos != 0")
        }
        lst, _ := self.linkSt[link].Top().(*LinkState)
        h := self.addHistory()
        nh++
        h.change, h.a = true, link
        p := 0
        if self.edge[lst.edge[p]].hasCut() == false {
            p = 1
        }
        q := self.nextEdge(lst.edge[p])
        if lst.length > 2 {
            self.addLinkState(link, 2, lst.length-1, q, lst.edge[p^1], q, -1)
        } else {
            self.addLinkState(link, 3, 1, q, lst.edge[p^1], q, lst.edge[p^1])
        }
        return
    }

    self.linkFunc[3] = func(link, pos int) (nh int) {
        lst, _ := self.linkSt[link].Top().(*LinkState)
        h := self.addHistory()
        nh++
        if pos == 0 {
            h.change, h.a = true, link
            self.addLinkState(link, 1, 0, lst.pe[1], lst.pe[1], lst.pe[1], -1)
        } else {
            h.change, h.a = true, link
            self.addLinkState(link, 4, 0, lst.pe[0], lst.pe[0], lst.pe[0], -1)
            np := self.edge[lst.pe[pos]].getMaxEnd()
            if np.degree == 2 {
                h = self.addHistory()
                nh++
                ea, eb := self.getTowEdge(np)
                h.a, h.b = self.getLinkIdx(ea), self.getLinkIdx(eb)
                if h.a == h.b {
                    h.change = true
                    lsa, _ := self.linkSt[h.a].Top().(*LinkState)
                    self.addLinkState(h.a, 12, lsa.length+1, lsa.edge[0], lsa.edge[1],
                        self.maxEdgeInLoop(lsa.edge[0], lsa.edge[1]), -1)
                } else {
                    h.union = true
                    self.union(ea, eb, h.a, h.b)
                }
            }
        }
        return
    }

    self.linkFunc[4] = func(link, pos int) (nh int) {
        if pos != 0 {
            panic("linkFunc 4: pos != 0")
        }
        h := self.addHistory()
        nh++
        h.change, h.a = true, link
        self.addLinkState(link, -1, -1, -1, -1, -1, -1)
        return
    }

    self.linkFunc[5] = func(link, pos int) (nh int) {
        if pos != 0 {
            panic("linkFunc 5: pos != 0")
        }
        lst, _ := self.linkSt[link].Top().(*LinkState)
        h := self.addHistory()
        nh++
        h.change, h.a = true, link
        p := 0
        if self.edge[lst.edge[p]].hasCut() == false {
            p = 1
        }
        q := self.nextEdge(lst.edge[p])
        self.addLinkState(link, 4, 0, q, q, q, -1)
        return
    }

    self.linkFunc[6] = func(link, pos int) (nh int) {
        if pos != 0 {
            panic("linkFunc 6: pos != 0")
        }
        lst, _ := self.linkSt[link].Top().(*LinkState)
        h := self.addHistory()
        nh++
        h.change, h.a = true, link
        p := 0
        if self.edge[lst.edge[p]].hasCut() == false {
            p = 1
        }
        q := self.nextEdge(lst.edge[p])
        if lst.length > 3 {
            if q > lst.edge[p^1] {
                self.addLinkState(link, 6, lst.length-1, q, lst.edge[p^1], q, -1)
            } else {
                self.addLinkState(link, 6, lst.length-1, q, lst.edge[p^1], lst.edge[p^1], -1)
            }
        } else {
            qq := q
            if qq < lst.edge[p^1] {
                qq = lst.edge[p^1]
            }
            self.addLinkState(link, 7, 2, q, lst.edge[p^1], qq, self.neightborEdge(qq))
        }
        return
    }

    self.linkFunc[7] = func(link, pos int) (nh int) {
        lst, _ := self.linkSt[link].Top().(*LinkState)
        h := self.addHistory()
        nh++
        if pos == 0 {
            h.change, h.a = true, link
            q := lst.edge[0]
            if q == lst.pe[pos] {
                q = lst.edge[1]
            }
            o := q
            if o < lst.pe[1] {
                o = lst.pe[1]
            }
            self.addLinkState(link, 5, 1, lst.pe[1], q, o, -1)
        } else {
            h.change, h.a = true, link
            self.addLinkState(link, -1, -1, -1, -1, -1, -1)
            for i := 0; i < 2; i++ {
                h = self.addHistory()
                nh++
                h.split = true
                h.a, h.b = lst.edge[i], self.parent[lst.edge[i]]
                self.newLinkState(h.a, 4, 0, lst.edge[i], lst.edge[i], lst.edge[i], -1)
            }
        }
        return
    }

    self.linkFunc[8] = func(link, pos int) (nh int) {
        if pos != 0 {
            panic("linkFunc 8: pos != 0")
        }
        lst, _ := self.linkSt[link].Top().(*LinkState)
        h := self.addHistory()
        nh++
        h.change, h.a = true, link
        self.addLinkState(link, -1, -1, -1, -1, -1, -1)
        for i := 0; i < 2; i++ {
            np := self.edge[lst.pe[pos]].node[i]
            if np.degree == 2 {
                h = self.addHistory()
                nh++
                ea, eb := self.getTowEdge(np)
                h.a, h.b = self.getLinkIdx(ea), self.getLinkIdx(eb)
                if h.a == link || h.b == link {
                    fmt.Println(link, h.a, h.b, ea, eb, "\n", self.Draw())
                    panic("h.a == link || h.b == link")
                }
                if h.a == h.b {
                    h.change = true
                    lsa, _ := self.linkSt[h.a].Top().(*LinkState)
                    self.addLinkState(h.a, 12, lsa.length+1, lsa.edge[0], lsa.edge[1],
                        self.maxEdgeInLoop(lsa.edge[0], lsa.edge[1]), -1)
                } else {
                    h.union = true
                    self.union(ea, eb, h.a, h.b)
                }
            }
        }
        return
    }

    self.linkFunc[9] = func(link, pos int) (nh int) {
        if pos != 0 {
            panic("linkFunc 9: pos != 0")
        }
        lst, _ := self.linkSt[link].Top().(*LinkState)
        h := self.addHistory()
        nh++
        h.change, h.a = true, link
        q := lst.edge[0]
        if q == lst.pe[pos] {
            q = lst.edge[1]
        }
        self.addLinkState(link, 1, 0, q, q, q, -1)
        np := self.edge[lst.pe[pos]].getMaxEnd()
        if np.degree == 2 {
            h = self.addHistory()
            nh++
            ea, eb := self.getTowEdge(np)
            h.a, h.b = self.getLinkIdx(ea), self.getLinkIdx(eb)
            if h.a == h.b {
                h.change = true
                lsa, _ := self.linkSt[h.a].Top().(*LinkState)
                self.addLinkState(h.a, 12, lsa.length+1, lsa.edge[0], lsa.edge[1],
                    self.maxEdgeInLoop(lsa.edge[0], lsa.edge[1]), -1)
            } else {
                h.union = true
                self.union(ea, eb, h.a, h.b)
            }
        }
        return
    }

    self.linkFunc[10] = func(link, pos int) (nh int) {
        lst, _ := self.linkSt[link].Top().(*LinkState)
        h := self.addHistory()
        nh++
        if pos == 0 {
            h.change, h.a = true, link
            q := lst.edge[0]
            if q == lst.pe[pos] {
                q = lst.edge[1]
            }
            self.addLinkState(link, 3, 1, lst.pe[1], q, lst.pe[1], q)
            np := self.edge[lst.pe[pos]].getMaxEnd()
            if np.degree == 2 {
                h = self.addHistory()
                nh++
                ea, eb := self.getTowEdge(np)
                h.a, h.b = self.getLinkIdx(ea), self.getLinkIdx(eb)
                if h.a == h.b {
                    h.change = true
                    lsa, _ := self.linkSt[h.a].Top().(*LinkState)
                    self.addLinkState(h.a, 12, lsa.length+1, lsa.edge[0], lsa.edge[1],
                        self.maxEdgeInLoop(lsa.edge[0], lsa.edge[1]), -1)
                } else {
                    h.union = true
                    self.union(ea, eb, h.a, h.b)
                }
            }
        } else {
            h.change, h.a = true, link
            self.addLinkState(link, -1, -1, -1, -1, -1, -1)
            for i := 0; i < 2; i++ {
                h = self.addHistory()
                nh++
                h.split = true
                h.a, h.b = lst.edge[i], self.parent[lst.edge[i]]
                self.newLinkState(h.a, 1, 0, lst.edge[i], lst.edge[i], lst.edge[i], -1)
            }
        }
        return
    }

    self.linkFunc[11] = func(link, pos int) (nh int) {
        if pos != 0 {
            panic("linkFunc 11: pos != 0")
        }
        lst, _ := self.linkSt[link].Top().(*LinkState)
        h := self.addHistory()
        nh++
        h.change, h.a = true, link
        q := lst.edge[0]
        if q == lst.pe[pos] {
            q = lst.edge[1]
        }
        o := self.nextEdge(lst.pe[pos])
        self.addLinkState(link, 2, lst.length-1, o, q, o, -1)
        np := self.edge[lst.pe[pos]].getMaxEnd()
        if np.degree == 2 {
            h = self.addHistory()
            nh++
            ea, eb := self.getTowEdge(np)
            h.a, h.b = self.getLinkIdx(ea), self.getLinkIdx(eb)
            if h.a == h.b {
                h.change = true
                lsa, _ := self.linkSt[h.a].Top().(*LinkState)
                self.addLinkState(h.a, 12, lsa.length+1, lsa.edge[0], lsa.edge[1],
                    self.maxEdgeInLoop(lsa.edge[0], lsa.edge[1]), -1)
            } else {
                h.union = true
                self.union(ea, eb, h.a, h.b)
            }
        }
        return
    }

    self.linkFunc[12] = func(link, pos int) (nh int) {
        if pos != 0 {
            panic("linkFunc 12: pos != 0")
        }
        lst, _ := self.linkSt[link].Top().(*LinkState)
        h := self.addHistory()
        nh++
        h.change, h.a = true, link
        o := self.edge[lst.pe[pos]].node[0].getEdge()
        q := self.edge[lst.pe[pos]].node[1].getEdge()
        if o < q {
            o, q = q, o
        }
        if lst.length == 4 {
            self.addLinkState(link, 7, 2, o, q, o, self.neightborEdge(o))
        } else {
            self.addLinkState(link, 6, lst.length-2, o, q, o, -1)
        }
        return
    }
}

func GetBoard(turn int) (b *QBoard, lastmoves []int) {
    b = new(QBoard)
    b.Init()
    lastmoves = make([]int, 0, 60)
    ms := make([]int, 0, 60)

    for int(b.Turn) < turn && b.IsEnd() == 0 {
        ms = b.GetMove12(ms)
        if len(ms) < 6 {
            ms = append(ms, b.GetMove3(ms[len(ms):cap(ms)])...)
        }
        lastmoves = lastmoves[:0]
        lastmoves = append(lastmoves, ms[rand.Intn(len(ms))])
        b.Move(lastmoves[0])
        ms = b.Play(ms)
        lastmoves = append(lastmoves, ms...)
    }
    return
}

func init() {
    rand.Seed(time.Now().Unix())
}
