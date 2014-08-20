/*********************************************************************************
*     File Name           :     qrcds.go
*     Created By          :     YIMMON, yimmon.zhuang@gmail.com
*     Creation Date       :     [2014-05-18 21:38]
*     Last Modified       :     [2014-06-22 21:55]
*     Description         :
**********************************************************************************/

package qrcds

import (
    "algorithm/qboard"
    "bufio"
    "errors"
    "fmt"
    "os"
    "path"
    "sync"
    "sync/atomic"
)

const (
    numHashBlock        = 73
    hashSize            = 3000000 / numHashBlock
    BUFSIZE      int    = 1 * 1024 * 1024
    RCEFMT_R     string = "0x%x 0x%x %d %d %d\n"
    RCEFMT_W     string = "0x%08x 0x%08x %02d %02d %+d\n"
    RCDLEN       int    = 10 + 1 + 10 + 1 + 2 + 1 + 2 + 1 + 2 + 1
)

type Hash struct {
    MeanCount     []uint32
    HashTable     [numHashBlock]map[HashKey]*HashValue
    RWMutex       [numHashBlock]sync.RWMutex
    CleanCallback func(*HashKey, *HashValue)
}

type HashKey struct {
    H, V   int32
    S0, S1 int8
}

type HashValue struct {
    Count   uint32
    Z, Turn int8
}

func (self *Hash) Init(cleanCallback func(*HashKey, *HashValue)) {
    self.MeanCount = make([]uint32, numHashBlock)
    self.CleanCallback = cleanCallback
    for i := 0; i < numHashBlock; i++ {
        self.HashTable[i] = make(map[HashKey]*HashValue, hashSize)
        self.MeanCount[i] = 2
    }
}

func (self *Hash) NewHashKey(b *qboard.QBoard, hk *HashKey) *HashKey {
    hk.H, hk.V = b.H, b.V
    hk.S0, hk.S1 = int8(b.S[b.Now]), int8(b.S[b.Now^1])
    return hk
}

func (self *Hash) Query(b *qboard.QBoard) (win int, ok bool) {
    var hk HashKey
    val := self.GetHashValue(self.NewHashKey(b, &hk))
    if val != nil {
        win, ok = b.Now, true
        if val.Z == -1 {
            win = b.Now ^ 1
        }
        return
    }
    return 8, false
}

func (self *Hash) Add(b *qboard.QBoard, win int) {
    var hk HashKey
    var z int = -1
    if win == b.Now {
        z = 1
    }
    self.InsertHashValue(self.NewHashKey(b, &hk), b.Turn, z)
}

func (self *Hash) GetHashValue(k *HashKey) *HashValue {
    idx := int(k.H^k.V) % numHashBlock
    self.RWMutex[idx].RLock()
    if v, ok := self.HashTable[idx][*k]; ok {
        self.RWMutex[idx].RUnlock()
        atomic.AddUint32(&v.Count, 1)
        return v
    }
    self.RWMutex[idx].RUnlock()
    return nil
}

func (self *Hash) InsertHashValue(k *HashKey, turn, z int) *HashValue {
    idx := int(k.H^k.V) % numHashBlock
    val := &HashValue{Z: int8(z), Turn: int8(turn), Count: 1}
    self.RWMutex[idx].Lock()
    if len(self.HashTable[idx]) >= hashSize {
        self.CleanHashTable(idx)
    }
    self.HashTable[idx][*k] = val
    self.RWMutex[idx].Unlock()
    return val
}

func (self *Hash) CleanHashTable(idx int) int {
    sum, count := uint32(0), 0
    for k, v := range self.HashTable[idx] {
        sum += v.Count
        if v.Count < self.MeanCount[idx] {
            self.CleanCallback(&k, v)
            delete(self.HashTable[idx], k)
            count++
        } else {
            v.Count = 0
        }
    }
    self.MeanCount[idx] = sum / uint32(hashSize)
    return count
}

func (self *Hash) ClearAll() {
    for i, ht := range self.HashTable {
        for k, _ := range ht {
            delete(ht, k)
        }
        self.MeanCount[i] = 2
    }
}

type Record struct {
    H, V      int32
    S0, S1, Z int8
}

type SortRecord []*Record

func (self SortRecord) Len() int      { return len(self) }
func (self SortRecord) Swap(i, j int) { self[i], self[j] = self[j], self[i] }
func (self SortRecord) Less(i, j int) bool {
    return Less(self[i], self[j])
}

func Less(r1, r2 *Record) bool {
    if r1 == nil {
        return false
    }
    if r2 == nil {
        return true
    }
    if r1.H == r2.H {
        if r1.V == r2.V {
            if r1.S0 == r2.S0 {
                if r1.S1 == r2.S1 {
                    return r1.Z < r2.Z
                }
                return r1.S1 < r2.S1
            }
            return r1.S0 < r2.S0
        }
        return r1.V < r2.V
    }
    return r1.H < r2.H
}

func (self *Record) Equal(r *Record) bool {
    if self.H == r.H && self.V == r.V &&
        self.S0 == r.S0 && self.S1 == r.S1 &&
        self.Z == r.Z {
        return true
    }
    return false
}

type ByString []string

func (self ByString) Len() int      { return len(self) }
func (self ByString) Swap(i, j int) { self[i], self[j] = self[j], self[i] }
func (self ByString) Less(i, j int) bool {
    return self[i] < self[j]
}

type File struct {
    filepath string
    fp       *os.File
    reader   *bufio.Reader
    writer   *bufio.Writer
    mutex    sync.Mutex
}

func (self *File) Open(filepath string) (err error) {
    self.mutex.Lock()
    defer self.mutex.Unlock()
    if !self.null() {
        self.close()
    }
    if self.fp, err = os.Open(filepath); err != nil {
        self.reset()
        return
    }
    self.filepath = filepath
    if self.reader == nil {
        self.reader = bufio.NewReaderSize(self.fp, BUFSIZE)
    } else {
        self.reader.Reset(self.fp)
    }
    self.writer = nil
    return
}

func (self *File) Create(filepath string) (err error) {
    self.mutex.Lock()
    defer self.mutex.Unlock()
    if !self.null() {
        self.close()
    }
    if self.fp, err = os.Create(filepath); err != nil {
        self.reset()
        return
    }
    self.filepath = filepath
    if self.writer == nil {
        self.writer = bufio.NewWriterSize(self.fp, BUFSIZE)
    } else {
        self.writer.Reset(self.fp)
    }
    self.reader = nil
    return
}

func (self *File) Seek(num int64) (ret int64, err error) {
    self.mutex.Lock()
    defer self.mutex.Unlock()
    if self.null() {
        return 0, errors.New("This is a null file, cannot seek.")
    }
    if ret, err = self.fp.Seek(int64(RCDLEN)*num, 0); err != nil {
        return
    }
    ret /= int64(RCDLEN)
    if self.reader != nil {
        self.reader.Reset(self.fp)
    }
    if self.writer != nil {
        self.writer.Reset(self.fp)
    }
    return
}

func Line2Record(line string, rcd *Record) {
    fmt.Sscanf(line, RCEFMT_R,
        &rcd.H, &rcd.V, &rcd.S0, &rcd.S1, &rcd.Z)
    return
}

func (self *File) ReadOneRecord() (rcd *Record, err error) {
    self.mutex.Lock()
    defer self.mutex.Unlock()
    if self.reader == nil {
        return nil, errors.New("Null reader, cannot read record.")
    }
    rcd = new(Record)
    _, err = fmt.Fscanf(self.reader, RCEFMT_R,
        &rcd.H, &rcd.V, &rcd.S0, &rcd.S1, &rcd.Z)
    return
}

func (self *File) WriteOneRecord(rcd *Record) (err error) {
    self.mutex.Lock()
    defer self.mutex.Unlock()
    if self.writer == nil {
        return errors.New("Null writer, cannot write record.")
    }
    _, err = fmt.Fprintf(self.writer, RCEFMT_W,
        rcd.H, rcd.V, rcd.S0, rcd.S1, rcd.Z)
    return
}

func (self *File) ReadOneLine() (line string, err error) {
    self.mutex.Lock()
    defer self.mutex.Unlock()
    if self.reader == nil {
        return "", errors.New("Null reader, cannot read line.")
    }
    line, err = self.reader.ReadString('\n')
    return
}

func (self *File) WriteOneLine(line string) (err error) {
    self.mutex.Lock()
    defer self.mutex.Unlock()
    if self.writer == nil {
        return errors.New("Null writer, cannot write line.")
    }
    _, err = self.writer.WriteString(line)
    return
}

func (self *File) NumRecords() int {
    self.mutex.Lock()
    defer self.mutex.Unlock()
    return self.numRecords()
}

func (self *File) numRecords() int {
    fi, err := self.fp.Stat()
    if err != nil {
        return -1
    }
    return int(fi.Size() / int64(RCDLEN))
}

func (self *File) GetDir() string {
    self.mutex.Lock()
    defer self.mutex.Unlock()
    return path.Dir(self.filepath)
}

func (self *File) GetBase() string {
    self.mutex.Lock()
    defer self.mutex.Unlock()
    return path.Base(self.filepath)
}

func (self *File) Move(newpath string) (err error) {
    self.mutex.Lock()
    defer self.mutex.Unlock()
    return self.move(newpath)
}

func (self *File) move(newpath string) (err error) {
    filepath := self.filepath
    self.close()
    if filepath == newpath {
        return
    }
    return os.Rename(filepath, newpath)
}

func (self *File) Null() bool {
    self.mutex.Lock()
    defer self.mutex.Unlock()
    return self.null()
}

func (self *File) null() bool {
    if self.fp != nil {
        return false
    }
    return true
}

func (self *File) Sync() {
    self.mutex.Lock()
    defer self.mutex.Unlock()
    self.sync()
}

func (self *File) sync() {
    if self.writer != nil {
        self.writer.Flush()
    }
    self.fp.Sync()
}

func (self *File) Close() (err error) {
    self.mutex.Lock()
    defer self.mutex.Unlock()
    return self.close()
}

func (self *File) close() (err error) {
    if self.null() {
        return errors.New("This is a null file, cannot close.")
    }
    self.sync()
    err = self.fp.Close()
    self.reset()
    return
}

func (self *File) Reset() {
    self.mutex.Lock()
    defer self.mutex.Unlock()
    self.reset()
}

func (self *File) reset() {
    self.filepath = ""
    self.fp = nil
}

func (self *File) Free() {
    self.mutex.Lock()
    defer self.mutex.Unlock()
    self.free()
}

func (self *File) free() {
    self.close()
    self.reader, self.writer = nil, nil
}
