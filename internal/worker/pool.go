package worker

import (
	"context"
	"sync"

	"github.com/geraldbahati/image-sourcer/internal/cloudinary"
	"github.com/geraldbahati/image-sourcer/internal/csv"
)

// Job represents a single upload job
type Job struct {
	ProductImage csv.ProductImage
	Index        int // For tracking progress
}

// Result represents the result of a job
type Result struct {
	UploadResult cloudinary.UploadResult
	Job          Job
}

// Pool manages a pool of worker goroutines
type Pool struct {
	workerCount int
	uploader    *cloudinary.Uploader
	jobs        chan Job
	results     chan Result
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

// NewPool creates a new worker pool
func NewPool(workerCount int, uploader *cloudinary.Uploader) *Pool {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &Pool{
		workerCount: workerCount,
		uploader:    uploader,
		jobs:        make(chan Job, workerCount*2), // Buffered channel
		results:     make(chan Result, workerCount*2),
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Start starts the worker pool
func (p *Pool) Start() {
	// Start workers
	for i := 0; i < p.workerCount; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
}

// worker is a single worker goroutine
func (p *Pool) worker(id int) {
	defer p.wg.Done()

	for {
		select {
		case <-p.ctx.Done():
			// Context cancelled, stop worker
			return
		case job, ok := <-p.jobs:
			if !ok {
				// Jobs channel closed, stop worker
				return
			}

			// Process job with retry
			uploadResult := p.uploader.UploadWithRetry(p.ctx, job.ProductImage)

			// Send result
			result := Result{
				UploadResult: uploadResult,
				Job:          job,
			}

			select {
			case p.results <- result:
			case <-p.ctx.Done():
				return
			}
		}
	}
}

// Submit submits a job to the pool
func (p *Pool) Submit(job Job) {
	select {
	case p.jobs <- job:
	case <-p.ctx.Done():
		// Pool is shutting down
	}
}

// Results returns the results channel
func (p *Pool) Results() <-chan Result {
	return p.results
}

// Close closes the pool and waits for all workers to finish
func (p *Pool) Close() {
	close(p.jobs)    // Signal workers to stop after processing remaining jobs
	p.wg.Wait()      // Wait for all workers to finish
	close(p.results) // Close results channel
}

// Cancel cancels all pending jobs
func (p *Pool) Cancel() {
	p.cancel()       // Cancel context
	p.Close()        // Close pool
}

// Wait waits for all workers to finish
func (p *Pool) Wait() {
	p.wg.Wait()
}
