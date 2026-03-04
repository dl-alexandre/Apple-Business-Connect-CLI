// Package queue provides offline operation management for API instability.
// It queues operations locally when the API is unavailable and retries them
// when connectivity is restored.
package queue

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// OperationType represents the type of queued operation
type OperationType string

const (
	OpCreateLocation OperationType = "CREATE_LOCATION"
	OpUpdateLocation OperationType = "UPDATE_LOCATION"
	OpDeleteLocation OperationType = "DELETE_LOCATION"
	OpCreateShowcase OperationType = "CREATE_SHOWCASE"
	OpUpdateShowcase OperationType = "UPDATE_SHOWCASE"
	OpDeleteShowcase OperationType = "DELETE_SHOWCASE"
)

// OperationStatus represents the current status of a queued operation
type OperationStatus string

const (
	StatusPending    OperationStatus = "PENDING"
	StatusProcessing OperationStatus = "PROCESSING"
	StatusCompleted  OperationStatus = "COMPLETED"
	StatusFailed     OperationStatus = "FAILED"
	StatusCancelled  OperationStatus = "CANCELLED"
)

// QueuedOperation represents a single operation in the queue
type QueuedOperation struct {
	ID          string          `json:"id"`
	Type        OperationType   `json:"type"`
	Status      OperationStatus `json:"status"`
	EntityID    string          `json:"entity_id,omitempty"`
	EntityType  string          `json:"entity_type,omitempty"`
	Payload     json.RawMessage `json:"payload"`
	Error       string          `json:"error,omitempty"`
	Retries     int             `json:"retries"`
	MaxRetries  int             `json:"max_retries"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	ProcessedAt *time.Time      `json:"processed_at,omitempty"`
}

// Queue manages offline operations
type Queue struct {
	mu         sync.RWMutex
	dataDir    string
	queueFile  string
	operations []QueuedOperation
}

// NewQueue creates a new queue instance
func NewQueue(dataDir string) (*Queue, error) {
	if dataDir == "" {
		dataDir = filepath.Join(os.Getenv("HOME"), ".config", "abc", "queue")
	}

	// Ensure queue directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create queue directory: %w", err)
	}

	q := &Queue{
		dataDir:   dataDir,
		queueFile: filepath.Join(dataDir, "queue.json"),
	}

	// Load existing operations
	if err := q.load(); err != nil {
		return nil, fmt.Errorf("failed to load queue: %w", err)
	}

	return q, nil
}

// Add adds an operation to the queue
func (q *Queue) Add(opType OperationType, entityID, entityType string, payload interface{}) (*QueuedOperation, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	op := &QueuedOperation{
		ID:         generateID(),
		Type:       opType,
		Status:     StatusPending,
		EntityID:   entityID,
		EntityType: entityType,
		Payload:    payloadBytes,
		MaxRetries: 3,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	q.operations = append(q.operations, *op)

	if err := q.save(); err != nil {
		return nil, fmt.Errorf("failed to save queue: %w", err)
	}

	return op, nil
}

// GetPending returns all pending operations
func (q *Queue) GetPending() []QueuedOperation {
	q.mu.RLock()
	defer q.mu.RUnlock()

	var pending []QueuedOperation
	for _, op := range q.operations {
		if op.Status == StatusPending || op.Status == StatusFailed {
			if op.Retries < op.MaxRetries {
				pending = append(pending, op)
			}
		}
	}
	return pending
}

// Update updates an operation's status
func (q *Queue) Update(opID string, status OperationStatus, errorMsg string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	for i, op := range q.operations {
		if op.ID == opID {
			q.operations[i].Status = status
			q.operations[i].Error = errorMsg
			q.operations[i].UpdatedAt = time.Now()
			q.operations[i].Retries++

			if status == StatusProcessing {
				now := time.Now()
				q.operations[i].ProcessedAt = &now
			}

			return q.save()
		}
	}

	return fmt.Errorf("operation not found: %s", opID)
}

// Remove removes an operation from the queue
func (q *Queue) Remove(opID string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	for i, op := range q.operations {
		if op.ID == opID {
			q.operations = append(q.operations[:i], q.operations[i+1:]...)
			return q.save()
		}
	}

	return fmt.Errorf("operation not found: %s", opID)
}

// Clear removes all completed and cancelled operations
func (q *Queue) Clear() error {
	q.mu.Lock()
	defer q.mu.Unlock()

	var active []QueuedOperation
	for _, op := range q.operations {
		if op.Status != StatusCompleted && op.Status != StatusCancelled {
			active = append(active, op)
		}
	}

	q.operations = active
	return q.save()
}

// GetAll returns all operations in the queue
func (q *Queue) GetAll() []QueuedOperation {
	q.mu.RLock()
	defer q.mu.RUnlock()

	result := make([]QueuedOperation, len(q.operations))
	copy(result, q.operations)
	return result
}

// GetStats returns queue statistics
func (q *Queue) GetStats() QueueStats {
	q.mu.RLock()
	defer q.mu.RUnlock()

	stats := QueueStats{
		Total: len(q.operations),
	}

	for _, op := range q.operations {
		switch op.Status {
		case StatusPending:
			stats.Pending++
		case StatusProcessing:
			stats.Processing++
		case StatusCompleted:
			stats.Completed++
		case StatusFailed:
			stats.Failed++
			if op.Retries >= op.MaxRetries {
				stats.PermanentlyFailed++
			}
		case StatusCancelled:
			stats.Cancelled++
		}
	}

	return stats
}

// QueueStats holds statistics about the queue
type QueueStats struct {
	Total             int `json:"total"`
	Pending           int `json:"pending"`
	Processing        int `json:"processing"`
	Completed         int `json:"completed"`
	Failed            int `json:"failed"`
	PermanentlyFailed int `json:"permanently_failed"`
	Cancelled         int `json:"cancelled"`
}

// save persists the queue to disk
func (q *Queue) save() error {
	data, err := json.MarshalIndent(q.operations, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(q.queueFile, data, 0644)
}

// load reads the queue from disk
func (q *Queue) load() error {
	data, err := os.ReadFile(q.queueFile)
	if err != nil {
		if os.IsNotExist(err) {
			// No existing queue file, start fresh
			return nil
		}
		return err
	}

	return json.Unmarshal(data, &q.operations)
}

// generateID creates a unique operation ID
func generateID() string {
	return fmt.Sprintf("op_%d_%d", time.Now().Unix(), time.Now().Nanosecond())
}

// RetryPolicy defines retry behavior
type RetryPolicy struct {
	MaxRetries int
	BaseDelay  time.Duration
	MaxDelay   time.Duration
	Multiplier float64
}

// DefaultRetryPolicy returns a sensible default retry policy
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxRetries: 3,
		BaseDelay:  5 * time.Second,
		MaxDelay:   5 * time.Minute,
		Multiplier: 2.0,
	}
}

// CalculateDelay calculates the delay before next retry using exponential backoff
func (rp RetryPolicy) CalculateDelay(attempt int) time.Duration {
	delay := rp.BaseDelay
	for i := 0; i < attempt; i++ {
		delay = time.Duration(float64(delay) * rp.Multiplier)
		if delay > rp.MaxDelay {
			delay = rp.MaxDelay
			break
		}
	}
	return delay
}

// ShouldRetry determines if an operation should be retried
func (op *QueuedOperation) ShouldRetry(policy RetryPolicy) bool {
	return op.Status == StatusFailed && op.Retries < policy.MaxRetries
}
