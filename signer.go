package main

import (
	"crypto/md5"
	"fmt"
	"hash/crc32"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

type job func(in, out chan interface{})

var debug map[string]string = map[string]string{}
var idx int32 = 0

func main() {
	inputData := []int{0, 1, 2, 3, 4}
	hashSignJobs := []job{
		job(func(in, out chan interface{}) {
			for _, fibNum := range inputData {
				out <- fibNum
			}
		}),
		job(SingleHash),
		job(MultiHash),
		//job(CombineResults),
	}
	ExecutePipeline(hashSignJobs...)
}

func ExecutePipeline(jobs ...job) {
	wg := &sync.WaitGroup{}
	in := make(chan interface{}, 100)
	out := make(chan interface{}, 100)
	for _, job := range jobs {
		wg.Add(1)
		go jobWorker(job, in, out, wg)
		in = out
		out = make(chan interface{}, 100)
	}
	wg.Wait()
	qw := make([]string, len(debug), len(debug))
	for _, item := range debug {
		qw = append(qw, item)
	}

	sort.Strings(qw)

	for _, item := range qw {
		fmt.Print(item)
	}
}

func jobWorker(job job, in, out chan interface{}, wg *sync.WaitGroup) {
	defer wg.Done()
	defer close(out)
	job(in, out)
}

var SingleHash = func(in, out chan interface{}) {
	mu := &sync.Mutex{}
	wg := &sync.WaitGroup{}

	for input := range in {
		wg.Add(1)
		go func(input interface{}) {
			defer wg.Done()
			value := input.(int)
			dataStr := strconv.Itoa(value)
			mu.Lock()
			md5Data := DataSignerMd5(dataStr)
			mu.Unlock()
			crc32DataWithMd5 := DataSignerCrc32(md5Data)
			crc32Data := DataSignerCrc32(dataStr)
			result := crc32Data + "~" + crc32DataWithMd5
			out <- result
			mu.Lock()
			debug[result] = "\n" + dataStr + " SingleHash data " + dataStr + "\n" +
				dataStr + " SingleHash md5(data) " + dataStr + "  " + md5Data + "\n" +
				dataStr + " SingleHash crc32(md5(data)) " + crc32DataWithMd5 + "\n" +
				dataStr + " SingleHash crc32(data) " + crc32Data + "\n" +
				dataStr + " SingleHash result " + result + "\n"
			mu.Unlock()
			//			atomic.AddInt32(&idx, 1)
		}(input)
	}
	wg.Wait()
}

func MultiHash(in, out chan interface{}) {
	mut := &sync.Mutex{}
	wg := &sync.WaitGroup{}
	//txt := ""
	for item := range in {
		wg.Add(1)
		go func(item interface{}) {
			defer wg.Done()
			// value := item.(int)
			data := item.(string)
			th := []int{0, 1, 2, 3, 4, 5}
			for i := 0; i < len(th); i++ {
				buf := strconv.Itoa(th[i]) + data
				bufCrc32 := DataSignerCrc32(buf)
				mut.Lock()
				debug[data] += data + " MultiHash: crc32(" + buf + ") " + string(th[i]) + " " + bufCrc32 + "\n"
				mut.Unlock()
				//atomic.AddInt32(&idx, 1)
			}
		}(item)
	}
	wg.Wait()
}

//Not need change!

const (
	MaxInputDataLen = 100
)

var (
	dataSignerOverheat uint32 = 0
	DataSignerSalt            = ""
)

var OverheatLock = func() {
	for {
		if swapped := atomic.CompareAndSwapUint32(&dataSignerOverheat, 0, 1); !swapped {
			fmt.Println("OverheatLock happend")
			time.Sleep(time.Second)
		} else {
			break
		}
	}
}

var OverheatUnlock = func() {
	for {
		if swapped := atomic.CompareAndSwapUint32(&dataSignerOverheat, 1, 0); !swapped {
			fmt.Println("OverheatUnlock happend")
			time.Sleep(time.Second)
		} else {
			break
		}
	}
}

var DataSignerMd5 = func(data string) string {
	OverheatLock()
	defer OverheatUnlock()
	data += DataSignerSalt
	dataHash := fmt.Sprintf("%x", md5.Sum([]byte(data)))
	time.Sleep(10 * time.Millisecond)
	return dataHash
}

var DataSignerCrc32 = func(data string) string {
	data += DataSignerSalt
	crcH := crc32.ChecksumIEEE([]byte(data))
	dataHash := strconv.FormatUint(uint64(crcH), 10)
	time.Sleep(time.Second)
	return dataHash
}
