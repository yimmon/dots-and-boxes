/*********************************************************************************
*     File Name           :     test-models.go
*     Created By          :     YIMMON, yimmon.zhuang@gmail.com
*     Creation Date       :     [2014-05-07 19:54]
*     Last Modified       :     [2014-05-27 13:39]
*     Description         :
**********************************************************************************/

package main

import (
    "ann"
    "ann/qeval"
    "ann/qrcds"
    "encoding/json"
    "flag"
    "fmt"
    "os"
    "path"
)

func main() {
    pturn := flag.Int("turn", -1, "Turn")
    flag.Parse()
    dir := path.Join("./DataSets", "ExamSets")
    //dir := path.Join("ExamSets")
    f := new(qrcds.File)
    annmodelsdir := "./AnnModels"
    anns := qeval.GetAnnModels(annmodelsdir)
    statsmap := make(map[string]qeval.Stats, 60)
    rate := make(map[int][2]ann.Type)

    rate[24] = [2]ann.Type{-0.40, -0.40}
    rate[25] = [2]ann.Type{-0.30, -0.30}
    rate[26] = [2]ann.Type{-0.24, -0.24}
    rate[27] = [2]ann.Type{-0.17, -0.17}
    rate[28] = [2]ann.Type{-0.38, -0.38}
    rate[29] = [2]ann.Type{-0.30, -0.38}
    rate[30] = [2]ann.Type{-0.30, -0.38}
    rate[31] = [2]ann.Type{-0.02, -0.40}
    rate[32] = [2]ann.Type{-0.10, -0.42}
    rate[33] = [2]ann.Type{0.05, -0.45}
    rate[34] = [2]ann.Type{0.10, -0.48}
    rate[35] = [2]ann.Type{0.12, -0.50}
    rate[36] = [2]ann.Type{0.10, -0.52}
    rate[37] = [2]ann.Type{-0.10, -0.55}
    rate[38] = [2]ann.Type{-0.15, -0.55}
    rate[39] = [2]ann.Type{-0.15, -0.55}
    rate[40] = [2]ann.Type{-0.20, -0.55}
    rate[41] = [2]ann.Type{-0.20, -0.55}
    rate[42] = [2]ann.Type{-0.20, -0.55}

    for t := 0; t < 60; t++ {
        pth := path.Join(dir, fmt.Sprintf("%02d.exam", t))
        if f.Open(pth) != nil {
            continue
        }

        var (
            sum, cor, incor, unknow     int
            wsum, wcor, wincor, wunknow int
            pwsum, pwcor, pwincor       int
            lsum, lcor, lincor, lunknow int
            plsum, plcor, plincor       int
            z                           ann.Type
            tmpinput                    [32]ann.Type
        )
        var wrate, lrate ann.Type = 0.10, -0.55
        if _, ok := rate[t]; ok {
            wrate, lrate = rate[t][0], rate[t][1]
        }
        for {
            if rcd, err := f.ReadOneRecord(); err != nil {
                break
            } else {
                input := qeval.NewAnnInput(rcd.H, rcd.V, int(rcd.S0), int(rcd.S1), tmpinput[:])
                if *pturn < 0 {
                    z = anns.Models[t].Run(input)[0]
                } else {
                    z = anns.Models[*pturn].Run(input)[0]
                }
                if (z > wrate && rcd.Z > 0) || (z < lrate && rcd.Z < 0) {
                    cor++
                    if rcd.Z > 0 {
                        wcor++
                    } else {
                        lcor++
                    }
                    if z > wrate {
                        pwcor++
                    } else if z < lrate {
                        plcor++
                    }
                } else if z > lrate && z < wrate {
                    unknow++
                    if rcd.Z > 0 {
                        wunknow++
                    } else {
                        lunknow++
                    }
                } else {
                    incor++
                    if rcd.Z > 0 {
                        wincor++
                    } else {
                        lincor++
                    }
                    if z > wrate {
                        pwincor++
                    } else if z < lrate {
                        plincor++
                    }
                }
                if sum++; rcd.Z > 0 {
                    wsum++
                } else {
                    lsum++
                }
                if z > wrate {
                    pwsum++
                } else if z < lrate {
                    plsum++
                }
            }
        }
        f.Close()
        stats := new(qeval.Stats)
        stats.Wrate, stats.Lrate = float64(wrate), float64(lrate)
        if pwsum == 0 {
            stats.Pwcor = 0
        } else {
            stats.Pwcor = float64(pwcor) / float64(pwsum)
        }
        if plsum == 0 {
            stats.Plcor = 0
        } else {
            stats.Plcor = float64(plcor) / float64(plsum)
        }
        tmp := fmt.Sprintf("%d", t)
        statsmap[tmp] = *stats
        fmt.Printf("%s:\n\t\t\t\tSum: %d, Correct: %d(%.2f%%), Incorrect: %d(%.2f%%), Unknown: %d(%.2f%%)\n",
            pth, sum, cor, float64(cor)*100/float64(sum), incor, float64(incor)*100/float64(sum), unknow,
            float64(unknow)*100/float64(sum))
        fmt.Printf("\t\t\t\tWSum: %d, WCorrect: %d(%.2f%%), WIncorrect: %d(%.2f%%), WUnknown: %d(%.2f%%)\n",
            wsum, wcor, float64(wcor)*100/float64(wsum), wincor, float64(wincor)*100/float64(wsum), wunknow,
            float64(wunknow)*100/float64(wsum))
        fmt.Printf("\t\t\t\tLSum: %d, LCorrect: %d(%.2f%%), LIncorrect: %d(%.2f%%), LUnknown: %d(%.2f%%)\n",
            lsum, lcor, float64(lcor)*100/float64(lsum), lincor, float64(lincor)*100/float64(lsum), lunknow,
            float64(lunknow)*100/float64(lsum))
        fmt.Printf("\t\t\t\t>PWSum: %d, PWCorrect: %d(%.2f%%), PWIncorrect: %d(%.2f%%)\n",
            pwsum, pwcor, stats.Pwcor*100, pwincor, float64(pwincor)*100/float64(pwsum))
        fmt.Printf("\t\t\t\t>PLSum: %d, PLCorrect: %d(%.2f%%), PLIncorrect: %d(%.2f%%)\n",
            plsum, plcor, stats.Plcor*100, plincor, float64(plincor)*100/float64(plsum))
    }

    if content, err := json.MarshalIndent(statsmap, "", "    "); err == nil {
        if f, err := os.Create(path.Join(annmodelsdir, "stats")); err == nil {
            f.Write(content)
            f.Close()
            fmt.Printf("Stats has been saved at %s\n", path.Join(annmodelsdir, "stats"))
        }
    } else {
        panic(err)
    }
}
