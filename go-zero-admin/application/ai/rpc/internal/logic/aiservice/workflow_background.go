package aiservicelogic

import (
	"context"
	"sync"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/ai/rpc/internal/model"
	"go-zero-admin/application/ai/rpc/internal/svc"
)

// 执行权由 SQL 原子领取；进程内注册仅用于并发上限和及时取消，不承担持久队列职责。
var workflowSlots = make(chan struct{}, 8)
var workflowCancels sync.Map

type workflowJobKey struct {
	service *svc.ServiceContext
	id      int64
}

func reserveWorkflowSlot() bool {
	select {
	case workflowSlots <- struct{}{}:
		return true
	default:
		return false
	}
}
func releaseWorkflowSlot() { <-workflowSlots }

func launchWorkflow(ctx context.Context, service *svc.ServiceContext, task *model.TaskRow, approved bool) {
	jobCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 90*time.Second)
	key := workflowJobKey{service, task.Id}
	workflowCancels.Store(key, cancel)
	go func() {
		defer releaseWorkflowSlot()
		defer cancel()
		defer workflowCancels.Delete(key)
		defer func() {
			if recover() != nil {
				_, _ = failWorkflow(jobCtx, service, task, "执行过程异常，工具结果可能未知，请核对后另行运行")
			}
		}()
		// 跨实例取消也以事实源状态为准，观察到终态即取消当前工具上下文。
		done := make(chan struct{})
		defer close(done)
		go func() {
			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-done:
					return
				case <-jobCtx.Done():
					return
				case <-ticker.C:
					checkCtx, stop := context.WithTimeout(jobCtx, 2*time.Second)
					running, err := service.AiTaskModel.CheckRunningOwned(checkCtx, task.Id, task.CreatedBy)
					stop()
					if err != nil || !running {
						cancel()
						return
					}
				}
			}
		}()
		if _, err := executeWorkflow(jobCtx, service, task, approved); err != nil {
			logx.WithContext(jobCtx).Errorf("workflow task %d finished without confirmed state", task.Id)
			_, _ = failWorkflow(jobCtx, service, task, "任务执行结果未确认，请核对步骤记录")
		}
	}()
}

func cancelLocalWorkflow(service *svc.ServiceContext, id int64) {
	if value, ok := workflowCancels.Load(workflowJobKey{service, id}); ok {
		value.(context.CancelFunc)()
	}
}
