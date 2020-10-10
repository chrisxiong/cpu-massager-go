package main

import (
	"flag"
	"fmt"
	"os"
	"sync"
	"time"

	cpumassager "github.com/chrisxiong/cpu-massager-go"
)

func doBenchMark(totalTaskNum int, routineNum int, tired bool) {
	var mode string
	if !tired {
		mode = "relax"
		err := cpumassager.StartMassagePlan()
		if err != nil {
			fmt.Printf("StartMassagePlan error:%s", err.Error())
			os.Exit(1)
		}
		if cpumassager.NeedMassage() {
			fmt.Println("should not need massage...")
			os.Exit(1)
		}
	} else {
		mode = "tired"
		const defaultHighLoadLevel = cpumassager.CounterTypeZero
		const defaultHighLoadRatio = 0.01
		const defaultCheckPeriodInSeconds = 1
		err := cpumassager.StartMassagePlan(cpumassager.WithHighLoadLevel(defaultHighLoadLevel),
			cpumassager.WithHighLoadRatio(defaultHighLoadRatio),
			cpumassager.WithCheckPeriodInseconds(defaultCheckPeriodInSeconds))
		if err != nil {
			fmt.Printf("StartMassagePlan error:%s\n", err.Error())
			os.Exit(1)
		}
		time.Sleep(2*time.Second + 100*time.Microsecond)
		if !cpumassager.NeedMassage() {
			fmt.Println("should need massage...")
			os.Exit(1)
		}
	}
	startTime := time.Now()
	var wg sync.WaitGroup
	for i := 0; i < routineNum; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for task := 0; task < totalTaskNum/routineNum; task++ {
				cpumassager.NeedMassage()
			}
		}()
	}
	wg.Wait()
	endTime := time.Now()
	period := endTime.Sub(startTime)
	fmt.Printf("mode:%s tasknum:%d routinenum:%-8d costtime(ms):%-8d qps:%.2f/s tpq:%dns\n",
		mode,
		totalTaskNum,
		routineNum,
		period.Milliseconds(),
		float64(totalTaskNum)/period.Seconds(),
		period.Nanoseconds()/int64(totalTaskNum))
}

var (
	help         bool
	totalTaskNum int
	routineNum   int
	tiredMode    bool
)

func init() {
	flag.BoolVar(&help, "h", false, "显示本帮助信息")
	flag.IntVar(&totalTaskNum, "t", 100000000, "任务数量，默认为100000000（1亿）")
	flag.IntVar(&routineNum, "r", 10, "协程数量，默认为10个")
	flag.BoolVar(&tiredMode, "m", false, "工作模式，默认值为false-relax模式，true-tired模式")
}

func main() {
	flag.Parse()
	if help {
		flag.Usage()
		os.Exit(0)
	}
	doBenchMark(totalTaskNum, routineNum, tiredMode)
}
