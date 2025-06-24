package hw

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

type job func(in, out chan interface{})

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

var md5Mutex sync.Mutex

func SingleHash(in, out chan interface{}) {
	var wg sync.WaitGroup
	for v := range in {
		wg.Add(1)
		go func(v interface{}) {
			defer wg.Done()
			vStr := fmt.Sprintf("%v", v)

			// Сначала синхронно получаем MD5 (только один вызов одновременно)
			md5Mutex.Lock()
			md5 := DataSignerMd5(vStr)
			md5Mutex.Unlock()

			// Теперь параллельно вычисляем оба CRC32
			var crc32Data, crc32Md5 string
			var innerWg sync.WaitGroup

			innerWg.Add(2)
			go func() {
				defer innerWg.Done()
				crc32Data = DataSignerCrc32(vStr)
			}()

			go func() {
				defer innerWg.Done()
				crc32Md5 = DataSignerCrc32(md5)
			}()

			innerWg.Wait()
			out <- crc32Data + "~" + crc32Md5
		}(v)
	}
	wg.Wait()
}

func MultiHash(in, out chan interface{}) {
	var wg sync.WaitGroup
	for v := range in {
		wg.Add(1)
		go func(v interface{}) {
			defer wg.Done()
			input := fmt.Sprintf("%v", v)
			results := make([]string, 6)
			var innerWg sync.WaitGroup
			for th := 0; th < 6; th++ {
				innerWg.Add(1)
				go func(th int) {
					defer innerWg.Done()
					results[th] = DataSignerCrc32(strconv.Itoa(th) + input)
				}(th)
			}
			innerWg.Wait()
			out <- strings.Join(results, "")
		}(v)
	}
	wg.Wait()
}

func CombineResults(in, out chan interface{}) {
	var results []string
	for v := range in {
		results = append(results, v.(string))
	}
	sort.Strings(results)
	out <- strings.Join(results, "_")
}

func ExecutePipeline(jobs ...job) {
	if len(jobs) == 0 {
		return
	}

	var in chan interface{}
	var out chan interface{}
	for _, j := range jobs {
		out = make(chan interface{})
		go func(job job, in, out chan interface{}) {
			defer close(out)
			job(in, out)
		}(j, in, out)
		in = out
	}

	if out != nil {
		for range out {
			// Просто читаем из канала до его закрытия
		}
	}
}
