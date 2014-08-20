/*********************************************************************************
*     File Name           :     merger.go
*     Created By          :     YIMMON, yimmon.zhuang@gmail.com
*     Creation Date       :     [2014-05-04 20:02]
*     Last Modified       :     [2014-05-05 00:02]
*     Description         :
**********************************************************************************/

package main

import (
    "ann/rcds"
    "flag"
    "fmt"
    "io/ioutil"
    "math/rand"
    "os"
    "path"
    "runtime"
    "time"
)

var (
    pminturn    = flag.Int("minturn", 99, "Min Turn.")
    pturnlength = flag.Int("turnlength", 8, "Turn Length.")
    phelp       = flag.Bool("help", false, "Help")
    num         int
)

func Merger(basedir, filename, tardir string) bool {
    var (
        fout, fin rcds.File
        topath    = path.Join(tardir, filename)
    )

    for t := 1; t < num; t++ {
        frompath := path.Join(basedir, fmt.Sprintf("%02d", t), filename)
        if fin.Open(frompath) != nil {
            continue
        }
        if fout.Null() {
            os.MkdirAll(path.Dir(topath), os.ModePerm)
            if err := fout.Create(topath); err != nil {
                panic(err)
            } else {
                fmt.Printf("Merging to %s\n", topath)
                defer fout.Close()
            }
        }
        fmt.Printf("\t%s\n", frompath)
        for {
            if line, err := fin.ReadOneLine(); err != nil {
                break
            } else {
                fout.WriteOneLine(line)
            }
        }
        fin.Close()
    }
    return !fout.Null()
}

func Shuffle(filename string) {
    var (
        fp        = make([]*rcds.File, 0, 1024)
        fin, fout rcds.File
        data      = make([]string, 0, 10000000)
    )
    if err := fin.Open(filename); err != nil {
        panic(err)
    }
    fmt.Printf("Shuffling %s\n", filename)
    for eof := false; !eof; {
        data = data[:0]
        for len(data) < cap(data) {
            if line, err := fin.ReadOneLine(); err != nil {
                eof = true
                break
            } else {
                data = append(data, line)
            }
        }
        if len(data) > 0 {
            tmpf, _ := ioutil.TempFile(path.Dir(filename), "tmpf-")
            nam := tmpf.Name()
            tmpf.Close()
            fout.Create(nam)
            for _, i := range rand.Perm(len(data)) {
                fout.WriteOneLine(data[i])
            }
            fout.Close()
            fr := new(rcds.File)
            fr.Open(nam)
            fp = append(fp, fr)
            fmt.Printf("\t%s : %d\n", nam, len(data))
        }
    }
    fin.Close()

    count := 0
    go func() {
        time.Sleep(2 * time.Second)
        for len(fp) > 0 {
            fmt.Printf("%d ", count)
            time.Sleep(2 * time.Second)
        }
    }()
    fout.Create(filename)
    for len(fp) > 0 {
        x := rand.Intn(len(fp))
        file := fp[x]
        if line, err := file.ReadOneLine(); err != nil {
            nam := path.Join(file.GetDir(), file.GetBase())
            file.Free()
            os.Remove(nam)
            fp = append(fp[:x], fp[x+1:]...)
        } else {
            fout.WriteOneLine(line)
            count++
        }
    }
    fout.Close()
    fmt.Printf("\n%d records writen.\n", count)
}

func init() {
    rand.Seed(time.Now().Unix())
    runtime.GOMAXPROCS(runtime.NumCPU())
    flag.Parse()
    if *phelp {
        fmt.Println("Usage: merger --minturn Num --turnlength Num")
        os.Exit(0)
    }
}

func main() {
    if fis, err := ioutil.ReadDir(path.Join("./DataSets", fmt.Sprintf("%02d-%02d", *pminturn, *pturnlength))); err != nil {
        panic(err)
    } else {
        if len(fis) < 2 {
            panic("The number of dataset less than 2.")
        }
        fmt.Sscanf(fis[len(fis)-1].Name(), "%d", &num)
        num++
    }
    tardir := path.Join("./DataSets", fmt.Sprintf("%02d-%02d", *pminturn, *pturnlength), fmt.Sprintf("%02d", num))
    fmt.Printf("Target dataset: %s\n", tardir)

    for turn := 0; turn < 60; turn++ {
        if Merger(path.Join("./DataSets", fmt.Sprintf("%02d-%02d", *pminturn, *pturnlength)),
            path.Join("TestSets", fmt.Sprintf("%02d.test", turn)), tardir) {
            Shuffle(path.Join(tardir, "TestSets", fmt.Sprintf("%02d.test", turn)))
        }
    }
    if Merger(path.Join("./DataSets", fmt.Sprintf("%02d-%02d", *pminturn, *pturnlength)),
        path.Join("TrainSets", "BALANCE.train"), tardir) {
        Shuffle(path.Join(tardir, "TrainSets", "BALANCE.train"))
    }

    fmt.Println("Done.")
}
