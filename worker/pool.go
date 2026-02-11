package worker

import (
    "dns_query_utility/config"
    "dns_query_utility/query"
    "dns_query_utility/result"
    "fmt"
    "sync"
)

type Pool struct {
    workerCount int
    jobs        chan query.QuerySpec
    results     chan result.QueryResult
    wg          sync.WaitGroup
    config      config.Config
    verbose     bool
}

func NewPool(workerCount int, cfg config.Config) *Pool {
    return &Pool{
        workerCount: workerCount,
        jobs:        make(chan query.QuerySpec, workerCount*2),
        results:     make(chan result.QueryResult, workerCount*2),
        config:      cfg,
        verbose:     false,
    }
}

func (p *Pool) Start() {
    for i := 1; i <= p.workerCount; i++ {
        p.wg.Add(1)
        go p.worker(i)
    }
}

func (p *Pool) worker(id int) {
    defer p.wg.Done()

    if p.verbose {
        fmt.Printf("[Worker %d] Started\n", id)
    }

    for spec := range p.jobs {
        if p.verbose {
            fmt.Printf("[Worker %d] Processing: %s (type=%s)\n", id, spec.Domain, spec.QueryType)
        }

        res := query.ExecuteQuery(spec, p.config)
        p.results <- res

        if p.verbose {
            fmt.Printf("[Worker %d] Completed: %s → %s\n", id, spec.Domain, res.Status)
        }
    }

    if p.verbose {
        fmt.Printf("[Worker %d] Finished\n", id)
    }
}

func (p *Pool) Submit(spec query.QuerySpec) {
    p.jobs <- spec
}

func (p *Pool) Close() {
    close(p.jobs)
}

func (p *Pool) Wait() {
    p.wg.Wait()
    close(p.results)
}

func (p *Pool) Results() <-chan result.QueryResult {
    return p.results
}

func (p *Pool) SetVerbose(verbose bool) {
    p.verbose = verbose
}

func Execute(specs []query.QuerySpec, cfg config.Config) []result.QueryResult {
    pool := NewPool(cfg.WorkerCount, cfg)

    pool.Start()

    go func() {
        for _, spec := range specs {
            pool.Submit(spec)
        }
        pool.Close()
    }()

    results := make([]result.QueryResult, 0, len(specs))
    for res := range pool.Results() {
        results = append(results, res)
    }

    pool.Wait()

    return results
}

func ExecuteWithProgress(specs []query.QuerySpec, cfg config.Config) []result.QueryResult {
    pool := NewPool(cfg.WorkerCount, cfg)
    totalJobs := len(specs)

    fmt.Printf("Starting %d workers to process %d queries...\n", cfg.WorkerCount, totalJobs)

    pool.Start()

    // Submit all jobs in a separate goroutine
    go func() {
        for _, spec := range specs {
            pool.Submit(spec)
        }
        pool.Close()
    }()

    // Close results channel after all workers finish
    go func() {
        pool.wg.Wait()
        close(pool.results)
    }()

    // Collect results with progress
    results := make([]result.QueryResult, 0, totalJobs)
    completed := 0

    for res := range pool.results {
        results = append(results, res)
        completed++

        percentage := float64(completed) / float64(totalJobs) * 100
        fmt.Printf("\rProgress: %d/%d (%.1f%%) - Last: %s → %s          ",
            completed, totalJobs, percentage, res.Domain, res.Status)
    }

    fmt.Println() // New line after progress

    return results
}