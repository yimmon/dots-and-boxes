/*********************************************************************************
*     File Name           :     balance.go
*     Created By          :     YIMMON, yimmon.zhuang@gmail.com
*     Creation Date       :     [2014-05-07 19:28]
*     Last Modified       :     [2014-05-07 19:44]
*     Description         :
**********************************************************************************/

package main

import (
    "ann/rcds"
    "fmt"
    "math/rand"
    "path"
    "time"
)

func main() {
    rand.Seed(time.Now().Unix())
    dir := path.Join("./DataSets", "ExamSets")
    f := new(rcds.File)
    var r [2][]*rcds.Record
    r[0] = make([]*rcds.Record, 0, 10240)
    r[1] = make([]*rcds.Record, 0, 10240)

    for t := 0; t < 60; t++ {
        pth := path.Join(dir, fmt.Sprintf("%02d.exam", t))
        if f.Open(pth) != nil {
            continue
        }

        r[0], r[1] = r[0][:0], r[1][:0]
        for {
            if rcd, err := f.ReadOneRecord(); err != nil {
                break
            } else {
                if rcd.Z < 0 {
                    r[0] = append(r[0], rcd)
                } else {
                    r[1] = append(r[1], rcd)
                }
            }
        }
        f.Close()
        p := 0
        if len(r[1]) < len(r[0]) {
            p = 1
        }
        length := len(r[p])
        for i, j := range rand.Perm(len(r[p^1])) {
            if i >= length {
                break
            }
            r[p] = append(r[p], r[p^1][j])
        }
        f.Create(pth)
        for _, i := range rand.Perm(len(r[p])) {
            f.WriteOneRecord(r[p][i])
        }
        f.Close()
    }
}
