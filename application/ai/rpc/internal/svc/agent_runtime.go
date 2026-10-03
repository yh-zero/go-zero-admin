package svc

import (
	"context"
	"errors"

	"go-zero-admin/pkg/agentjobs"
	"go-zero-admin/pkg/aiagent"
)

// StartAgentRuntime constructs shared model and task dependencies once at
// startup. The executor is injected by main so svc does not import business logic.
// Disabled mode neither contacts a provider nor requires the AI schema.
func (s *ServiceContext) StartAgentRuntime(execute agentjobs.Executor) error {
	if s == nil || s.DB == nil || s.DB.DB == nil {
		return errors.New("AI task database is not initialized")
	}
	if s.AgentJobs != nil {
		return errors.New("AI runtime is already initialized")
	}
	cfg, err := aiagent.LoadConfig(s.Config.AI)
	if err != nil {
		return err
	}
	s.AgentConfig = cfg
	if cfg.Enabled {
		s.AgentRunner, err = aiagent.New(context.Background(), cfg)
		if err != nil {
			return err
		}
	}
	s.AgentJobs, err = agentjobs.New(s.DB.DB, agentjobs.Config{
		Enabled: cfg.Enabled, RunTimeout: cfg.RunTimeout,
		MaxQuestionBytes: cfg.MaxInputChars * 4, MaxResultBytes: 64 * 1024,
	}, execute)
	if err != nil {
		return err
	}
	if cfg.Enabled {
		return s.AgentJobs.Start()
	}
	return nil
}
