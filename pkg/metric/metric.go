package metric

import (
	"fmt"
	"gateway/pkg/utils"
	"gateway/pkg/version"
	"os/exec"
	"runtime/debug"
	"runtime/metrics"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	StartTime     = time.Now() // Startup Time
	Data          string
	lastTCPStatus = make(map[string]int)

	CountConnection         atomic.Int64 // Connection Count
	CountPublicTCPRequest   atomic.Int64 // agent TCP Request Count
	CountPublicHTTPRequest  atomic.Int64 // agent HTTP Request Count
	CountPrivateHTTPRequest atomic.Int64 // private Request Count
	CountGoroutine          atomic.Uint64
	CountFreeMemory         atomic.Uint64
	CountReleasedMemory     atomic.Uint64
	CountStackMemory        atomic.Uint64
	CountUnusedMemory       atomic.Uint64
	CountTotalMemory        atomic.Uint64
	CountObjectsMemory      atomic.Uint64

	P99PublicHTTPRequestLatency  ProtoP99 = ProtoP99{} // public http Request Duration p99
	P99PrivateHTTPRequestLatency ProtoP99 = ProtoP99{} // private http Request Duration p99

	Buckets = []int64{1, 5, 10, 20, 50, 100, 150, 200, 250, 300, 400, 500, 1000, 2000, 5000, 10000, 9999999999} // 毫秒
)

type ProtoP99 sync.Map

type P99 struct {
	count   atomic.Int64
	datas   sync.Map
	buckets []int64
}

func NewP99(buckets []int64) *P99 {
	b := new(P99)
	b.buckets = buckets
	for _, bucket := range b.buckets {
		b.datas.Store(bucket, new(atomic.Int64))
	}
	return b
}

func (b *P99) Reset() {
	b.count.Store(0)
	for _, bucket := range b.buckets {
		v, _ := b.datas.Load(bucket)
		v.(*atomic.Int64).Store(0)
	}
}

func (b *P99) In(i int64) {
	for _, bucket := range b.buckets {

		if bucket >= i {
			v, _ := b.datas.Load(bucket)
			v.(*atomic.Int64).Add(1)
			b.count.Add(1)
			break
		}
	}
}

func (b *P99) Out() int64 {
	var sum int64
	count := b.count.Load()

	if count == 0 {
		return 0
	}

	for _, bucket := range b.buckets {
		v, _ := b.datas.Load(bucket)
		sum += v.(*atomic.Int64).Load()
		if float64(sum)/float64(count) >= 0.99 {
			return bucket
		}
	}

	return 0
}

func (pp99 *ProtoP99) In(proto string, value int64) {
	p := (*sync.Map)(pp99)
	v, ok := p.Load(proto)
	if !ok {
		v = NewP99(Buckets) // FIXME Edge Case: Duplicate Creation May Occur
		p.Store(proto, v)
	}

	p99, ok := v.(*P99)
	if !ok {
		return
	}
	p99.In(value)
}

func (pp99 *ProtoP99) Out() map[string]int64 {
	p := (*sync.Map)(pp99)
	ret := make(map[string]int64)

	p.Range(func(k, v any) bool {
		proto := k.(string)
		p99, _ := v.(*P99)
		vv := p99.Out()
		if vv > 0 {
			ret[proto] = vv
		}
		return true
	})

	return ret
}

func (pp99 *ProtoP99) Reset() {
	p := (*sync.Map)(pp99)
	p.Range(func(k, v any) bool {
		if p99, ok := v.(*P99); ok {
			p99.Reset()
		}

		return true
	})
}

func getRuntume() {
	// 定义要获取的指标
	metricsList := []string{
		"/sched/goroutines:goroutines",        // 当前协程数
		"/memory/classes/heap/free:bytes",     // 空闲堆内存字节数
		"/memory/classes/heap/released:bytes", // 已释放回操作系统的堆内存字节数
		"/memory/classes/heap/stacks:bytes",   // 堆栈使用的内存字节数
		"/memory/classes/heap/unused:bytes",   // 堆上未使用的内存字节数
		"/memory/classes/total:bytes",         // 总内存使用量（包括堆、栈、全局变量等）
		"/memory/classes/heap/objects:bytes",  // 堆上分配的对象占用的字节数
	}

	// 创建样本切片
	samples := make([]metrics.Sample, len(metricsList))
	for i, metric := range metricsList {
		samples[i].Name = metric
	}

	// 获取指标值
	metrics.Read(samples)

	// 打印结果
	for index, sample := range samples {
		if sample.Value.Kind() == metrics.KindBad {
			continue
		}

		if index == 0 && sample.Value.Kind() == metrics.KindUint64 {
			CountGoroutine.Store(sample.Value.Uint64())
		}

		if index == 1 && sample.Value.Kind() == metrics.KindUint64 {
			CountFreeMemory.Store(sample.Value.Uint64())
		}

		if index == 2 && sample.Value.Kind() == metrics.KindUint64 {
			CountReleasedMemory.Store(sample.Value.Uint64())
		}

		if index == 3 && sample.Value.Kind() == metrics.KindUint64 {
			CountStackMemory.Store(sample.Value.Uint64())
		}

		if index == 4 && sample.Value.Kind() == metrics.KindUint64 {
			CountUnusedMemory.Store(sample.Value.Uint64())
		}

		if index == 5 && sample.Value.Kind() == metrics.KindUint64 {
			CountTotalMemory.Store(sample.Value.Uint64())
		}

		if index == 6 && sample.Value.Kind() == metrics.KindUint64 {
			CountObjectsMemory.Store(sample.Value.Uint64())
		}
	}
}

func getTcpStatus() map[string]int {
	stateCount := make(map[string]int)
	cmd := exec.Command("ss", "-t", "-a")
	output, err := cmd.Output()
	if err != nil {
		fmt.Println("执行命令失败:", err)
		return stateCount
	}

	outputStr := string(output)

	lines := strings.Split(outputStr, "\n")

	for _, line := range lines {
		if strings.TrimSpace(line) == "" || strings.Contains(line, "State") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) >= 1 {
			state := fields[0]
			stateCount[state]++
		}
	}

	return stateCount

}

func collect(interval int64) {
	getRuntume()
	Data = fmt.Sprintf(
		`key: %d
uptime: %s
count connection: %d
count public tcp request qps: %d
count private http request qps: %d
count goroutine: %d
count free memory: %d
count released memory: %d
count stack memory: %d
count unused memory: %d
count total memory: %d
count objects memory: %d
count tcp status: %v
p99 public http request latency: %v
p99 private http request latency: %v`,
		time.Now().UnixMilli(),
		time.Since(StartTime).String(),
		CountConnection.Load(),
		CountPublicTCPRequest.Load()/interval,
		CountPrivateHTTPRequest.Load()/interval,
		CountGoroutine.Load(),
		CountFreeMemory.Load(),
		CountReleasedMemory.Load(),
		CountStackMemory.Load(),
		CountUnusedMemory.Load(),
		CountTotalMemory.Load(),
		CountObjectsMemory.Load(),
		lastTCPStatus,
		P99PublicHTTPRequestLatency.Out(),
		P99PrivateHTTPRequestLatency.Out())

	// 清理
	reset()
}

func Report(intervalSec int) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				utils.AlertAuto(fmt.Sprintf("report panic: %v stack: %s", r, string(debug.Stack())))
			}
		}()

		t0 := time.NewTicker(time.Duration(intervalSec) * time.Second)
		defer t0.Stop()

		t1 := time.NewTicker(60 * time.Second)
		defer t1.Stop()

		t2 := time.NewTicker(10 * time.Second)
		defer t2.Stop()

		for {
			select {
			case <-t0.C:
				// Report to Feishu
				utils.AlertAuto(Data)
			case <-t1.C:
				if version.ENV != version.EnvRelease {
					//lastTCPStatus = getTcpStatus()
				}
			case <-t2.C:
				// Collect Performance Metrics Every 10 Seconds
				collect(10)
			}
		}

	}()
}

func reset() {
	CountPublicTCPRequest.Store(0)
	CountPrivateHTTPRequest.Store(0)
	CountGoroutine.Store(0)
	CountFreeMemory.Store(0)
	CountReleasedMemory.Store(0)
	CountStackMemory.Store(0)
	CountUnusedMemory.Store(0)
	CountTotalMemory.Store(0)
	CountObjectsMemory.Store(0)
	P99PublicHTTPRequestLatency.Reset()
	P99PrivateHTTPRequestLatency.Reset()
}
