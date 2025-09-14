package progress

import (
	"context"
	"sync"
	"time"

	"github.com/a4abhishek/fileops/pkg/domain"
)

// Tracker manages progress reporting for operations
type Tracker struct {
	mu            sync.RWMutex
	operations    map[string]*OperationTracker
	subscribers   map[string][]chan domain.ProgressInfo
	subscribersMu sync.RWMutex
}

// NewTracker creates a new progress tracker
func NewTracker() *Tracker {
	return &Tracker{
		operations:  make(map[string]*OperationTracker),
		subscribers: make(map[string][]chan domain.ProgressInfo),
	}
}

// OperationTracker tracks progress for a single operation
type OperationTracker struct {
	mu              sync.RWMutex
	id              string
	operationType   domain.OperationType
	status          domain.OperationStatus
	startTime       time.Time
	endTime         *time.Time
	currentStep     string
	stepsCompleted  int
	totalSteps      int
	itemsProcessed  int64
	totalItems      int64
	bytesProcessed  int64
	totalBytes      int64
	lastUpdate      time.Time
	speedSamples    []speedSample
	maxSpeedSamples int
	details         map[string]interface{}
	errors          []string
	ctx             context.Context
	cancel          context.CancelFunc
	pauseChannel    chan bool
	resumeChannel   chan bool
	isPaused        bool
}

type speedSample struct {
	timestamp time.Time
	items     int64
	bytes     int64
}

// StartOperation creates and starts tracking a new operation
func (t *Tracker) StartOperation(id string, operationType domain.OperationType, totalSteps int) *OperationTracker {
	ctx, cancel := context.WithCancel(context.Background())

	tracker := &OperationTracker{
		id:              id,
		operationType:   operationType,
		status:          domain.StatusRunning,
		startTime:       time.Now(),
		currentStep:     "Initializing",
		stepsCompleted:  0,
		totalSteps:      totalSteps,
		lastUpdate:      time.Now(),
		speedSamples:    make([]speedSample, 0, 10),
		maxSpeedSamples: 10,
		details:         make(map[string]interface{}),
		ctx:             ctx,
		cancel:          cancel,
		pauseChannel:    make(chan bool, 1),
		resumeChannel:   make(chan bool, 1),
	}

	t.mu.Lock()
	t.operations[id] = tracker
	t.mu.Unlock()

	// Report initial progress
	t.reportProgress(tracker.GetProgressInfo())

	return tracker
}

// UpdateStep updates the current step of an operation
func (ot *OperationTracker) UpdateStep(step string) {
	ot.mu.Lock()
	defer ot.mu.Unlock()

	ot.currentStep = step
	ot.stepsCompleted++
	ot.lastUpdate = time.Now()
}

// UpdateProgress updates the progress counters
func (ot *OperationTracker) UpdateProgress(itemsProcessed, totalItems, bytesProcessed, totalBytes int64) {
	ot.mu.Lock()
	defer ot.mu.Unlock()

	now := time.Now()

	// Update counters
	ot.itemsProcessed = itemsProcessed
	ot.totalItems = totalItems
	ot.bytesProcessed = bytesProcessed
	ot.totalBytes = totalBytes

	// Add speed sample
	sample := speedSample{
		timestamp: now,
		items:     itemsProcessed,
		bytes:     bytesProcessed,
	}

	ot.speedSamples = append(ot.speedSamples, sample)
	if len(ot.speedSamples) > ot.maxSpeedSamples {
		ot.speedSamples = ot.speedSamples[1:]
	}

	ot.lastUpdate = now
}

// IncrementProgress increments the progress counters
func (ot *OperationTracker) IncrementProgress(items, bytes int64) {
	ot.mu.Lock()
	defer ot.mu.Unlock()

	ot.itemsProcessed += items
	ot.bytesProcessed += bytes
	ot.lastUpdate = time.Now()

	// Add speed sample
	sample := speedSample{
		timestamp: ot.lastUpdate,
		items:     ot.itemsProcessed,
		bytes:     ot.bytesProcessed,
	}

	ot.speedSamples = append(ot.speedSamples, sample)
	if len(ot.speedSamples) > ot.maxSpeedSamples {
		ot.speedSamples = ot.speedSamples[1:]
	}
}

// SetTotals updates the total counters
func (ot *OperationTracker) SetTotals(totalItems, totalBytes int64) {
	ot.mu.Lock()
	defer ot.mu.Unlock()

	ot.totalItems = totalItems
	ot.totalBytes = totalBytes
}

// SetDetail adds or updates a detail field
func (ot *OperationTracker) SetDetail(key string, value interface{}) {
	ot.mu.Lock()
	defer ot.mu.Unlock()

	ot.details[key] = value
}

// AddError adds an error to the operation
func (ot *OperationTracker) AddError(err string) {
	ot.mu.Lock()
	defer ot.mu.Unlock()

	ot.errors = append(ot.errors, err)
}

// Complete marks the operation as completed
func (ot *OperationTracker) Complete() {
	ot.mu.Lock()
	defer ot.mu.Unlock()

	ot.status = domain.StatusCompleted
	now := time.Now()
	ot.endTime = &now
}

// Fail marks the operation as failed
func (ot *OperationTracker) Fail(err string) {
	ot.mu.Lock()
	defer ot.mu.Unlock()

	ot.status = domain.StatusFailed
	now := time.Now()
	ot.endTime = &now
	ot.errors = append(ot.errors, err)
}

// Cancel cancels the operation
func (ot *OperationTracker) Cancel() {
	ot.mu.Lock()
	defer ot.mu.Unlock()

	ot.status = domain.StatusCancelled
	now := time.Now()
	ot.endTime = &now
	ot.cancel()
}

// Pause pauses the operation
func (ot *OperationTracker) Pause() {
	ot.mu.Lock()
	defer ot.mu.Unlock()

	if ot.status == domain.StatusRunning && !ot.isPaused {
		ot.status = domain.StatusPaused
		ot.isPaused = true
		select {
		case ot.pauseChannel <- true:
		default:
		}
	}
}

// Resume resumes the operation
func (ot *OperationTracker) Resume() {
	ot.mu.Lock()
	defer ot.mu.Unlock()

	if ot.status == domain.StatusPaused && ot.isPaused {
		ot.status = domain.StatusRunning
		ot.isPaused = false
		select {
		case ot.resumeChannel <- true:
		default:
		}
	}
}

// Context returns the operation context
func (ot *OperationTracker) Context() context.Context {
	return ot.ctx
}

// IsPaused returns whether the operation is paused
func (ot *OperationTracker) IsPaused() bool {
	ot.mu.RLock()
	defer ot.mu.RUnlock()
	return ot.isPaused
}

// WaitForResume waits for the operation to be resumed if paused
func (ot *OperationTracker) WaitForResume() {
	for ot.IsPaused() {
		select {
		case <-ot.resumeChannel:
			return
		case <-ot.ctx.Done():
			return
		case <-time.After(100 * time.Millisecond):
			// Check again
		}
	}
}

// calculateSpeed calculates the current processing speed
func (ot *OperationTracker) calculateSpeed() int64 {
	if len(ot.speedSamples) < 2 {
		return 0
	}

	latest := ot.speedSamples[len(ot.speedSamples)-1]
	oldest := ot.speedSamples[0]

	timeDiff := latest.timestamp.Sub(oldest.timestamp).Seconds()
	if timeDiff == 0 {
		return 0
	}

	itemsDiff := latest.items - oldest.items
	return int64(float64(itemsDiff) / timeDiff)
}

// calculateETA calculates the estimated time to completion
func (ot *OperationTracker) calculateETA() *time.Duration {
	if ot.totalItems == 0 || ot.itemsProcessed >= ot.totalItems {
		return nil
	}

	speed := ot.calculateSpeed()
	if speed == 0 {
		return nil
	}

	remaining := ot.totalItems - ot.itemsProcessed
	etaSeconds := float64(remaining) / float64(speed)
	eta := time.Duration(etaSeconds) * time.Second

	return &eta
}

// GetProgressInfo creates a ProgressInfo snapshot
func (ot *OperationTracker) GetProgressInfo() domain.ProgressInfo {
	ot.mu.RLock()
	defer ot.mu.RUnlock()

	progress := domain.ProgressInfo{
		ID:             ot.id,
		OperationType:  ot.operationType,
		Status:         ot.status,
		StartTime:      ot.startTime,
		EndTime:        ot.endTime,
		CurrentStep:    ot.currentStep,
		StepsCompleted: ot.stepsCompleted,
		TotalSteps:     ot.totalSteps,
		ItemsProcessed: ot.itemsProcessed,
		TotalItems:     ot.totalItems,
		BytesProcessed: ot.bytesProcessed,
		TotalBytes:     ot.totalBytes,
		Speed:          ot.calculateSpeed(),
		EstimatedETA:   ot.calculateETA(),
		Details:        make(map[string]interface{}),
	}

	// Copy details
	for k, v := range ot.details {
		progress.Details[k] = v
	}

	// Add errors if any
	if len(ot.errors) > 0 {
		progress.Error = ot.errors[len(ot.errors)-1] // Latest error
		progress.Details["error_count"] = len(ot.errors)
		progress.Details["all_errors"] = ot.errors
	}

	return progress
}

// GetOperation returns the operation tracker for the given ID
func (t *Tracker) GetOperation(id string) *OperationTracker {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.operations[id]
}

// GetProgress returns current progress for an operation
func (t *Tracker) GetProgress(id string) *domain.ProgressInfo {
	t.mu.RLock()
	tracker, exists := t.operations[id]
	t.mu.RUnlock()

	if !exists {
		return nil
	}

	progress := tracker.GetProgressInfo()
	return &progress
}

// GetAllProgress returns progress for all operations
func (t *Tracker) GetAllProgress() []domain.ProgressInfo {
	t.mu.RLock()
	defer t.mu.RUnlock()

	progress := make([]domain.ProgressInfo, 0, len(t.operations))
	for _, tracker := range t.operations {
		progress = append(progress, tracker.GetProgressInfo())
	}

	return progress
}

// ReportProgress implements ProgressReporter interface
func (t *Tracker) ReportProgress(progress domain.ProgressInfo) error {
	return t.reportProgress(progress)
}

// Subscribe implements ProgressReporter interface
func (t *Tracker) Subscribe(operationID string) (<-chan domain.ProgressInfo, error) {
	t.subscribersMu.Lock()
	defer t.subscribersMu.Unlock()

	ch := make(chan domain.ProgressInfo, 10) // Buffered channel
	t.subscribers[operationID] = append(t.subscribers[operationID], ch)

	return ch, nil
}

// Unsubscribe implements ProgressReporter interface
func (t *Tracker) Unsubscribe(operationID string) error {
	t.subscribersMu.Lock()
	defer t.subscribersMu.Unlock()

	delete(t.subscribers, operationID)
	return nil
}

// reportProgress sends progress updates to subscribers
func (t *Tracker) reportProgress(progress domain.ProgressInfo) error {
	t.subscribersMu.RLock()
	subscribers := t.subscribers[progress.ID]
	allSubscribers := t.subscribers["*"] // Global subscribers
	t.subscribersMu.RUnlock()

	// Send to operation-specific subscribers
	for _, ch := range subscribers {
		select {
		case ch <- progress:
		default:
			// Channel is full, skip this subscriber
		}
	}

	// Send to global subscribers
	for _, ch := range allSubscribers {
		select {
		case ch <- progress:
		default:
			// Channel is full, skip this subscriber
		}
	}

	return nil
}

// StartAutoReporting starts automatic progress reporting
func (t *Tracker) StartAutoReporting(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			t.mu.RLock()
			for _, tracker := range t.operations {
				if tracker.status == domain.StatusRunning {
					progress := tracker.GetProgressInfo()
					t.reportProgress(progress)
				}
			}
			t.mu.RUnlock()
		}
	}
}

// CleanupCompleted removes completed operations from tracking
func (t *Tracker) CleanupCompleted(maxAge time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)

	for id, tracker := range t.operations {
		tracker.mu.RLock()
		shouldRemove := tracker.status != domain.StatusRunning &&
			tracker.status != domain.StatusPaused &&
			tracker.endTime != nil &&
			tracker.endTime.Before(cutoff)
		tracker.mu.RUnlock()

		if shouldRemove {
			delete(t.operations, id)

			// Close subscriber channels
			t.subscribersMu.Lock()
			if subscribers, exists := t.subscribers[id]; exists {
				for _, ch := range subscribers {
					close(ch)
				}
				delete(t.subscribers, id)
			}
			t.subscribersMu.Unlock()
		}
	}
}

// Stats returns statistics about tracked operations
func (t *Tracker) Stats() map[string]interface{} {
	t.mu.RLock()
	defer t.mu.RUnlock()

	stats := map[string]interface{}{
		"total_operations": len(t.operations),
		"by_status":        make(map[domain.OperationStatus]int),
		"by_type":          make(map[domain.OperationType]int),
	}

	statusCounts := stats["by_status"].(map[domain.OperationStatus]int)
	typeCounts := stats["by_type"].(map[domain.OperationType]int)

	for _, tracker := range t.operations {
		tracker.mu.RLock()
		statusCounts[tracker.status]++
		typeCounts[tracker.operationType]++
		tracker.mu.RUnlock()
	}

	return stats
}
