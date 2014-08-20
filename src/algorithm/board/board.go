/*********************************************************************************
*     File Name           :     board.go
*     Created By          :     YIMMON, yimmon.zhuang@gmail.com
*     Creation Date       :     [2014-04-08 22:44]
*     Last Modified       :     [2014-05-26 11:56]
*     Description         :
**********************************************************************************/

package board

import (
    "errors"
    "fmt"
    "log"
    "math/rand"
    "runtime/debug"
    "time"
)

type IAlgorithm interface {
    GetName() string
    MakeMove(b *Board, timeout uint, verbose bool) (h, v int32, err error)
}

type Board struct {
    H, V       int32
    S          [2]int8
    Now, Turn  int8
    node       [7][7]Node
    edge       [2][5][6]Edge
    head, tail [5]Node
    vst        [7][7]bool
    records    []*Edge
}

type Node struct {
    degree, x, y int8
    neightbor    [4]*Node
    edge         [4]*Edge
    edgeEntryIdx int8
    pre, next    *Node
}

type Edge struct {
    exist     bool
    code      int32
    x, y, z   int8
    idx       [2]int8
    node      [2]*Node
    pre, next [2]*Edge
}

type Moves struct {
    H, V, M int32
    ms      []*Edge
}

func (self *Board) Init() (err error) {
    self.H, self.V = 0, 0
    self.S[0], self.S[1] = 0, 0
    self.Now, self.Turn = 0, 0

    for i := int8(0); i < 7; i++ {
        for j := int8(0); j < 7; j++ {
            self.node[i][j].degree = 0
            self.node[i][j].x = i
            self.node[i][j].y = j
            self.node[i][j].neightbor = [4]*Node{nil, nil, nil, nil}
            self.node[i][j].edge = [4]*Edge{nil, nil, nil, nil}
            if i == 0 {
                self.node[i][j].edgeEntryIdx = 2
            } else if j == 0 {
                self.node[i][j].edgeEntryIdx = 1
            } else if j == 6 {
                self.node[i][j].edgeEntryIdx = 3
            } else {
                self.node[i][j].edgeEntryIdx = 0
            }
        }
    }
    for i := 0; i < 2; i++ {
        for j := 0; j < 5; j++ {
            for k := 0; k < 6; k++ {
                e := &self.edge[i][j][k]
                e.exist = true
                e.code = self.edgeXYZ2code(i, j, k)
                e.x, e.y, e.z = int8(i), int8(j), int8(k)
                if i == 0 {
                    e.idx = [2]int8{1, 3}
                    e.node[0] = &(self.node[j+1][k])
                    e.node[1] = &(self.node[j+1][k+1])
                } else {
                    e.idx = [2]int8{0, 2}
                    e.node[0] = &(self.node[k+1][j+1])
                    e.node[1] = &(self.node[k][j+1])
                }
                e.node[0].degree = 4
                e.node[1].degree = 4
                e.node[0].neightbor[e.idx[0]] = e.node[1]
                e.node[1].neightbor[e.idx[1]] = e.node[0]
                e.node[0].edge[e.idx[0]] = e
                e.node[1].edge[e.idx[1]] = e
            }
        }
    }
    for i := 0; i < 2; i++ {
        for j := 0; j < 5; j++ {
            for k := 0; k < 6; k++ {
                e := &self.edge[i][j][k]
                for l := 0; l < 2; l++ {
                    e.pre[l] = e.node[l].edge[(e.idx[l]+3)%4]
                    e.next[l] = e.node[l].edge[(e.idx[l]+1)%4]
                    if e.pre[l] == nil || e.next[l] == nil {
                        e.pre[l], e.next[l] = e, e
                    }
                }
            }
        }
    }

    self.records = make([]*Edge, 0, 60)
    for i := 0; i < 4; i++ {
        self.head[i].pre = &self.tail[i]
        self.head[i].next = &self.tail[i]
        self.tail[i].pre = &self.head[i]
        self.tail[i].next = &self.head[i]
    }
    np := &self.head[4]
    for i := 0; i < 7; i++ {
        for j := 0; j < 7; j++ {
            if (i == 0 && (j == 0 || j == 6)) || (i == 6 && (j == 0 || j == 6)) {
                continue
            }
            np.next = &self.node[i][j]
            self.node[i][j].pre = np
            np = np.next
        }
    }
    np.next = &self.tail[4]
    self.tail[4].pre = np
    self.tail[4].next = &self.head[4]
    self.head[4].pre = &self.tail[4]

    return
}

func (self *Board) Play() (moves *Moves, err error) {
    if self.head[1].next == &self.tail[1] {
        return
    }
    var (
        sumEdges     []*Edge
        sep          int = 0
        repeat, vst2     = true, false
        add              = func(buf []*Edge) {
            if sumEdges == nil {
                sumEdges = make([]*Edge, 0, 8)
            }
            sumEdges = append(sumEdges, buf...)
        }
    )

    for repeat {
        repeat, vst2 = false, false
        for node := self.head[1].next; node != &self.tail[1]; node = node.next {
            buf, end := self.walkToEnd(node)
            if len(end) == 1 {
                if len(buf[0]) > 1 {
                    vst2 = true
                }
                if end[0].degree == 1 {
                    if len(buf[0]) == 2 {
                        add(buf[0])
                    } else if len(buf[0]) >= 4 {
                        if self.nodeComp(node, end[0]) == 1 {
                            add(buf[0][:len(buf[0])-3])
                        } else {
                            add(buf[0][3:])
                        }
                    } else if len(buf[0]) == 1 && self.nodeComp(node, end[0]) == 1 {
                        add(buf[0])
                    }
                } else { // end[0].degree >= 3
                    if len(buf[0]) == 1 {
                        add(buf[0])
                        repeat = true
                    } else if len(buf[0]) >= 3 {
                        add(buf[0][:len(buf[0])-2])
                    }
                }
            }
        }
        if vst2 {
            for node := self.head[2].next; node != &self.tail[2]; node = node.next {
                self.vst[node.x][node.y] = false
            }
        }
        if sep == len(sumEdges) {
            break
        }
        if err = self.Move(self.newMoves(sumEdges[sep:])); err != nil {
            log.Panic("Play: ", err)
        }
        sep = len(sumEdges)
    }
    if len(sumEdges) != 0 {
        moves = self.newMoves(sumEdges)
    }
    return
}

func (self *Board) PlayRandomOne() (moves *Moves, err error) {
    if self.S[0]+self.S[1] == 25 {
        return nil, errors.New("PlayRandomOne: Game is over.")
    }

    var (
        ms          [30]uint8
        nm, x, y, z int = 0, 0, 0, 0
    )

    if (self.H != 0x3fffffff && rand.Intn(2) == 1) || self.V == 0x3fffffff {
        for n := uint8(0); n < 30; n++ {
            if (1<<n)&self.H == 0 {
                ms[nm] = n
                nm++
            }
        }
        x = 0
        y, z = self.edgeCodeN2YZ(uint(ms[rand.Intn(nm)]))
    } else {
        for n := uint8(0); n < 30; n++ {
            if (1<<n)&self.V == 0 {
                ms[nm] = n
                nm++
            }
        }
        x = 1
        y, z = self.edgeCodeN2YZ(uint(ms[rand.Intn(nm)]))
    }
    moves = self.newMoves([]*Edge{&self.edge[x][y][z]})
    err = self.Move(moves)
    return
}

/* 调用前保证已经调用过Play */
func (self *Board) GetMove() (moves []*Moves, noC int, err error) {
    /*
       if pm, err := self.Play(); pm != nil || err != nil {
           debug.PrintStack()
           log.Fatal("Haven't Play before GetMove.\n" + self.Draw())
       }
    */
    moves = make([]*Moves, 0, 60)
    movesT := make([]*Moves, 0, 60)

    for degree := 4; degree >= 3; degree-- {
        for node := self.head[degree].next; node != &self.tail[degree]; node = node.next {
            if node.degree == 4 || (node.degree == 3 && node.x >= 1 && node.x <= 5 && node.y >= 1 && node.y <= 5) {
                buf, end := self.walkToEnd(node)
                for k := 0; k < len(end); k++ {
                    if end[k].degree == 1 { // 口-O-O
                        moves = append(moves, self.newMoves(buf[k][1:2]))
                        movesT = append(movesT, self.newMoves(buf[k][:1]))
                    } else {
                        if self.nodeComp(node, end[k]) == 1 {
                            if len(buf[k]) <= 2 {
                                moves = append(moves, self.newMoves(buf[k][0:1])) // 口-...-口
                                if len(buf[k]) == 1 {
                                    noC++
                                }
                            } else {
                                movesT = append(movesT, self.newMoves(buf[k][0:1])) // 口-...-口
                            }
                        } else if len(buf[k]) > 1 { // 口-...-O-...-口
                            if len(buf[k]) == 2 {
                                moves = append(moves, self.newMoves(buf[k][len(buf[k])-1:]))
                            } else {
                                movesT = append(movesT, self.newMoves(buf[k][len(buf[k])-1:]))
                            }
                        }
                        if len(buf[k]) == 3 { // 口-O-O-口
                            movesT = append(movesT, self.newMoves(buf[k][1:2]))
                        }
                    }
                }
            }
        }
    }
    for node := self.head[1].next; node != &self.tail[1]; node = node.next {
        buf, end := self.walkToEnd(node)
        if len(buf) == 1 {
            if end[0].degree >= 3 { // 口-O-O
                moves = append(moves, self.newMoves(buf[0][:1]))
                movesT = append(movesT, self.newMoves(buf[0][1:2]))
            } else { // O-O-O-O
                if self.nodeComp(node, end[0]) == 1 {
                    moves = append(moves, self.newMoves(buf[0][:1]))
                    movesT = append(movesT, self.newMoves(buf[0][1:2]))
                } else {
                    moves = append(moves, self.newMoves(buf[0][len(buf[0])-1:]))
                    movesT = append(movesT, self.newMoves(buf[0][len(buf[0])-2:len(buf[0])-1]))
                }
            }
        }
    }
    for node := self.head[2].next; node != &self.tail[2]; node = node.next {
        if !self.vst[node.x][node.y] {
            e, _ := self.walkLoop(node)
            movesT = append(movesT, self.newMoves([]*Edge{e}))
        }
    }
    for node := self.head[2].next; node != &self.tail[2]; node = node.next {
        self.vst[node.x][node.y] = false
    }
    moves = append(moves, movesT...)

    if len(moves) == 0 {
        err = errors.New("No moves.")
        //debug.PrintStack()
        //log.Fatal("GetMove: No moves.\n" + self.Draw())
    }
    return
}

func (self *Board) Move(allmoves ...*Moves) (err error) {
    for _, moves := range allmoves {
        if moves == nil || len(moves.ms) == 0 {
            continue
        }
        h, v := moves.Moves2HV()
        if self.H&h != 0 || self.V&v != 0 {
            debug.PrintStack()
            log.Fatal("Move: Repeative move.\n" + self.Draw() +
                fmt.Sprintf("move h: 0x%08x, v: 0x%08x\n", h, v))
        }
        self.H |= h
        self.V |= v
        self.records = append(self.records, moves.ms...)
        count := int8(0)
        for _, e := range moves.ms {
            e.exist = false
            for i := 0; i < 2; i++ {
                np := e.node[i]
                if np == e.pre[i].node[0] {
                    e.pre[i].next[0] = e.next[i]
                } else {
                    e.pre[i].next[1] = e.next[i]
                }
                if np == e.next[i].node[0] {
                    e.next[i].pre[0] = e.pre[i]
                } else {
                    e.next[i].pre[1] = e.pre[i]
                }

                if np.degree--; np.degree == 0 {
                    count++
                }
                if e == np.edge[np.edgeEntryIdx] {
                    if e.next[i].node[0] == np {
                        np.edgeEntryIdx = e.next[i].idx[0]
                    } else {
                        np.edgeEntryIdx = e.next[i].idx[1]
                    }
                }
                np.pre.next = np.next
                np.next.pre = np.pre
                np.pre = &self.head[np.degree]
                np.next = self.head[np.degree].next
                np.pre.next = np
                np.next.pre = np
            }
        }

        self.S[self.Now] += count
        if moves.ms[len(moves.ms)-1].node[0].degree != 0 &&
            moves.ms[len(moves.ms)-1].node[1].degree != 0 {
            self.Now ^= 1
            self.Turn++
        }
    }
    return
}

func (self *Board) UnMove(allmoves ...*Moves) (err error) {
    rcdslen := len(self.records)
    for _, moves := range allmoves {
        if moves == nil || len(moves.ms) == 0 {
            continue
        }
        h, v := moves.Moves2HV()
        if self.H&h != h || self.V&v != v {
            debug.PrintStack()
            fmt.Println(self.Draw())
            log.Fatal("UnMove: Invalid unmove.\n" + self.Draw() +
                fmt.Sprintf("move h: 0x%08x, v: 0x%08x\n", h, v))
        }
        self.H &= ^h
        self.V &= ^v
        if moves.ms[len(moves.ms)-1].node[0].degree != 0 &&
            moves.ms[len(moves.ms)-1].node[1].degree != 0 {
            self.Now ^= 1
            self.Turn--
        }
        count := int8(0)
        for t := len(moves.ms) - 1; t >= 0; t-- {
            e := moves.ms[t]
            if rcdslen--; e != self.records[rcdslen] {
                log.Panic("UnMove error. ", e, self.records[rcdslen])
            }
            e.exist = true
            for i := 0; i < 2; i++ {
                np := e.node[i]
                if np == e.pre[i].node[0] {
                    e.pre[i].next[0] = e
                } else {
                    e.pre[i].next[1] = e
                }
                if np == e.next[i].node[0] {
                    e.next[i].pre[0] = e
                } else {
                    e.next[i].pre[1] = e
                }

                if np.degree++; np.degree == 1 {
                    count++
                }
                np.pre.next = np.next
                np.next.pre = np.pre
                np.pre = &self.head[np.degree]
                np.next = self.head[np.degree].next
                np.pre.next = np
                np.next.pre = np
            }
        }
        self.S[self.Now] -= count
    }
    self.records = self.records[:rcdslen]

    return
}

func (self *Board) GetCMoves() (moves *Moves, err error) {
    es := make([]*Edge, 0, 12)
    for node := self.head[1].next; node != &self.tail[1]; node = node.next {
        es = append(es, node.edge[node.edgeEntryIdx])
    }
    moves = self.newMoves(es)
    return
}

func (self *Board) HasNoCAfter(m *Moves) bool {
    if m.ms[0].node[0].degree >= 3 && m.ms[0].node[1].degree >= 3 {
        return true
    }
    return false
}

func (self *Board) LinksTwo4(m *Moves) bool {
    if m.ms[0].node[0].degree >= 4 && m.ms[0].node[1].degree >= 4 {
        return true
    }
    return false
}

func (self *Board) CanGetPointAfter(m *Moves) bool {
    if m.ms[0].node[0].degree == 1 || m.ms[0].node[1].degree == 1 {
        return true
    }
    return false
}

func (self *Board) LoseOneAfter(m *Moves) bool {
    if (m.ms[0].node[0].degree == 2 && m.ms[0].node[1].degree >= 3) ||
        (m.ms[0].node[0].degree >= 3 && m.ms[0].node[1].degree == 2) {
        p := 0
        if m.ms[0].node[p].degree != 2 {
            p = 1
        }
        e := m.ms[0].next[p]
        if (e.node[0].degree == 2 && e.node[1].degree >= 3) ||
            (e.node[0].degree >= 3 && e.node[1].degree == 2) {
            return true
        }
    }
    return false
}

func (self *Board) IsEnd() int {
    switch {
    case self.S[0] >= 13:
        return -1
    case self.S[1] >= 13:
        return 1
    default:
        return 0
    }
}

func (self *Board) edgeXYZ2code(x, y, z int) (ret int32) {
    ret = 1 << uint(y*6+z)
    if x == 1 {
        ret |= -0x80000000
    }
    return
}

func (self *Board) edgeCode2XYZ(code int32) (x, y, z int) {
    if code < 0 {
        x = 1
        code &= 0x7fffffff
    } else {
        x = 0
    }
    var i int
    for i = 0; code != 1; i++ {
        code >>= 1
    }
    y, z = i/6, i%6
    return
}

func (self *Board) edgeCodeN2YZ(n uint) (y, z int) {
    y, z = int(n)/6, int(n)%6
    return
}

func (self *Board) nodeComp(a, b *Node) int {
    if a.x > b.x {
        return 1
    } else if a.x < b.x {
        return -1
    } else {
        if a.y > b.y {
            return 1
        } else if a.y < b.y {
            return -1
        }
    }
    return 0
}

/* 入口s的度不能为2 */
func (self *Board) walkToEnd(s *Node) (buf [][]*Edge, end []*Node) {
    se, to, cur := s.edge[s.edgeEntryIdx], 0, -1
    if s.edgeEntryIdx < 2 {
        to = 1
    }
    for {
        if !self.vst[se.node[to].x][se.node[to].y] {
            if buf == nil {
                buf = make([][]*Edge, 0, 4)
                end = make([]*Node, 0, 4)
            }
            ne, nt, np := se, to, se.node[to]
            buf, cur = append(buf, make([]*Edge, 0, 12)), cur+1
            buf[cur] = append(buf[cur], ne)
            for np.degree == 2 {
                self.vst[np.x][np.y] = true
                if ne = ne.next[nt]; ne.node[nt] == np {
                    nt ^= 1
                }
                np = ne.node[nt]
                buf[cur] = append(buf[cur], ne)
            }
            end = append(end, np)
        }
        if se = se.next[to^1]; se == s.edge[s.edgeEntryIdx] {
            break
        }
        if se.node[to] == s {
            to ^= 1
        }
    }
    return
}

/* 入口s必是环中一点 */
func (self *Board) walkLoop(s *Node) (e *Edge, length int) {
    ne, nt, np := s.edge[s.edgeEntryIdx], 0, s
    if ne.node[nt] != s {
        nt ^= 1
    }
    length = 0

    for e = ne; !self.vst[np.x][np.y]; np = ne.node[nt] {
        self.vst[np.x][np.y] = true
        length++
        if ne = ne.next[nt]; ne.node[nt] == np {
            nt ^= 1
        }
        if ne.code > e.code {
            e = ne
        }
    }
    return
}

func NewBoard(h, v int32, s0, s1, now, turn int8) (b *Board) {
    b = new(Board)
    b.Init()
    b.Move(b.NewMoves(h, v, 0))
    b.H, b.V = h, v
    b.S[0], b.S[1], b.Now, b.Turn = s0, s1, now, turn
    return
}

func (self *Board) newMoves(es []*Edge) (moves *Moves) {
    if len(es) == 0 {
        return nil
    }
    if moves = new(Moves); moves == nil {
        debug.PrintStack()
        log.Fatal("newMoves fail.")
    }
    moves.ms = es
    if len(es) > 1 {
        for _, e := range es[:len(es)-1] {
            if e.code > 0 {
                moves.H |= e.code
            } else {
                moves.V |= (e.code & 0x7fffffff)
            }
        }
    }
    moves.M = es[len(es)-1].code
    return
}

func (self *Board) NewMoves(h, v, m int32) (moves *Moves) {
    if h == 0 && v == 0 && m == 0 {
        return nil
    }
    if moves = new(Moves); moves == nil {
        debug.PrintStack()
        log.Fatal("NewMoves fail.")
    }
    moves.H, moves.V, moves.M = h, v, m
    moves.ms = make([]*Edge, 0, 16)
    for i, o := range [2]int32{h, v} {
        for j := uint(0); (1 << j) <= o; j++ {
            if (1<<j)&o != 0 {
                x, y := self.edgeCodeN2YZ(j)
                moves.ms = append(moves.ms, &self.edge[i][x][y])
            }
        }
    }
    if m != 0 {
        i, j, k := self.edgeCode2XYZ(m)
        moves.ms = append(moves.ms, &self.edge[i][j][k])
    }
    return
}

func (self *Board) MoveHV(h, v int32) (moves []*Moves, err error) {
    ms := make([]*Moves, 0, 16)
    for i, o := range [2]int32{h, v} {
        for j := uint(0); (1 << j) <= o; j++ {
            if (1<<j)&o != 0 {
                ms = append(ms, self.newMoves([]*Edge{&self.edge[i][j/6][j%6]}))
            }
        }
    }
    moves = make([]*Moves, 0, len(ms))
    for {
        for i, m := range ms {
            if m != nil && self.CanGetPointAfter(m) {
                self.Move(m)
                moves = append(moves, m)
                ms[i] = nil
            }
        }
        tmpNo, tmpYes := 0, 0
        for _, m := range ms {
            if m != nil {
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
                if m != nil {
                    self.Move(m)
                    moves = append(moves, m)
                    return
                }
            }
        } else if tmpYes == 0 {
            log.Println(self.Draw())
            moves, err = nil, errors.New("Illegal h, v.")
            return
        }
    }
}

func (self *Board) Draw() string {
    var layout [11][12]byte

    for i := 0; i < 11; i++ {
        for j := 0; j < 11; j++ {
            layout[i][j] = byte(' ')
        }
        layout[i][11] = byte('\n')
    }

    for i := 0; i < 5; i++ {
        for j := 0; j < 6; j++ {
            if !self.edge[0][i][j].exist {
                layout[i*2+1][j*2] = byte('|')
            }
        }
    }
    for i := 0; i < 5; i++ {
        for j := 0; j < 6; j++ {
            layout[j*2][i*2] = byte('.')
            layout[j*2][i*2+2] = byte('.')
            if !self.edge[1][i][j].exist {
                layout[j*2][i*2+1] = byte('_')
            }
        }
    }

    str := ""
    for i := 0; i < 11; i++ {
        str += " " + string(layout[i][:])
    }
    str = fmt.Sprintf("%sTurn=%d, S[0]=%d, S[1]=%d, Now=%d\n",
        str, self.Turn, self.S[0], self.S[1], self.Now)
    return str
}

func (self *Moves) Moves2HV() (h, v int32) {
    h, v = self.H, self.V
    if self.M > 0 {
        h |= self.M
    } else {
        v |= (self.M & 0x7fffffff)
    }
    return
}

type Info struct {
    Link                [11]int
    Loop                [5]int
    Halfheart, Fourlink int
    Point               [5]int
}

func (self *Board) GetInfo() (info *Info) {
    /*
       if pm, err := self.Play(); pm != nil || err != nil {
           debug.PrintStack()
           log.Fatal("Haven't Play before GetInfo.\n" + self.Draw())
       }
    */

    info = new(Info)
    for i := 1; i <= 5; i++ {
        for j := 1; j <= 5; j++ {
            info.Point[self.node[i][j].degree]++
        }
    }
    info.Point[0] = 0
    for _, i := range [2]int{0, 6} {
        for j := 1; j <= 5; j++ {
            if self.node[i][j].degree == 4 {
                info.Point[0]++
            }
            if self.node[j][i].degree == 4 {
                info.Point[0]++
            }
        }
    }

    for degree := 4; degree >= 3; degree-- {
        for node := self.head[degree].next; node != &self.tail[degree]; node = node.next {
            if node.degree == 4 || (node.degree == 3 && node.x >= 1 && node.x <= 5 && node.y >= 1 && node.y <= 5) {
                buf, end := self.walkToEnd(node)
                for k := 0; k < len(end); k++ {
                    if end[k].degree == 1 {
                        info.Halfheart++
                    } else {
                        ll := len(buf[k]) - 1
                        if ll > 10 {
                            ll = 10
                        }
                        if self.nodeComp(node, end[k]) == 1 {
                            info.Link[ll]++
                        } else if len(buf[k]) > 1 {
                            info.Link[ll]++
                        }
                    }
                }
            }
        }
    }
    for node := self.head[1].next; node != &self.tail[1]; node = node.next {
        buf, end := self.walkToEnd(node)
        if len(buf) == 1 {
            if end[0].degree >= 3 {
                info.Halfheart++
            } else {
                info.Fourlink++
            }
        }
    }
    for node := self.head[2].next; node != &self.tail[2]; node = node.next {
        if !self.vst[node.x][node.y] {
            _, length := self.walkLoop(node)
            if length = (length - 4) >> 1; length > 4 {
                length = 4
            }
            info.Loop[length]++
        }
    }
    for node := self.head[2].next; node != &self.tail[2]; node = node.next {
        self.vst[node.x][node.y] = false
    }

    return
}

func GetBoard(turn int) (b *Board, lastmoves []*Moves) {
    b = NewBoard(0, 0, 0, 0, 0, 0)
    moves := make([]*Moves, 0, 60)
    lastmoves = make([]*Moves, 0, 60)

    for int(b.Turn) < turn && b.IsEnd() == 0 {
        moves = moves[:0]
        ms, noC, _ := b.GetMove()
        if noC >= 4 && len(ms)-noC > 0 { // **布局阶段未结束
            for _, m := range ms {
                if b.HasNoCAfter(m) || b.CanGetPointAfter(m) || b.LoseOneAfter(m) {
                    moves = append(moves, m)
                }
            }
        }
        lastmoves = lastmoves[:0]
        if len(moves) >= 4 {
            lastmoves = append(lastmoves, moves[rand.Intn(len(moves))])
        } else {
            lastmoves = append(lastmoves, ms[rand.Intn(len(ms))])
        }
        b.Move(lastmoves[0])
        pm, _ := b.Play()
        lastmoves = append(lastmoves, pm)
    }

    return
}

func init() {
    rand.Seed(time.Now().Unix())
}
