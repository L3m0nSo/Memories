package task_scheduler

import "github.com/L3m0nSo/Memories/server/task"

type TaskScheduler interface {
	StartTasks()
	AddTask(task task.Task) error
	StopTasks()
}
