package main

import (
	"crypto/md5"
	"fmt"
	"hash/crc32"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var mut = &sync.Mutex{}

type job func(in, out chan interface{})

var debug map[string]string = map[string]string{}
var result string = ""

func main() {
	inputData := []int{0, 1, 2, 3, 4, 5}
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
	fmt.Println(result)
}

func jobWorker(job job, in, out chan interface{}, wg *sync.WaitGroup) {
	defer wg.Done()
	defer close(out)
	job(in, out)
}

var SingleHash = func(in, out chan interface{}) {
	mu := &sync.Mutex{}
	wgSH := &sync.WaitGroup{}

	for input := range in {
		wgSH.Add(1)
		go func(input interface{}) {
			defer wgSH.Done()
			value := input.(int)
			dataStr := strconv.Itoa(value)
			mu.Lock()
			md5Data := DataSignerMd5(dataStr)
			mu.Unlock()

			var crc32Hash (map[string]string) = map[string]string{
				"data":    dataStr,
				"md5Data": md5Data,
			}
			fmt.Println(crc32Hash["data"])
			fmt.Println(crc32Hash["md5Data"])
			wg := &sync.WaitGroup{}
			//Разпаралеливание crc32 вычислений
			for item := range crc32Hash {
				wg.Add(1)
				go func(item string) {
					defer wg.Done()
					crc32Data := DataSignerCrc32(crc32Hash[item])
					mu.Lock()
					crc32Hash[item] = crc32Data
					mu.Unlock()
				}(item)
			}
			wg.Wait()
			result := crc32Hash["data"] + "~" + crc32Hash["md5Data"]
			out <- result
			mut.Lock()
			debug[result] = "\n" + dataStr + " SingleHash data " + dataStr + "\n" +
				dataStr + " SingleHash md5(data) " + dataStr + "  " + md5Data + "\n" +
				dataStr + " SingleHash crc32(md5(data)) " + crc32Hash["md5Data"] + "\n" +
				dataStr + " SingleHash crc32(data) " + crc32Hash["data"] + "\n" +
				dataStr + " SingleHash result " + result + "\n"
			mut.Unlock()
		}(input)
	}
	wgSH.Wait()
}

func MultiHash(in, out chan interface{}) {
	mu := &sync.Mutex{}
	wg := &sync.WaitGroup{}
	for item := range in {
		wg.Add(1)
		go func(item interface{}) {
			txt := ""
			defer wg.Done()
			// value := item.(int)
			data := item.(string)
			th := []int{0, 1, 2, 3, 4, 5}
			for i := 0; i < len(th); i++ {
				buf := strconv.Itoa(th[i]) + data
				bufCrc32 := DataSignerCrc32(buf)
				mut.Lock()
				txt += bufCrc32
				debug[data] += data + " MultiHash: crc32(" + buf + ") " + string(th[i]) + " " + bufCrc32 + "\n"
				mut.Unlock()
				// atomic.AddInt32(&idx, 1)
			}
			mu.Lock()
			out <- txt
			mu.Unlock()
			mut.Lock()
			debug[data] += data + " MultiHash result: " + txt + "\n"
			mut.Unlock()
		}(item)
	}
	wg.Wait()
}

func CombineResults(in, out chan interface{}) {
	mu := &sync.Mutex{}
	wg := &sync.WaitGroup{}
	//txt := ""
	i := 0
	var slice []string
	for item := range in {
		wg.Add(1)
		go func(item interface{}) {
			defer wg.Done()
			mu.Lock()
			slice = append(slice, item.(string))
			i++
			mu.Unlock()
		}(item)
	}
	wg.Wait()
	sort.Strings(slice)
	result = "\nCombineResults  " + strings.Join(slice, "_") + "\n"
	out <- strings.Join(slice, "_")
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
