package main

import (
	
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

// 全局变量，用于跟踪由本服务器启动的 pprof 进程
var (
	runningPprofs = make(map[int]*os.Process) // 存储 PID 到 Process 指针的映射
	pprofMutex    sync.Mutex                  // 用于保护 runningPprofs 的互斥锁
)

// setupSignalHandler 设置信号处理，用于在服务器退出时清理 pprof 进程。
// 这个函数应该在 main 函数中被调用一次。
func setupSignalHandler() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs
		log.Printf("Received signal: %s. Cleaning up running pprof processes...", sig)

		pprofMutex.Lock()
		pidsToTerminate := make([]int, 0, len(runningPprofs))
		processesToTerminate := make([]*os.Process, 0, len(runningPprofs))
		for pid, process := range runningPprofs {
			pidsToTerminate = append(pidsToTerminate, pid)
			processesToTerminate = append(processesToTerminate, process)
		}
		runningPprofs = make(map[int]*os.Process) // 清空 map
		pprofMutex.Unlock()

		if len(pidsToTerminate) == 0 {
			log.Println("No running pprof processes to terminate.")
			return
		}

		log.Printf("Terminating %d pprof processes: %v", len(pidsToTerminate), pidsToTerminate)
		var wg sync.WaitGroup
		wg.Add(len(processesToTerminate))

		for i, process := range processesToTerminate {
			go func(p *os.Process, pid int) {
				defer wg.Done()
				log.Printf("Sending Interrupt signal to PID %d...", pid)
				err := p.Signal(os.Interrupt)
				if err != nil {
					log.Printf("Failed to send Interrupt to PID %d: %v. Trying Kill.", pid, err)
					err = p.Signal(os.Kill)
					if err != nil {
						log.Printf("Failed to send Kill to PID %d: %v", pid, err)
					}
				}
			}(process, pidsToTerminate[i])
		}
		wg.Wait() // 等待所有终止 goroutine 完成尝试
		log.Println("Cleanup finished.")
	}()
}
