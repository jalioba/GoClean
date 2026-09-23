package scanner

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"goclean/internal/model"
)

type dirTask struct {
	path string
	node *model.Node
}

type workQueue struct {
	mu       sync.Mutex
	cond     *sync.Cond
	tasks    []dirTask
	inFlight int64
	closed   bool
}

func newWorkQueue() *workQueue {
	q := &workQueue{
		tasks: make([]dirTask, 0, 1024),
	}
	q.cond = sync.NewCond(&q.mu)
	return q
}

func (q *workQueue) Push(task dirTask) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return
	}
	q.tasks = append(q.tasks, task)
	q.inFlight++
	q.cond.Signal()
}

func (q *workQueue) Pop() (dirTask, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	for len(q.tasks) == 0 && !q.closed {
		if q.inFlight == 0 {
			q.closed = true
			q.cond.Broadcast()
			return dirTask{}, false
		}
		q.cond.Wait()
	}

	if q.closed && len(q.tasks) == 0 {
		return dirTask{}, false
	}

	task := q.tasks[0]
	q.tasks = q.tasks[1:]
	return task, true
}

func (q *workQueue) TaskDone() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.inFlight--
	if q.inFlight == 0 && len(q.tasks) == 0 {
		q.closed = true
		q.cond.Broadcast()
	}
}

func (q *workQueue) Close() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.closed = true
	q.cond.Broadcast()
}

func (s *Scanner) Scan(ctx context.Context, rootPath string) (*model.Node, *model.ScanStats, error) {
	absRoot, err := filepath.Abs(rootPath)
	if err != nil {
		absRoot = rootPath
	}

	rootInfo, err := os.Stat(absRoot)
	if err != nil {
		return nil, nil, err
	}

	startTime := time.Now()
	rootNode := &model.Node{
		Name:    filepath.Base(absRoot),
		Path:    absRoot,
		Type:    model.NodeTypeDir,
		ModTime: rootInfo.ModTime(),
	}

	if !rootInfo.IsDir() {
		rootNode.Type = model.NodeTypeFile
		rootNode.Size = rootInfo.Size()
		rootNode.ItemCount = 1
		stats := &model.ScanStats{
			TotalFiles: 1,
			TotalBytes: rootNode.Size,
			Duration:   time.Since(startTime),
		}
		return rootNode, stats, nil
	}

	visited := &VisitedMap{}
	visited.MarkIfNew(absRoot)

	queue := newWorkQueue()
	var totalFiles atomic.Int64
	var totalDirs atomic.Int64
	var totalErrors atomic.Int64

	// IMPORTANT: Push the initial task BEFORE starting workers so that inFlight > 0
	queue.Push(dirTask{path: absRoot, node: rootNode})

	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Context watcher to unblock queue on cancellation
	go func() {
		<-ctx.Done()
		queue.Close()
	}()

	for i := 0; i < s.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}

				task, ok := queue.Pop()
				if !ok {
					return
				}

				s.processDir(ctx, task, queue, visited, &totalFiles, &totalDirs, &totalErrors)
				queue.TaskDone()
			}
		}()
	}

	wg.Wait()

	rootNode.PostOrderAggregate()

	stats := &model.ScanStats{
		TotalFiles:  totalFiles.Load(),
		TotalDirs:   totalDirs.Load(),
		TotalBytes:  rootNode.Size,
		ErrorsCount: totalErrors.Load(),
		Duration:    time.Since(startTime),
	}

	return rootNode, stats, nil
}

func (s *Scanner) processDir(
	ctx context.Context,
	task dirTask,
	queue *workQueue,
	visited *VisitedMap,
	totalFiles *atomic.Int64,
	totalDirs *atomic.Int64,
	totalErrors *atomic.Int64,
) {
	entries, err := os.ReadDir(task.path)
	if err != nil {
		totalErrors.Add(1)
		return
	}

	totalDirs.Add(1)

	for _, entry := range entries {
		select {
		case <-ctx.Done():
			return
		default:
		}

		childPath := filepath.Join(task.path, entry.Name())

		if IsSymlink(entry) {
			continue
		}

		if entry.IsDir() {
			if !visited.MarkIfNew(childPath) {
				continue
			}

			childNode := &model.Node{
				Name:   entry.Name(),
				Path:   childPath,
				Type:   model.NodeTypeDir,
				Parent: task.node,
			}
			task.node.AddChild(childNode)

			queue.Push(dirTask{path: childPath, node: childNode})
		} else {
			info, err := entry.Info()
			var size int64
			var modTime time.Time
			if err != nil {
				totalErrors.Add(1)
			} else {
				size = info.Size()
				modTime = info.ModTime()
			}

			childNode := &model.Node{
				Name:      entry.Name(),
				Path:      childPath,
				Size:      size,
				Type:      model.NodeTypeFile,
				ModTime:   modTime,
				ItemCount: 1,
				Parent:    task.node,
			}
			task.node.AddChild(childNode)
			totalFiles.Add(1)
		}
	}
}
