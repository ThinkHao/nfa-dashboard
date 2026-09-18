package scheduler

import (
	"context"
	"log"
	"time"

	"nfa-dashboard/internal/service"
)

// TrafficReportScheduler only wakes the report service. Due-task claiming is
// atomic in the repository, so multiple dashboard instances remain safe.
type TrafficReportScheduler struct {
	service service.TrafficReportService
	stop    chan struct{}
	running bool
}

func NewTrafficReportScheduler(svc service.TrafficReportService) *TrafficReportScheduler {
	return &TrafficReportScheduler{service: svc, stop: make(chan struct{})}
}

func (s *TrafficReportScheduler) Start() {
	if s == nil || s.service == nil || s.running {
		return
	}
	s.running = true
	go func() {
		s.service.RunDueTasks(context.Background())
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.service.RunDueTasks(context.Background())
			case <-s.stop:
				return
			}
		}
	}()
	log.Println("流量报表调度器已启动")
}

func (s *TrafficReportScheduler) Stop() {
	if s == nil || !s.running {
		return
	}
	s.running = false
	close(s.stop)
}
