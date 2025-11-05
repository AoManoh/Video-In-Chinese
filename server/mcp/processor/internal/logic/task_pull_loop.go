package logic

import (
	"context"
	"encoding/json"
	"time"

	"video-in-chinese/server/mcp/processor/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

// TaskMessage represents the message structure in the Redis queue.
type TaskMessage struct {
	TaskID           string `json:"task_id"`
	OriginalFilePath string `json:"original_file_path"`
}

// StartTaskPullLoop starts the task pulling loop.
//
// This function runs in a separate goroutine and continuously pulls tasks from Redis queue.
// It implements the following logic:
//  1. Periodically poll Redis queue (every 5 seconds)
//  2. Try to acquire worker slot (using Channel semaphore)
//  3. If max concurrency reached, skip this pull
//  4. If task pulled, start new Goroutine to process
//
// Parameters:
//   - ctx: context for graceful shutdown
//   - svcCtx: service context containing dependencies
//   - workerSem: channel semaphore for concurrency control
func StartTaskPullLoop(ctx context.Context, svcCtx *svc.ServiceContext, workerSem chan struct{}) {
	ticker := time.NewTicker(time.Duration(svcCtx.Config.TaskPullIntervalSeconds) * time.Second)
	defer ticker.Stop()

	logx.Infof("[TaskPullLoop] Started with interval %d seconds, max concurrency %d",
		svcCtx.Config.TaskPullIntervalSeconds, svcCtx.Config.MaxConcurrency)

	for {
		select {
		case <-ctx.Done():
			logx.Info("[TaskPullLoop] Received shutdown signal, exiting...")
			return

		case <-ticker.C:
			// Try to acquire worker slot (non-blocking)
			select {
			case workerSem <- struct{}{}:
				// Successfully acquired slot, pull task
				task, err := pullTask(ctx, svcCtx)
				if err != nil {
					logx.Errorf("[TaskPullLoop] Failed to pull task: %v", err)
					<-workerSem // Release slot
					continue
				}

				if task == nil {
					// No task available
					<-workerSem // Release slot
					continue
				}

				// Start processing in new goroutine
				go func(t *TaskMessage) {
					defer func() {
						<-workerSem // Release slot when done
					}()

					processTask(ctx, svcCtx, t)
				}(task)

			default:
				// Max concurrency reached, skip this pull
				logx.Infof("[TaskPullLoop] Max concurrency reached (%d), skipping pull",
					svcCtx.Config.MaxConcurrency)
			}
		}
	}
}

// pullTask pulls a task from Redis queue.
//
// Returns:
//   - *TaskMessage: task message if available, nil if queue is empty
//   - error: error if pull fails
func pullTask(ctx context.Context, svcCtx *svc.ServiceContext) (*TaskMessage, error) {
	// LPOP from task:pending queue (using PopTask method)
	taskJSON, err := svcCtx.RedisClient.PopTask(ctx, svcCtx.Config.TaskQueueKey)
	if err != nil {
		return nil, err
	}

	// Empty queue
	if taskJSON == "" {
		return nil, nil
	}

	// Parse task message
	var task TaskMessage
	if err := json.Unmarshal([]byte(taskJSON), &task); err != nil {
		logx.Errorf("[TaskPullLoop] Failed to parse task message: %v, raw: %s", err, taskJSON)
		return nil, err
	}

	logx.Infof("[TaskPullLoop] Pulled task: %s", task.TaskID)
	return &task, nil
}
