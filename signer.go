package main

import (
	"crypto/md5"
	"fmt"
	"hash/crc32"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

type job func(in, out chan interface{})

func main() {
	inputData := []int{0, 1}
	hashSignJobs := []job{
		job(func(in, out chan interface{}) {
			for _, fibNum := range inputData {
				out <- fibNum
			}
		}),
		job(SingleHash),
		//job(MultiHash),
		//job(CombineResults),
	}
	ExecutePipeline(hashSignJobs...)
}

func ExecutePipeline(jobs ...job) {
	wg := &sync.WaitGroup{}
	in := make(chan interface{}, 1)
	quotaCh := make(chan struct{}, 2)
	for _, job := range jobs {
		out := make(chan interface{}, 1)
		wg.Add(2)
		go jobWorker(job, in, out, wg)
		in = out
		go SingleHashWorker(in, out, wg, quotaCh)
	}
	wg.Wait()
	defer close(in)
	defer close(quotaCh)
}

func jobWorker(job job, in, out chan interface{}, wg *sync.WaitGroup) {
	job(in, out)
	defer wg.Done()
}

func SingleHashWorker(in, out chan interface{}, wg *sync.WaitGroup, quotaCh chan struct{}) {
	for _ = range in {
		quotaCh <- struct{}{}
		SingleHash(in, out)
		<-quotaCh
	}
	defer wg.Done()
}

var SingleHash = func(in, out chan interface{}) {
	for item := range in {
		dataStr := item.(int)
		md5Data := DataSignerMd5(strconv.Itoa(dataStr))
		crc32DataWithMd5 := DataSignerCrc32(md5Data)
		crc32Data := DataSignerCrc32(strconv.Itoa(dataStr))
		result := crc32Data + "~" + crc32DataWithMd5

		out <- result // Результат перезаписывается в int в цикле, из-за этого случается deadlock

		fmt.Printf("%v SingleHash data %v\n", dataStr, dataStr)
		fmt.Printf("%v SingleHash md5(data) %v\n", dataStr, md5Data)
		fmt.Printf("%v SingleHash crc32(md5(data)) %v\n", dataStr, crc32DataWithMd5)
		fmt.Printf("%v SingleHash crc32(data) %v\n", dataStr, crc32Data)
		fmt.Printf("%v SingleHash result %v\n", dataStr, result)
	}
	//defer close(in)
}

func multiHash(data string) {
	th := []int{0, 1, 2, 3, 4, 5}
	valueMap := make(map[int]string)
	txt := ""
	for i := 0; i < len(th); i++ {
		buf := strconv.Itoa(th[i]) + data
		bufCrc32 := DataSignerCrc32(buf)
		fmt.Printf("%v MultiHash: crc32(%v) %v %v\n", data, buf, th[i], bufCrc32)
		valueMap[i] = bufCrc32
		txt += bufCrc32
	}
	fmt.Printf("%v MultiHash: result: %v\n", data, txt)
}

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
