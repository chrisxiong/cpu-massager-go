cpu-massager-go
====
cpu-massager是一个过载保护器，称为"CPU按摩器"。

本仓库提供一个用go语言实现用来给CPU做马杀鸡的按摩器，利用该按摩器，可以设定一些参数，根据CPU的状态和所设参数，动态调整拒绝服务的概率，服务程序使用按摩器提供的相关API来决定对收到请求的应对方式：是照常处理还是拒绝服务，以此来让CPU的使用率维持在一个合适的水位，避免服务雪崩。

---

## 目录
* [使用方法](#使用方法)
  * [启动按摩计划](#启动按摩计划)
  * [判断是否拒绝服务](#判断是否拒绝服务)
* [工作原理](#工作原理)
  * [CPU使用率收集器](#CPU使用率收集器)
  * [CPU使用率记录器](#CPU使用率记录器)
  * [处理方式决断器](#处理方式决断器)
    * [判断CPU高低负荷](#判断CPU高低负荷)
    * [切换CPU状态](#切换CPU状态)
    * [疲累时拒绝服务](#疲累时拒绝服务)

## 使用方法
分两步：
1. 调用StartMassagePlan系列API启动按摩计划；
2. 调用NeedMassage这个API判断是否要拒绝服务。

### 启动按摩计划
在程序启动的时候调用StartMassagePlan系列API。例如，在Linux环境下：
```go
func main() {
    err := cpumassager.StartMassagePlanLinux()
    if err != nil {
        handleError() //  处理出错的情况，一般打印一下出错信息
        os.Exit(1) //  然后退出就好了
    }
    serve() //  进入服务程序正常处理流程
}
```
StartMassagePlanLinux是Linux环境下用一系列默认参数启动的建议API，有需要调整相关参数的可以使用StartMassagePlan这个API来设定相关参数启动按摩计划，具体参数的说明，可以参照代码中对于massagePlan结构的注释。

### 判断是否拒绝服务
程序启动，接收到请求，开始处理之前，先调用NeedMassage这个API来决定是正常处理该请求还是拒绝为其服务返回过载的错误信息。
```go
func handleARequest() {
    if cpumassager.NeedMassage() {
        refuse() //  拒绝服务该请求，做一些简单的处理，例如设定回包的错误码，上报过载告警等
        return  //  然后直接返回
    }
    process() //  正常处理该请求
}
```

## 工作原理
按摩器分为如下几个部分：
1. 提供给服务程序调用的API，具体可以参照"使用方法"部分的说明；
2. 按摩器内部的CPU使用率收集器、记录器和用于判断是否需要拒绝服务的决断器。

按摩器涉及模块的示意图：

![工作原理示意图](/diagrams/working_principle.png "工作原理示意图")

### CPU使用率收集器
服务程序启动按摩计划之后，按摩器就会启动一个routine来每隔1秒钟定期收集CPU使用率。收集器以一个接口的形式提供，不同的操作系统对于CPU使用率的取用方法可能会不一样，可以根据具体情况来提供具体的实现。

Linux平台上CPU使用率，读取procfs(进程文件系统)中的"/proc/stat"文件得到当下的CPU时间，取一个时间段前后的差值就可以得到。具体的可以参照htop的源码[LinuxProcessList_scanCPUTime](https://github.com/hishamhm/htop/blob/402e46bb82964366746b86d77eb5afa69c279539/linux/LinuxProcessList.c#L967)

CPU使用率收集器示意图：

![CPU使用率收集器](/diagrams/cpusage_collector.png "CPU使用率收集器")

### CPU使用率记录器
记录器用来记录收集器收集到的CPU使用率数据。记录器并不记录所有的CPU使用率数据，而是维护了一系列计数器，每个CPU使用率采集周期，都对每个计数器进行调整：
1. 如果CPU使用率数据>=当前计数器对应的CPU使用率，那么将该计数器加1，最高加到100；
2. 如果CPU使用率数据<当前计数器对应的CPU使用率，那么将该计数器减1，最低减至0。

由于每个计数器的范围是[0, 100]，这样就维护了最近100个采集周期（也就是最近100秒）的CPU使用率在不同水位的占比情况。例如，如果">=80计数器"的数值是75，那么就表示最近100次采集数据中，有75次CPU使用率不低于80%。维护这样的计数可以避免某[几]次的CPU使用率统计数据可能的误差，用一段时间内的集聚效果来确保获取到最近一段时间的CPU使用率真实水准。

CPU使用率记录器示意图：

![CPU使用率记录器](/diagrams/cpusage_recorder.png "CPU使用率记录器")

### 处理方式决断器
处理决断器，用来决定每个请求的处理方式：
1. 当CPU处在空闲的 ***轻松*** 状态时，每个请求都需要正常处理；
2. 当CPU处在繁忙的 ***疲累*** 状态时，需要根据设定参数以动态变化的概率拒绝为相关请求服务。

#### 判断CPU高低负荷

在具体判断CPU的"轻松"和"疲累"状态时，需要在每个CPU使用率采集时间点计算当下的CPU高低负荷，这个是根据启动按摩计划传入的两个参数和CPU记录器的计数器读数计算得到的：
1. tirenessLevel，疲累等级，这个是和CPU使用率记录器的计数器相匹配的，按摩器在判断CPU状态的时候，会根据该疲累等级获取对应的计数器读数；
2. tiredRatio，疲累比例，这个需要和疲累等级配合使用，如果CPU使用率记录器中对应疲累等级的计数器读数的占比高于疲累比例，则认为CPU处于 ***高负荷*** 中，否则处于 ***低负荷*** 中。

#### 切换CPU状态

CPU状态由轻松到疲累状态比较简单，CPU使用率采集器每个定期采集动作都会判断CPU是否高负荷，如果是在低负荷时检测到高负荷，直接将CPU状态由"轻松"转变成"疲累"即可。

由疲累到轻松的切换，相对复杂一些，先说明一个 ***intensity-按摩力度*** 的概念，按摩力度是指拒绝服务的概率，以百分比表示，取值范围是[0, 100]，例如，50表示以50%的概率拒绝服务。

还需要说明一个 ***CPU高/低负荷持续时长*** 的概念，这个会作为CPU疲累程度调整的依据，高/低负荷状态持续时长只有CPU处于疲累时才有意义，该时长是根据oldestTiredTime（最早判断疲累状态的时间）和latestTiredTime（最近判断为疲累状态的时间）和currentCPUsageRecordTime（当前CPU使用率采集时间）计算得到的：
1. 在疲累状态下检测到当前CPU处于高负荷，依据如下方式计算得到 ***CPU高负荷持续时长*** ：currentCPUsageRecordTime - oldestTiredTime；
2. 在疲累状态下检测到当前CPU处于低负荷，依据如下方式计算得到 ***CPU低负荷持续时长*** ：currentCPUsageRecordTime - latestTiredTime；
3. CPU每次由轻松转变到疲累或者调整按摩力度的时候都会重置oldestTiredTime和latestTiredTime为currentCPUsageRecordTime；
4. 每次检测到CPU处于高负荷时，会重置latestTiredTime为currentCPUsageRecordTime。

具体做疲累到轻松的切换时，会根据每个采样动作的判断结果，评估CPU高/低负荷持续时长，以此来对CPU的按摩力度做动态提高或降低，在合适的时机（CPU的按摩力度降为0的时候）才能进入轻松状态，调整按摩力度的时候，需要考虑启动按摩计划时传入的相关参数：
1. initialIntensity，初始化按摩力度，表示当CPU初次进入疲累状态时候拒绝服务的概率；
2. stepIntensity，按摩力度调整步进值，如果CPU进入疲累状态后持续处于高负荷/低负荷状态的时间超过配置的检查周期，会以此参数来增加/减少实际的按摩力度；
3. currentIntensity，实际的按摩力度；
4. checkPeriodInSeconds，调整按摩力度的检查周期，在CPU处在疲累状态下，CPU状态持续时长超过该值，则会调整按摩力度。

切换CPU状态示意图：

![切换CPU状态](/diagrams/change_cpu_state.png "切换CPU状态")

#### 疲累时拒绝服务

CPU处于疲累状态时，会根据如下几个方式来决定是否需要拒绝服务：
1. todoTasks，待处理任务数，疲累状态下每调用一次NeedMassage()就会加1；
2. doneTasks，已完成任务数，疲累状态下每次调用NeedMassage()返回false的情况下就会加1；
3. requireTasks，需要完成的任务数，用如下公式计算得到：todoTasks * (100 - currentIntensity) / 100；
4. 如果doneTasks < requireTasks，则需要提供服务，否则拒绝服务。

这种拒绝服务的方式，基于本地的信息作出决断，算法也非常简单，可以在不增加额外依赖的情况下，提供均匀的拒绝概率。配合前面的动态按摩力度，达到了在过载时候动态维护高服务水准的目的。
