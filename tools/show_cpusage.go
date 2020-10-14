package main

import (
	"fmt"
	"os"
	"sync"
	"time"

	cpumassager "github.com/chrisxiong/cpu-massager-go"
)

func main() {
	linuxCollector, err := cpumassager.NewLinuxCPUsageCollector()
	if err != nil {
		fmt.Printf("NewLinuxCPUsageCollector error:%s\n", err.Error())
		os.Exit(1)
	}
	dockerCollector, err := cpumassager.NewDockerCPUsageCollector()
	if err != nil {
		fmt.Printf("NewDockerCPUsageCollector error:%s\n", err.Error())
		os.Exit(1)
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			time.Sleep(time.Duration(time.Second * 1))
			usageLinux := linuxCollector.GetCPUsage()
			usageDocker := dockerCollector.GetCPUsage()
			fmt.Printf("usage from linuxCollector:%f, usage from dockerCollector:%f\n", usageLinux, usageDocker)
		}
	}()
	wg.Wait()
	os.Exit(0)
}
