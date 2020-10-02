#!/bin/bash

go clean
go build benchmark.go

taskNum=100000000
#taskNum=100000
# relax模式
./benchmark -t ${taskNum} -r 10 
./benchmark -t ${taskNum} -r 100
./benchmark -t ${taskNum} -r 1000
./benchmark -t ${taskNum} -r 10000
./benchmark -t ${taskNum} -r 100000

# tired模式
./benchmark -t ${taskNum} -r 10 -m 
./benchmark -t ${taskNum} -r 100 -m 
./benchmark -t ${taskNum} -r 1000 -m 
./benchmark -t ${taskNum} -r 10000 -m 
./benchmark -t ${taskNum} -r 100000 -m 

go clean
