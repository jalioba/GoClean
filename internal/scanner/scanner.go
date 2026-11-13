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

// workQueue is a ring-buffer task queue with batch push support to minimize
// lock contention and avoid backing-array memory leaks.
type workQueue struct {
	mu       sync.Mutex
	cond     *sync.Cond
	tasks    []dirTask
	head     int
	tail     int
	count    int
	inFlight int64
	closed   bool
}

func newWorkQueue() *workQueue {
	q := &workQueue{
		tasks: make([]dirTask, 1024),
	}
	q.cond = sync.NewCond(&q.mu)
	return q
}

func (q *workQueue) growLocked(needed int) {
	newCap := len(q.tasks) * 2
	for newCap < q.count+needed {
		newCap *= 2
	}
	newTasks := make([]dirTask, newCap)
	if q.head < q.tail {
		copy(newTasks, q.tasks[q.head:q.tail])
	} else if q.count > 0 {
		n := copy(newTasks, q.tasks[q.head:])
		copy(newTasks[n:], q.tasks[:q.tail])
	}
	q.tasks = newTasks
	q.head = 0
	q.tail = q.count
}

func (q *workQueue) Push(task dirTask) {
	q.mu.Lock()
	if q.closed {
		q.mu.Unlock()
		return
	}
	if q.count == len(q.tasks) {
		q.growLocked(1)
	}
	q.tasks[q.tail] = task
	q.tail = (q.tail + 1) % len(q.tasks)
	q.count++
	q.inFlight++
	q.cond.Signal()
	q.mu.Unlock()
}

func (q *workQueue) PushBatch(batch []dirTask) {
	if len(batch) == 0 {
		return
	}
	q.mu.Lock()
	if q.closed {
		q.mu.Unlock()
		return
	}
	if q.count+len(batch) > len(q.tasks) {
		q.growLocked(len(batch))
	}
	for _, task := range batch {
		q.tasks[q.tail] = task
		q.tail = (q.tail + 1) % len(q.tasks)
	}
	q.count += len(batch)
	q.inFlight += int64(len(batch))
	if len(batch) == 1 {
		q.cond.Signal()
	} else {
		q.cond.Broadcast()
	}
	q.mu.Unlock()
}

func (q *workQueue) Pop() (dirTask, bool) {
	q.mu.Lock()
	for q.count == 0 && !q.closed {
		if q.inFlight == 0 {
			q.closed = true
			q.cond.Broadcast()
			q.mu.Unlock()
			return dirTask{}, false
		}
		q.cond.Wait()
	}

	if q.closed && q.count == 0 {
		q.mu.Unlock()
		return dirTask{}, false
	}

	task := q.tasks[q.head]
	q.tasks[q.head] = dirTask{} // clear pointer for GC
	q.head = (q.head + 1) % len(q.tasks)
	q.count--
	q.mu.Unlock()
	return task, true
}

func (q *workQueue) TaskDone() {
	q.mu.Lock()
	q.inFlight--
	if q.inFlight == 0 && q.count == 0 {
		q.closed = true
		q.cond.Broadcast()
	}
	q.mu.Unlock()
}

func (q *workQueue) Close() {
	q.mu.Lock()
	q.closed = true
	q.cond.Broadcast()
	q.mu.Unlock()
}

func joinPath(dir, name string) string {
	if len(dir) == 0 {
		return name
	}
	last := dir[len(dir)-1]
	if last == os.PathSeparator || last == '/' {
		return dir + name
	}
	return dir + string(os.PathSeparator) + name
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
		Name: filepath.Base(absRoot),
		Path: absRoot,
		Type: model.NodeTypeDir,
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
	visited.MarkRoot(rootInfo)

	queue := newWorkQueue()
	var totalFiles atomic.Int64
	var totalDirs atomic.Int64
	var totalErrors atomic.Int64

	// Push the initial task BEFORE starting workers so that inFlight > 0.
	queue.Push(dirTask{path: absRoot, node: rootNode})

	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Context watcher to unblock queue on cancellation.
	go func() {
		<-ctx.Done()
		queue.Close()
	}()

	var stopProgress chan struct{}
	var progressDone chan struct{}
	if s.onProgress != nil {
		stopProgress = make(chan struct{})
		progressDone = make(chan struct{})
		go func() {
			defer close(progressDone)
			ticker := time.NewTicker(80 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-stopProgress:
					return
				case <-ctx.Done():
					return
				case <-ticker.C:
					s.onProgress(totalDirs.Load(), totalFiles.Load(), totalErrors.Load())
				}
			}
		}()
	}

	for i := 0; i < s.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var subDirs []dirTask
			for {
				if ctx.Err() != nil {
					return
				}

				task, ok := queue.Pop()
				if !ok {
					return
				}

				subDirs = subDirs[:0]
				subDirs = s.processDir(ctx, task, subDirs, visited, &totalFiles, &totalDirs, &totalErrors)
				if len(subDirs) > 0 {
					queue.PushBatch(subDirs)
				}
				queue.TaskDone()
			}
		}()
	}

	wg.Wait()
	if stopProgress != nil {
		close(stopProgress)
		<-progressDone
	}

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
	subDirs []dirTask,
	visited *VisitedMap,
	totalFiles *atomic.Int64,
	totalDirs *atomic.Int64,
	totalErrors *atomic.Int64,
) []dirTask {
	if ctx.Err() != nil {
		return subDirs
	}

	entries, err := os.ReadDir(task.path)
	if err != nil {
		totalErrors.Add(1)
		return subDirs
	}

	totalDirs.Add(1)

	var localFiles int64
	var localErrors int64
	children := make([]*model.Node, 0, len(entries))

	for _, entry := range entries {
		if IsSymlink(entry) {
			continue
		}

		name := entry.Name()
		if entry.IsDir() {
			if !visited.MarkDirEntry(entry) {
				continue
			}

			childNode := &model.Node{
				Name:   name,
				Type:   model.NodeTypeDir,
				Parent: task.node,
			}
			children = append(children, childNode)
			subDirs = append(subDirs, dirTask{
				path: joinPath(task.path, name),
				node: childNode,
			})
		} else {
			info, err := entry.Info()
			var size int64
			if err != nil {
				localErrors++
			} else {
				size = info.Size()
			}

			children = append(children, &model.Node{
				Name:      name,
				Size:      size,
				Type:      model.NodeTypeFile,
				ItemCount: 1,
				Parent:    task.node,
			})
			localFiles++
		}
	}

	task.node.Children = children
	if localFiles > 0 {
		totalFiles.Add(localFiles)
	}
	if localErrors > 0 {
		totalErrors.Add(localErrors)
	}
	return subDirs
}
