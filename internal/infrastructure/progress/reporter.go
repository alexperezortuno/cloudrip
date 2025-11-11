package progress

import (
	"fmt"
	"sync"
	"time"
)

type Reporter struct {
	mu         sync.RWMutex
	total      int
	completed  int
	startTime  time.Time
	lastUpdate time.Time
	stopChan   chan bool
	isRunning  bool
}

func NewReporter() *Reporter {
	return &Reporter{
		stopChan: make(chan bool),
	}
}

func (p *Reporter) Start(total int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.total = total
	p.completed = 0
	p.startTime = time.Now()
	p.lastUpdate = time.Now()
	p.isRunning = true

	go p.reportLoop()
}

func (p *Reporter) Increment() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.completed++
	p.lastUpdate = time.Now()
}

func (p *Reporter) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.isRunning {
		p.isRunning = false
		p.stopChan <- true
		p.printProgress(true) // Print final progress
	}
}

func (p *Reporter) reportLoop() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.mu.Lock()
			if p.isRunning {
				p.printProgress(false)
			}
			p.mu.Unlock()
		case <-p.stopChan:
			return
		}
	}
}

func (p *Reporter) printProgress(final bool) {
	if p.total == 0 {
		return
	}

	percent := float64(p.completed) / float64(p.total) * 100
	elapsed := time.Since(p.startTime)
	rate := float64(p.completed) / elapsed.Seconds()

	status := "Progreso"
	if final {
		status = "Completado"
	}

	fmt.Printf("\r[%s] %d/%d (%.1f%%) - %.1f ops/sec - %v elapsed",
		status, p.completed, p.total, percent, rate, elapsed.Round(time.Second))

	if final {
		fmt.Println() // Nueva línea al final
	}
}

func (p *Reporter) GetProgress() float64 {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.total == 0 {
		return 0
	}
	return float64(p.completed) / float64(p.total) * 100
}
