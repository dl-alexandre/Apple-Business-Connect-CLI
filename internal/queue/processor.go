// Package queue provides offline operation management
package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/dl-alexandre/abc/internal/api"
)

// Processor handles queue processing and retry logic
type Processor struct {
	queue    *Queue
	client   *api.Client
	policy   RetryPolicy
	running  bool
	stopChan chan struct{}
}

// NewProcessor creates a new queue processor
func NewProcessor(queue *Queue, client *api.Client, policy RetryPolicy) *Processor {
	return &Processor{
		queue:    queue,
		client:   client,
		policy:   policy,
		stopChan: make(chan struct{}),
	}
}

// ProcessOnce processes all pending operations once
func (p *Processor) ProcessOnce(ctx context.Context) error {
	pending := p.queue.GetPending()

	if len(pending) == 0 {
		return nil
	}

	fmt.Printf("Processing %d queued operations...\n", len(pending))

	for _, op := range pending {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := p.processOperation(ctx, op); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process operation %s: %v\n", op.ID, err)
		}
	}

	return nil
}

// processOperation handles a single operation
func (p *Processor) processOperation(ctx context.Context, op QueuedOperation) error {
	// Mark as processing
	if err := p.queue.Update(op.ID, StatusProcessing, ""); err != nil {
		return fmt.Errorf("failed to mark operation as processing: %w", err)
	}

	var err error

	// Execute the operation based on type
	switch op.Type {
	case OpCreateLocation:
		err = p.executeCreateLocation(ctx, op)
	case OpUpdateLocation:
		err = p.executeUpdateLocation(ctx, op)
	case OpDeleteLocation:
		err = p.executeDeleteLocation(ctx, op)
	case OpCreateShowcase:
		err = p.executeCreateShowcase(ctx, op)
	case OpUpdateShowcase:
		err = p.executeUpdateShowcase(ctx, op)
	case OpDeleteShowcase:
		err = p.executeDeleteShowcase(ctx, op)
	default:
		err = fmt.Errorf("unknown operation type: %s", op.Type)
	}

	if err != nil {
		// Mark as failed, will be retried later
		p.queue.Update(op.ID, StatusFailed, err.Error())
		return err
	}

	// Mark as completed
	return p.queue.Update(op.ID, StatusCompleted, "")
}

// executeCreateLocation creates a location via API
func (p *Processor) executeCreateLocation(ctx context.Context, op QueuedOperation) error {
	var location api.Location
	if err := json.Unmarshal(op.Payload, &location); err != nil {
		return fmt.Errorf("failed to unmarshal location payload: %w", err)
	}

	_, err := p.client.CreateLocation(ctx, &location)
	return err
}

// executeUpdateLocation updates a location via API
func (p *Processor) executeUpdateLocation(ctx context.Context, op QueuedOperation) error {
	var location api.Location
	if err := json.Unmarshal(op.Payload, &location); err != nil {
		return fmt.Errorf("failed to unmarshal location payload: %w", err)
	}

	_, err := p.client.UpdateLocation(ctx, op.EntityID, &location)
	return err
}

// executeDeleteLocation deletes a location via API
func (p *Processor) executeDeleteLocation(ctx context.Context, op QueuedOperation) error {
	return p.client.DeleteLocation(ctx, op.EntityID)
}

// executeCreateShowcase creates a showcase via API
func (p *Processor) executeCreateShowcase(ctx context.Context, op QueuedOperation) error {
	var showcase api.Showcase
	if err := json.Unmarshal(op.Payload, &showcase); err != nil {
		return fmt.Errorf("failed to unmarshal showcase payload: %w", err)
	}

	locationID := op.EntityID
	_, err := p.client.CreateShowcase(ctx, locationID, &showcase)
	return err
}

// executeUpdateShowcase updates a showcase via API
func (p *Processor) executeUpdateShowcase(ctx context.Context, op QueuedOperation) error {
	var showcase api.Showcase
	if err := json.Unmarshal(op.Payload, &showcase); err != nil {
		return fmt.Errorf("failed to unmarshal showcase payload: %w", err)
	}

	// EntityID format: "locationID/showcaseID"
	_, err := p.client.UpdateShowcase(ctx, op.EntityID, op.EntityID, &showcase)
	return err
}

// executeDeleteShowcase deletes a showcase via API
func (p *Processor) executeDeleteShowcase(ctx context.Context, op QueuedOperation) error {
	return p.client.DeleteShowcase(ctx, op.EntityID, op.EntityID)
}

// StartBackgroundProcessing starts automatic retry processing in background
func (p *Processor) StartBackgroundProcessing(ctx context.Context) {
	if p.running {
		return
	}

	p.running = true
	go p.backgroundLoop(ctx)
}

// Stop stops the background processing
func (p *Processor) Stop() {
	if !p.running {
		return
	}

	close(p.stopChan)
	p.running = false
}

// backgroundLoop continuously processes the queue
func (p *Processor) backgroundLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.stopChan:
			return
		case <-ticker.C:
			if err := p.ProcessOnce(ctx); err != nil {
				fmt.Fprintf(os.Stderr, "Background processing error: %v\n", err)
			}
		}
	}
}

// RetryFailed retries all failed operations
func (p *Processor) RetryFailed(ctx context.Context) error {
	all := p.queue.GetAll()

	retryCount := 0
	for _, op := range all {
		if op.ShouldRetry(p.policy) {
			// Wait before retry (exponential backoff)
			delay := p.policy.CalculateDelay(op.Retries)
			fmt.Printf("Waiting %v before retrying operation %s (attempt %d/%d)...\n",
				delay, op.ID, op.Retries+1, op.MaxRetries)
			time.Sleep(delay)

			// Reset status to pending
			if err := p.queue.Update(op.ID, StatusPending, ""); err != nil {
				return err
			}
			retryCount++
		}
	}

	if retryCount > 0 {
		fmt.Printf("Queued %d operations for retry\n", retryCount)
		return p.ProcessOnce(ctx)
	}

	fmt.Println("No failed operations to retry")
	return nil
}
