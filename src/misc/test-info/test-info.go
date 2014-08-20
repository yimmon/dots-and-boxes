/*********************************************************************************
*     File Name           :     test-info.go
*     Created By          :     YIMMON, yimmon.zhuang@gmail.com
*     Creation Date       :     [2014-05-21 14:04]
*     Last Modified       :     [2014-05-21 14:09]
*     Description         :
**********************************************************************************/

package main

import (
    "algorithm/board"
    "algorithm/qboard"
    "fmt"
)

func main() {
    for {
        var h, v int32
        var s0, s1, turn int
        fmt.Scanf("0x%x 0x%x %d %d %d", &h, &v, &s0, &s1, &turn)
        b := board.NewBoard(h, v, int8(s0), int8(s1), 0, int8(turn))
        qb := qboard.NewQBoard(h, v, s0, s1, 0, turn)
        fmt.Println("Board:")
        fmt.Println(b.Draw())
        fmt.Println(b.GetInfo())
        fmt.Println("QBoard:")
        fmt.Println(qb.Draw() + qb.ShowLinks())
        fmt.Println(qb.GetInfo())
    }
}
