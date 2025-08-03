package task

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// TaskStatus 任务状态
type TaskStatus string

const (
	StatusPending    TaskStatus = "pending"    // 等待中
	StatusProcessing TaskStatus = "processing" // 处理中
	StatusSuccess    TaskStatus = "success"    // 成功
	StatusFailed     TaskStatus = "failed"     // 失败
	StatusPartial    TaskStatus = "partial"    // 部分成功
)

// Task 任务信息
type Task struct {
	ID          string                 `json:"id"`          // 任务ID
	Status      TaskStatus             `json:"status"`      // 任务状态
	CreatedAt   time.Time              `json:"created_at"`  // 创建时间
	UpdatedAt   time.Time              `json:"updated_at"`  // 更新时间
	CompletedAt *time.Time             `json:"completed_at,omitempty"` // 完成时间
	Request     interface{}            `json:"request"`     // 原始请求
	Results     []PushResult           `json:"results"`     // 推送结果
	Error       string                 `json:"error,omitempty"` // 错误信息
	Progress    TaskProgress           `json:"progress"`    // 进度信息
}

// TaskProgress 任务进度
type TaskProgress struct {
	Total     int `json:"total"`     // 总数
	Success   int `json:"success"`   // 成功数
	Failed    int `json:"failed"`    // 失败数
	Pending   int `json:"pending"`   // 等待数
}

// PushResult 推送结果
type PushResult struct {
	Platform  string    `json:"platform"`  // 平台
	Webhook   string    `json:"webhook"`   // Webhook名称
	Status    string    `json:"status"`    // 状态
	Message   string    `json:"message"`   // 消息
	Timestamp time.Time `json:"timestamp"` // 时间戳
}

// TaskManager 任务管理器
type TaskManager struct {
	tasks       map[string]*Task
	mutex       sync.RWMutex
	cleanupTick *time.Ticker
	maxAge      time.Duration
}

var Manager *TaskManager

// InitTaskManager 初始化任务管理器
func InitTaskManager(cleanupInterval, maxAge int) {
	Manager = &TaskManager{
		tasks:       make(map[string]*Task),
		mutex:       sync.RWMutex{},
		cleanupTick: time.NewTicker(time.Duration(cleanupInterval) * time.Second),
		maxAge:      time.Duration(maxAge) * time.Second,
	}

	// 启动清理协程
	go Manager.cleanup()
}

// CreateTask 创建新任务
func (tm *TaskManager) CreateTask(request interface{}) *Task {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	task := &Task{
		ID:        uuid.New().String(),
		Status:    StatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Request:   request,
		Results:   make([]PushResult, 0),
		Progress:  TaskProgress{},
	}

	tm.tasks[task.ID] = task
	return task
}

// GetTask 获取任务
func (tm *TaskManager) GetTask(id string) (*Task, bool) {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	task, exists := tm.tasks[id]
	return task, exists
}

// UpdateTask 更新任务
func (tm *TaskManager) UpdateTask(id string, updater func(*Task)) {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	if task, exists := tm.tasks[id]; exists {
		updater(task)
		task.UpdatedAt = time.Now()
	}
}

// AddResult 添加推送结果
func (tm *TaskManager) AddResult(id string, result PushResult) {
	tm.UpdateTask(id, func(task *Task) {
		task.Results = append(task.Results, result)
		
		// 更新进度
		switch result.Status {
		case "success":
			task.Progress.Success++
		case "failed":
			task.Progress.Failed++
		}
		task.Progress.Pending = task.Progress.Total - task.Progress.Success - task.Progress.Failed

		// 检查是否完成
		if task.Progress.Pending == 0 {
			now := time.Now()
			task.CompletedAt = &now
			
			if task.Progress.Failed == 0 {
				task.Status = StatusSuccess
			} else if task.Progress.Success == 0 {
				task.Status = StatusFailed
			} else {
				task.Status = StatusPartial
			}
		}
	})
}

// SetTaskTotal 设置任务总数
func (tm *TaskManager) SetTaskTotal(id string, total int) {
	tm.UpdateTask(id, func(task *Task) {
		task.Progress.Total = total
		task.Progress.Pending = total
		task.Status = StatusProcessing
	})
}

// SetTaskError 设置任务错误
func (tm *TaskManager) SetTaskError(id string, err string) {
	tm.UpdateTask(id, func(task *Task) {
		task.Status = StatusFailed
		task.Error = err
		now := time.Now()
		task.CompletedAt = &now
	})
}

// cleanup 清理过期任务
func (tm *TaskManager) cleanup() {
	for range tm.cleanupTick.C {
		tm.mutex.Lock()
		now := time.Now()
		for id, task := range tm.tasks {
			if now.Sub(task.CreatedAt) > tm.maxAge {
				delete(tm.tasks, id)
			}
		}
		tm.mutex.Unlock()
	}
}

// Stop 停止任务管理器
func (tm *TaskManager) Stop() {
	if tm.cleanupTick != nil {
		tm.cleanupTick.Stop()
	}
}