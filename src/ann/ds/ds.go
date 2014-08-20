/*********************************************************************************
*     File Name           :     ds.go
*     Created By          :     YIMMON, yimmon.zhuang@gmail.com
*     Creation Date       :     [2014-04-29 19:52]
*     Last Modified       :     [2014-05-14 13:54]
*     Description         :
**********************************************************************************/

package main

import (
    "algorithm/board"
    "ann/eval"
    "ann/rcds"
    "flag"
    "fmt"
    "io/ioutil"
    "log"
    "math/rand"
    "os"
    "path"
    "runtime"
    "runtime/debug"
    "sort"
    "sync/atomic"
    "time"
)

type Ds struct {
    TrainSetSize               [60]int64
    HashManager                rcds.Hash
    TestFile, TrainFile        [60]rcds.File
    NTest, MinTurn, TurnLength int
    Dir                        string
}

const (
    maxTrainSetSize int64 = 35000000
)

var (
    numThread int = 1
    beginTime time.Time
)

func (self *Ds) Init(minturn, turnlength, ntest int) {
    self.NTest, self.MinTurn, self.TurnLength = ntest, minturn, turnlength
    self.Dir = path.Join("./DataSets", fmt.Sprintf("%02d-%02d", minturn, turnlength))
    self.Dir = path.Join(self.Dir, fmt.Sprintf("%02d", self.GetNum()))
    for i := 0; i < 60; i++ {
        self.TrainSetSize[i] = 0
    }
    self.HashManager.Init(func(k *rcds.HashKey, v *rcds.HashValue) {
        rcd := &rcds.Record{H: k.H, V: k.V, S0: k.S0, S1: k.S1, Z: v.Z}
        self.WriteTrain(int(v.Turn), rcd)
    })
}

func (self *Ds) GetNum() (num int) {
    os.MkdirAll(self.Dir, os.ModePerm)
    if fis, err := ioutil.ReadDir(self.Dir); err != nil {
        panic(err)
    } else {
        num = 1
        if len(fis) > 0 {
            fmt.Sscanf(fis[len(fis)-1].Name(), "%d", &num)
            num++
        }
    }
    return
}

func (self *Ds) WriteTest(turn int, rcd *rcds.Record) {
    if self.TestFile[turn].Null() {
        if err := self.TestFile[turn].Create(path.Join(self.Dir, "TestSets",
            fmt.Sprintf("%02d.test", turn))); err != nil {
            panic(err)
        }
    }
    if err := self.TestFile[turn].WriteOneRecord(rcd); err != nil {
        fmt.Println(err)
    }
}

func (self *Ds) WriteTrain(turn int, rcd *rcds.Record) bool {
    if turn >= self.MinTurn+self.TurnLength ||
        atomic.LoadInt64(&self.TrainSetSize[turn]) >= maxTrainSetSize {
        return false
    }
    if self.TrainFile[turn].Null() {
        if err := self.TrainFile[turn].Create(path.Join(self.Dir, "TrainSets",
            fmt.Sprintf("%02d.train", turn))); err != nil {
            panic(err)
        }
    }
    if err := self.TrainFile[turn].WriteOneRecord(rcd); err != nil {
        fmt.Println(err)
        return false
    } else {
        atomic.AddInt64(&self.TrainSetSize[turn], 1)
        return true
    }
}

func (self *Ds) CloseHash() {
    defer runtime.GC()
    for i := 0; i < len(self.HashManager.HashTable); i++ {
        count := int64(0)
        self.HashManager.RWMutex[i].Lock()
        for k, v := range self.HashManager.HashTable[i] {
            rcd := &rcds.Record{H: k.H, V: k.V, S0: k.S0, S1: k.S1, Z: v.Z}
            if self.WriteTrain(int(v.Turn), rcd) {
                count++
            }
        }
        self.HashManager.HashTable[i] = nil
        self.HashManager.RWMutex[i].Unlock()
        fmt.Printf("Flushed idx[%d] %d train records.\n", i, count)
    }
}

func (self *Ds) GenerateTest(cnt int32, anns *eval.Anns) bool {
    defer func() {
        if r := recover(); r != nil {
            fmt.Println("panic:", r)
            debug.PrintStack()
        }
    }()
    var (
        z        int
        b        *board.Board
        lms      []*board.Moves
        timeout  = make(chan int, 1)
        finish   = make(chan int, 1)
        stop     = make(chan int, 1)
        leafturn = self.MinTurn + self.TurnLength + 1
    )
    turn := RandomTurn(self.MinTurn, self.TurnLength)
    b, lms = board.GetBoard(turn)
    if b.IsEnd() != 0 {
        for i := len(lms) - 1; i >= 0; i-- {
            b.UnMove(lms[i])
        }
        if b.IsEnd() != 0 {
            panic("b is end.")
        }
    }
    turn = int(b.Turn)

    go func() {
        time.Sleep(5 * 60 * time.Second)
        timeout <- 1
    }()
    go func() {
        defer func() {
            if r := recover(); r != nil {
                fmt.Println("panic:", r)
                debug.PrintStack()
                z = 0
                finish <- 1
            }
        }()
        if z = -1; eval.WhoWillWin(b, &self.HashManager, leafturn, anns, stop) == b.Now {
            z = 1
        }
        finish <- 1
    }()

    fmt.Printf("Generating the %dth test record [turn: %d].\n", cnt, turn)
    select {
    case <-timeout:
        stop <- 1
        <-finish
        return false
    case <-finish:
        if z != 0 {
            rcd := &rcds.Record{H: b.H, V: b.V, S0: b.S[b.Now], S1: b.S[b.Now^1], Z: int8(z)}
            self.WriteTest(turn, rcd)
            return true
        }
    }
    return false
}

func (self *Ds) Generate() {
    defer runtime.GC()
    fmt.Printf("DataSets dir: %s\n", self.Dir)
    if err := os.MkdirAll(path.Join(self.Dir, "TestSets"), os.ModePerm); err != nil {
        panic(err)
    }
    fmt.Printf("Make dir: %s\n", path.Join(self.Dir, "TestSets"))
    if err := os.MkdirAll(path.Join(self.Dir, "TrainSets"), os.ModePerm); err != nil {
        panic(err)
    }
    fmt.Printf("Make dir: %s\n", path.Join(self.Dir, "TrainSets"))

    var (
        exit              = make(chan int, numThread)
        count, succ int32 = 0, 0
    )
    for i := 0; i < numThread; i++ {
        go func() {
            var cnt int32
            anns := eval.GetAnnModels("./AnnModels")
            for {
                if cnt = atomic.AddInt32(&count, 1); cnt > int32(self.NTest) {
                    break
                }
                if self.GenerateTest(cnt, anns) {
                    atomic.AddInt32(&succ, 1)
                }
                if cnt >= 200 && cnt%200 < int32(numThread) {
                    fmt.Printf("count %d, sleep 1 minute.\n", cnt)
                    time.Sleep(1 * time.Minute)
                    runtime.GC()
                }
            }
            anns.DestroyAnnModels()
            exit <- 1
        }()
    }

    for i := 0; i < numThread; i++ {
        <-exit
    }
    fmt.Printf("Generated (%d/%d) test records.\n", succ, int(count)-numThread)
    fmt.Println("Flushing train records.")
    self.CloseHash()
    for i := 0; i < 60; i++ {
        self.TestFile[i].Free()
        self.TrainFile[i].Free()
    }

    fmt.Println("Generate() Done.\n")
}

func (self *Ds) Sort() {
    defer runtime.GC()
    var (
        r, no    int
        filepath [2]string
        file     [3]rcds.File
        offset   [2][]int64
        records  = make([]string, 0, 10000000)
    )
    dir := path.Join(self.Dir, "TrainSets")
    sorteddir := path.Join(dir, "Sorted")
    if err := os.MkdirAll(sorteddir, os.ModePerm); err != nil {
        fmt.Println(err)
    }
    fmt.Printf("Make dir: %s\n", sorteddir)

    for i := 0; i < 60; i++ {
        filepath[0] = path.Join(dir, fmt.Sprintf("%02d.train", i))
        if err := file[0].Open(filepath[0]); err != nil {
            if os.IsExist(err) {
                fmt.Println("Sort", filepath[0], err)
            }
            continue
        }
        no = 0
        fmt.Printf("\nSorting %02d.train\n", i)
        tmpf, _ := ioutil.TempFile(dir, "tmpf-")
        filepath[1] = tmpf.Name()
        tmpf.Close()
        file[1].Create(filepath[1])

        offset[1] = make([]int64, 0, 128)
        for eof := false; !eof; {
            records = records[:0]
            for count := 0; count < cap(records); count++ {
                if line, err := file[0].ReadOneLine(); err != nil {
                    eof = true
                    break
                } else {
                    records = append(records, line)
                }
            }
            sort.Sort(rcds.ByString(records))

            last := ""
            count := 0
            for _, line := range records {
                if line != last {
                    file[1].WriteOneLine(line)
                    last = line
                    count++
                }
            }
            if count > 0 {
                offset[1] = append(offset[1], int64(count))
                fmt.Printf("%d ", count)
            }
        }
        file[0].Close()
        file[1].Close()
        no++
        fmt.Printf("\nTurn: %d, No.%d, %d blocks.\n", i, no, len(offset[1]))

        for r = 1; len(offset[r]) > 1; r ^= 1 {
            file[0].Open(filepath[r])
            file[1].Open(filepath[r])
            os.Truncate(filepath[r^1], 0)
            file[2].Create(filepath[r^1])

            if (len(offset[r]) & 1) == 1 {
                offset[r] = append(offset[r], 0)
            }
            offset[r^1] = make([]int64, 0, 128)
            sumoffset := int64(0)
            for j := 0; j < len(offset[r]); j += 2 {
                file[0].Seek(sumoffset)
                file[1].Seek(sumoffset + offset[r][j])
                sumoffset += offset[r][j] + offset[r][j+1]

                var (
                    cnt  [2]int64
                    line [2]string
                    err  error
                )
                cnt[0], cnt[1] = offset[r][j], offset[r][j+1]
                for k := 0; k < 2; k++ {
                    if line[k], err = file[k].ReadOneLine(); err != nil {
                        line[k] = ""
                    } else {
                        cnt[k]--
                    }
                }

                last := ""
                count := int64(0)
                for line[0] != "" || line[1] != "" {
                    p := 1
                    if line[0] != "" && line[0] < line[1] {
                        p = 0
                    }
                    if line[p] != last {
                        file[2].WriteOneLine(line[p])
                        last = line[p]
                        count++
                    }
                    if cnt[p] > 0 {
                        line[p], _ = file[p].ReadOneLine()
                        cnt[p]--
                    } else {
                        line[p] = ""
                    }
                }
                if count > 0 {
                    offset[r^1] = append(offset[r^1], count)
                    fmt.Printf("%d ", count)
                }
            }

            file[0].Close()
            file[1].Close()
            file[2].Close()
            no++
            fmt.Printf("\nTurn: %d, No.%d, %d blocks.\n", i, no, len(offset[r^1]))
        }

        if err := os.Rename(filepath[r], path.Join(sorteddir, fmt.Sprintf("%02d.train", i))); err != nil {
            fmt.Println(err)
        }
        fmt.Printf("Rename %s to %s\n", filepath[r], path.Join(sorteddir, fmt.Sprintf("%02d.train", i)))
        if err := os.Remove(filepath[r^1]); err != nil {
            fmt.Println(err)
        }
        fmt.Printf("Remove %s\n", filepath[r^1])
    }

    fmt.Println("Sort() Done.")
}

func (self *Ds) Merge() {
    defer runtime.GC()
    pathall := path.Join(self.Dir, "TrainSets", "ALL.train")
    fileall := &rcds.File{}
    if err := fileall.Create(pathall); err != nil {
        panic(err)
    }
    defer fileall.Close()
    fmt.Printf("\nMerging to %s\n", pathall)

    var (
        fileslice = make([]*rcds.File, 0, 128)
        records   = make([]string, 0, 10000000)
    )
    file, tmpf := &rcds.File{}, &rcds.File{}
    for i := 0; i < 60; i++ {
        filepath := path.Join(self.Dir, "TrainSets", "Sorted", fmt.Sprintf("%02d.train", i))
        if err := file.Open(filepath); err != nil {
            continue
        }

        for eof := false; !eof; {
            records = records[:0]
            for count := 0; count < cap(records); count++ {
                if line, err := file.ReadOneLine(); err != nil {
                    eof = true
                    break
                } else {
                    records = append(records, line)
                }
            }
            tmp, _ := ioutil.TempFile(path.Dir(pathall), "tmp-")
            nam := tmp.Name()
            tmp.Close()
            tmpf.Create(nam)
            for _, j := range rand.Perm(len(records)) {
                tmpf.WriteOneLine(records[j])
            }
            tmpf.Close()
            tmpr := new(rcds.File)
            tmpr.Open(nam)
            fileslice = append(fileslice, tmpr)
            fmt.Printf("%d ", len(fileslice))
        }
        file.Close()
    }

    fmt.Printf("\nWriting to All.train\n")
    var count int64 = 0
    go func() {
        time.Sleep(2 * time.Second)
        for len(fileslice) > 0 {
            fmt.Printf("%d ", count)
            time.Sleep(2 * time.Second)
        }
    }()
    for len(fileslice) > 0 {
        x := rand.Intn(len(fileslice))
        file := fileslice[x]
        if line, err := file.ReadOneLine(); err != nil {
            nam := path.Join(file.GetDir(), file.GetBase())
            file.Free()
            os.Remove(nam)
            fileslice = append(fileslice[:x], fileslice[x+1:]...)
        } else {
            fileall.WriteOneLine(line)
            count++
        }
    }
    fmt.Printf("\n%d records writen.\n", count)

    fmt.Println("Merge() Done.")
}

func (self *Ds) Balance() {
    var (
        eof              bool = false
        count, quota     int  = 0, 0
        fileall, filebal rcds.File
        dirpath          = path.Join(self.Dir, "TrainSets")
        pathall          = path.Join(dirpath, "ALL.train")
        pathbal          = path.Join(dirpath, "BALANCE.train")
    )
    if err := fileall.Open(pathall); err != nil {
        panic(err)
    }
    defer fileall.Close()
    if err := filebal.Create(pathbal); err != nil {
        panic(err)
    }
    defer filebal.Close()

    fmt.Println("\nBalancing")
    go func() {
        time.Sleep(2 * time.Second)
        for !eof {
            fmt.Printf("%d ", count)
            time.Sleep(2 * time.Second)
        }
    }()
    for {
        if rcd, err := fileall.ReadOneRecord(); err != nil {
            eof = true
            break
        } else {
            if rcd.Z < 0 {
                filebal.WriteOneRecord(rcd)
                count++
                quota++
            } else if rcd.Z > 0 && quota > 0 && rand.Intn(4) == 0 {
                filebal.WriteOneRecord(rcd)
                count++
                quota--
            }
        }
    }
    fmt.Printf("\n%d records writen.\n", count)

    fmt.Println("Balance() Done.")
}

func (self *Ds) Run() {
    fmt.Printf("Base dir: %s\nMin turn: %d\nTurn length: %d\nNTest: %d\n",
        self.Dir, self.MinTurn, self.TurnLength, self.NTest)

    self.Generate()
    self.Sort()
    self.Merge()
    self.Balance()
}

func RandomTurn(minturn, length int) int {
    if minturn+length-1 > 36 {
        length = 37 - minturn
    }
    arr := make([]int, 0, 64)
    for i := 1; i <= length; i++ {
        for j := 0; j < i; j++ {
            arr = append(arr, minturn+length-i)
        }
    }
    return arr[rand.Intn(len(arr))]
}

func init() {
    beginTime = time.Now()
    rand.Seed(time.Now().Unix())
    numThread = runtime.NumCPU()
    runtime.GOMAXPROCS(numThread)
    nullfile, _ := os.Create("/dev/null")
    log.SetOutput(nullfile)
}

func main() {
    minturnp := flag.Int("minturn", 99, "Min turn start to search.")
    turnlengthp := flag.Int("turnlength", 8, "Turn Length.")
    ntestp := flag.Int("ntest", 0, "The number of test records.")
    phelp := flag.Bool("help", false, "Help")
    flag.Parse()
    if *phelp {
        fmt.Println("Usage: ds --minturn Num --turnlength Num --ntest Num")
        os.Exit(0)
    }
    if *ntestp <= 0 {
        fmt.Println("ntest must > 0.")
        os.Exit(0)
    }
    if *minturnp > 32 {
        fmt.Println("minturn must <= 32")
        os.Exit(0)
    }

    var ds Ds
    ds.Init(*minturnp, *turnlengthp, *ntestp)
    ds.Run()

    fmt.Println("Finish. ", time.Since(beginTime))
}
