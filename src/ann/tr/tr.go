/*********************************************************************************
*     File Name           :     tr.go
*     Created By          :     YIMMON, yimmon.zhuang@gmail.com
*     Creation Date       :     [2014-05-01 18:54]
*     Last Modified       :     [2014-05-22 13:05]
*     Description         :
**********************************************************************************/

package main

import (
    "ann"
    "ann/eval"
    "ann/rcds"
    "flag"
    "fmt"
    "math"
    "math/rand"
    "os"
    "path"
    "runtime"
    "time"
)

type Tr struct {
    TimeStamp                       int64
    DataSet                         rcds.File
    Ann                             *ann.Ann
    DsDir, AnnDir                   string
    MinTurn, TurnLength, Num, Epoch int
}

const (
    EpochNum             = 5
    EpochSize            = 1000000
    DesiredError         = 0.052
    MaxEpochsPerTrain    = 300
    EpochsBetweenReports = 10
)

var (
    numThread   int = 1
    pminturn        = flag.Int("minturn", 99, "Min turn.")
    pturnlength     = flag.Int("turnlength", 6, "Turn Length.")
    pnum            = flag.Int("num", 0, "Which dataset used to train the ann model.")
    pepoch          = flag.Int("epoch", 3, "The number of epochs.")
    phelp           = flag.Bool("help", false, "Help")
    beginTime   time.Time
)

func (self *Tr) CreateNewAnn() {
    self.Ann = ann.CreateStandard(3, 25, 7, 1)
    self.Ann.SetActivationFunctionHidden(ann.SIGMOID_SYMMETRIC_STEPWISE)
    self.Ann.SetActivationFunctionOutput(ann.SIGMOID_SYMMETRIC)
    self.Ann.SetActivationSteepnessHidden(0.5)
    self.Ann.SetActivationSteepnessOutput(0.5)
    self.Ann.RandomizeWeights(-0.1, 0.1)
    self.Ann.SetBitFailLimit(0.27)
}

func (self *Tr) ReadAnn() {
    filepath := path.Join(self.AnnDir, "latest.ann")
    if f, err := os.Open(filepath); err == nil {
        f.Close()
        self.Ann = ann.CreateFromFile(filepath)
        fmt.Printf("Read %s\n", filepath)
    } else {
        self.CreateNewAnn()
        fmt.Println("Create new ann model.")
    }
    //self.Ann.PrintParameters()
}

func (self *Tr) Train() {
    defer runtime.GC()
    if err := self.DataSet.Open(path.Join(self.DsDir, "TrainSets", "BALANCE.train")); err != nil {
        panic(err)
    }
    defer self.DataSet.Close()

    self.Ann.SetTrainingAlgorithm(ann.TRAIN_RPROP)

    fmt.Println("\nTraining")
    var data = make([][]ann.Type, 0, EpochSize)
    numRecords := self.DataSet.NumRecords()
    odd := numRecords/(EpochSize*EpochNum) + 1
    for epoch := 1; epoch <= self.Epoch; epoch++ {
        self.DataSet.Seek(0)
        fmt.Printf("Epoch %d\n", epoch)

        count := 0
        rcd := new(rcds.Record)
        for eof := false; !eof; {
            data = data[:0]
            for len(data) < EpochSize {
                if line, err := self.DataSet.ReadOneLine(); err != nil {
                    eof = true
                    break
                } else if rand.Intn(odd) == 0 {
                    rcds.Line2Record(line, rcd)
                    tmp := eval.NewAnnInput(rcd.H, rcd.V, int8(rcd.S0), int8(rcd.S1))
                    tmp = append(tmp, ann.Type(rcd.Z))
                    data = append(data, tmp)
                }
            }
            count += len(data)
            fmt.Printf("Count %d, Cur Epoch %d\n", count, epoch)
            self.TrainOnData(data)
        }
    }

    fmt.Println("Train() Done.")
}

func (self *Tr) TrainOnData(data [][]ann.Type) {
    var getData ann.GetTrainDataCallback = func(num, numInput, numOutput uint, input, output []ann.Type) {
        for i := uint(0); i < numInput; i++ {
            input[i] = ann.Type(data[num][i])
        }
        for i := uint(0); i < numOutput; i++ {
            output[i] = ann.Type(data[num][numInput+i] * 0.9) // -0.9 and 0.9
        }
    }
    trainData := ann.CreateTrainFromCallback(uint(len(data)), self.Ann.GetNumInput(), self.Ann.GetNumOutput(), getData)
    defer trainData.Destroy()
    trainData.Shuffle()
    self.Ann.ResetMSE()
    self.Ann.TrainOnData(trainData, MaxEpochsPerTrain, EpochsBetweenReports, DesiredError)
}

func (self *Tr) Test() {
    defer runtime.GC()
    var (
        data    = make([][]ann.Type, 0, 1024)
        testdir = path.Join(self.DsDir, "TestSets")
    )
    os.MkdirAll(self.AnnDir, os.ModePerm)
    logfile, _ := os.Create(path.Join(self.AnnDir, fmt.Sprintf("%d.log", self.TimeStamp)))
    logTest := func(str string) {
        fmt.Print(str)
        fmt.Fprint(logfile, str)
    }
    defer logfile.Close()

    logTest(fmt.Sprintln("\nTesting"))
    for num := 0; num < 60; num++ {
        filepath := path.Join(testdir, fmt.Sprintf("%02d.test", num))
        if err := self.DataSet.Open(filepath); err != nil {
            continue
        }

        data = data[:0]
        for {
            if rcd, err := self.DataSet.ReadOneRecord(); err != nil {
                break
            } else {
                tmp := eval.NewAnnInput(rcd.H, rcd.V, int8(rcd.S0), int8(rcd.S1))
                tmp = append(tmp, ann.Type(rcd.Z))
                data = append(data, tmp)
            }
        }
        if len(data) > 0 {
            self.TestOnData(data)
            logTest(fmt.Sprintf("%02d.test: MSE %f, Bit fail %d, Sum %d, Succ %.2f%%\n",
                num, self.Ann.GetMSE(), self.Ann.GetBitFail(), len(data),
                float32(len(data)-int(self.Ann.GetBitFail()))*100/float32(len(data))))
        }
        self.DataSet.Close()
    }

    logTest(fmt.Sprintln("Test() Done."))
}

func (self *Tr) TestOnData(data [][]ann.Type) {
    var getData ann.GetTrainDataCallback = func(num, numInput, numOutput uint, input, output []ann.Type) {
        for i := uint(0); i < numInput; i++ {
            input[i] = ann.Type(data[num][i])
        }
        for i := uint(0); i < numOutput; i++ {
            output[i] = ann.Type(data[num][numInput+i] * 0.9) // -0.9 and 0.9
        }
    }
    testData := ann.CreateTrainFromCallback(uint(len(data)), self.Ann.GetNumInput(), self.Ann.GetNumOutput(), getData)
    defer testData.Destroy()
    self.Ann.ResetMSE()
    self.Ann.TestData(testData)
}

func (self *Tr) Save() {
    fmt.Println("\nSaving")
    targetpath := self.GetTargetPath()
    os.MkdirAll(path.Dir(targetpath), os.ModePerm)
    if self.Ann.Save(targetpath) != 0 {
        panic("Fann save fail: " + targetpath)
    }
    fmt.Printf("New ann model has been saved at %s\n", targetpath)
    rep := ""
    for rep != "y" && rep != "n" {
        fmt.Print("Do you want to create symlink to new model? [y/n] ")
        fmt.Scanf("%s", &rep)
    }
    if rep == "y" {
        sympath := path.Join(path.Dir(targetpath), "latest.ann")
        os.Remove(sympath)
        if err := os.Symlink(path.Base(targetpath), sympath); err != nil {
            fmt.Println(err)
        } else {
            fmt.Printf("%s now points to %s\n", sympath, path.Base(targetpath))
        }
    }
    fmt.Println("Save() Done.")
}

func (self *Tr) GetTargetPath() string {
    a, b := self.Exam()
    filename := fmt.Sprintf("%d-%.0f%%-%.0f%%.ann", self.TimeStamp, a, b)
    return path.Join(self.AnnDir, filename)
}

func (self *Tr) Exam() (a, b float64) {
    var (
        na, nb, nn int = 0, 0, 0
        data           = make([][]ann.Type, 0, 2048)
    )
    fmt.Println("Examing")
    for turn := 0; turn < 60; turn++ {
        exampath := path.Join("./DataSets", "ExamSets", fmt.Sprintf("%02d.exam", turn))
        if self.DataSet.Open(exampath) != nil {
            continue
        }

        data = data[:0]
        for {
            if rcd, err := self.DataSet.ReadOneRecord(); err != nil {
                break
            } else {
                input := eval.NewAnnInput(rcd.H, rcd.V, int8(rcd.S0), int8(rcd.S1))
                z := self.Ann.Run(input)[0]
                //fmt.Printf("Z: %d, z: %.3f\n", rcd.Z, z)
                if math.Abs(float64(z)-float64(rcd.Z)*0.9) < 0.55 {
                    na++
                }
                if (float64(z) * float64(rcd.Z)) > 0 {
                    nb++
                }
                nn++
                input = append(input, ann.Type(rcd.Z))
                data = append(data, input)
            }
        }
        if len(data) > 0 {
            self.TestOnData(data)
            fmt.Printf("%02d.exam: MSE %f, Bit fail %d, Sum %d, Succ %.2f%%\n",
                turn, self.Ann.GetMSE(), self.Ann.GetBitFail(), len(data),
                float32(len(data)-int(self.Ann.GetBitFail()))*100/float32(len(data)))
        }
        self.DataSet.Close()
    }
    a = float64(na) * 100 / float64(nn)
    b = float64(nb) * 100 / float64(nn)
    return
}

func (self *Tr) Clean() {
    fmt.Println("\nCleaning")
    self.Ann.Destroy()
    self.DataSet.Free()
    fmt.Println("Clean() Done.")
}

func init() {
    beginTime = time.Now()
    rand.Seed(time.Now().Unix())
    numThread = runtime.NumCPU()
    runtime.GOMAXPROCS(numThread)
    flag.Parse()
    if *phelp {
        fmt.Println("Usage: tr --minturn Num --turnlength Num --num Num --epoch Num")
        os.Exit(0)
    }
    if *pepoch < 1 {
        fmt.Println("epoch must >= 1.")
        os.Exit(0)
    }
}

func main() {
    var tr Tr
    tr.TimeStamp = time.Now().Unix()
    tr.MinTurn, tr.TurnLength, tr.Num, tr.Epoch = *pminturn, *pturnlength, *pnum, *pepoch
    tr.DsDir = path.Join("./DataSets", fmt.Sprintf("%02d-%02d", *pminturn, *pturnlength),
        fmt.Sprintf("%02d", *pnum))
    tr.AnnDir = path.Join("./AnnModels", fmt.Sprintf("%02d-%02d", *pminturn, *pturnlength))
    tr.ReadAnn()

    tr.Train()
    tr.Test()
    tr.Save()
    tr.Clean()

    fmt.Println("Finish. ", time.Since(beginTime))
}
