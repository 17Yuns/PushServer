package queue

import (
	"context"
	"fmt"
	"sync"

	"PushServer/internal/config"
	"PushServer/internal/logger"
	"PushServer/internal/model"
	"PushServer/internal/pusher"
	"PushServer/internal/task"
)

// PushJob 推送任务
type PushJob struct {
	TaskID  string             `json:"task_id"`
	Request model.PushRequest  `json:"request"`
}

// Queue 队列结构
type Queue struct {
	jobs        chan PushJob
	workers     int
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	pushService *pusher.PushService
}

var PushQueue *Queue

// InitQueue 初始化队列
func InitQueue() {
	ctx, cancel := context.WithCancel(context.Background())
	
	PushQueue = &Queue{
		jobs:        make(chan PushJob, config.AppConfig.Queue.BufferSize),
		workers:     config.AppConfig.Queue.WorkerCount,
		ctx:         ctx,
		cancel:      cancel,
		pushService: pusher.NewPushService(),
	}

	// 启动工作协程
	for i := 0; i < PushQueue.workers; i++ {
		PushQueue.wg.Add(1)
		go PushQueue.worker(i)
	}

	logger.Infof("队列系统初始化完成，工作协程数: %d，缓冲区大小: %d", 
		PushQueue.workers, config.AppConfig.Queue.BufferSize)
}

// AddJob 添加任务到队列
func (q *Queue) AddJob(job PushJob) error {
	select {
	case q.jobs <- job:
		logger.Debugf("任务已添加到队列: %s", job.TaskID)
		return nil
	case <-q.ctx.Done():
		return q.ctx.Err()
	default:
		logger.Warnf("队列已满，任务被拒绝: %s", job.TaskID)
		return ErrQueueFull
	}
}

// worker 工作协程
func (q *Queue) worker(id int) {
	defer q.wg.Done()
	
	logger.Infof("工作协程 %d 启动", id)
	
	for {
		select {
		case job := <-q.jobs:
			logger.Debugf("工作协程 %d 处理任务: %s", id, job.TaskID)
			q.processJob(job)
		case <-q.ctx.Done():
			logger.Infof("工作协程 %d 停止", id)
			return
		}
	}
}

// processJob 处理推送任务
func (q *Queue) processJob(job PushJob) {
	// 获取接收者配置
	recipient, exists := config.AppConfig.GetRecipient(job.Request.RecipientAlias)
	if !exists {
		task.Manager.SetTaskError(job.TaskID, "接收者不存在: "+job.Request.RecipientAlias)
		return
	}

	// 计算需要推送的总数
	totalPushes := q.calculateTotalPushes(recipient, job.Request)
	task.Manager.SetTaskTotal(job.TaskID, totalPushes)

	logger.Infof("开始处理推送任务: %s, 接收者: %s, 总推送数: %d", 
		job.TaskID, recipient.Name, totalPushes)

	// 使用推送服务执行策略
	q.pushService.ExecuteStrategy(job.TaskID, job.Request, recipient)
}

// calculateTotalPushes 计算总推送数
func (q *Queue) calculateTotalPushes(recipient config.RecipientConfig, req model.PushRequest) int {
	total := 0
	
	// 如果指定了平台，只计算该平台
	if req.Platform != "" {
		if platform, exists := recipient.Platforms[req.Platform]; exists && platform.Enabled {
			switch req.Strategy {
			case model.StrategyAll, model.StrategyMixed:
				total = len(platform.Webhooks)
			case model.StrategyFailover, model.StrategyWebhookAll, model.StrategyWebhookFailover:
				if len(platform.Webhooks) > 0 {
					total = 1
				}
			}
		}
		return total
	}

	// 计算所有启用平台的推送数
	for _, platform := range recipient.Platforms {
		if platform.Enabled {
			switch req.Strategy {
			case model.StrategyAll:
				total += len(platform.Webhooks)
			case model.StrategyFailover:
				if len(platform.Webhooks) > 0 {
					total = 1 // 故障转移只需要一个成功
					break
				}
			case model.StrategyWebhookAll, model.StrategyMixed:
				total += len(platform.Webhooks)
			case model.StrategyWebhookFailover:
				if len(platform.Webhooks) > 0 {
					total++
				}
			}
		}
	}
	
	return total
}



// Stop 停止队列
func (q *Queue) Stop() {
	logger.Info("正在停止队列系统...")
	q.cancel()
	close(q.jobs)
	q.wg.Wait()
	logger.Info("队列系统已停止")
}


// 错误定义
var (
	ErrQueueFull = fmt.Errorf("队列已满")
)
