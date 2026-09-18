package model

import (
	"time"

	"gorm.io/datatypes"
)

const (
	TrafficReportKindOneOff    = "one_off"
	TrafficReportKindPeriodic  = "periodic"
	TrafficReportStatusPending = "pending"
	TrafficReportStatusRunning = "running"
	TrafficReportStatusSuccess = "success"
	TrafficReportStatusFailed  = "failed"
)

type TrafficReportTask struct {
	ID             uint64         `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	OwnerUserID    uint64         `gorm:"column:owner_user_id;not null;index" json:"owner_user_id"`
	Name           string         `gorm:"column:name;size:200;not null" json:"name"`
	Kind           string         `gorm:"column:kind;size:20;not null" json:"kind"`
	Active         bool           `gorm:"column:active;not null;default:true" json:"active"`
	DataSourceType string         `gorm:"column:data_source_type;size:20;not null" json:"data_source_type"`
	ScheduleType   *string        `gorm:"column:schedule_type;size:20" json:"schedule_type,omitempty"`
	ScheduleExpr   *string        `gorm:"column:schedule_expr;size:100" json:"schedule_expr,omitempty"`
	Timezone       string         `gorm:"column:timezone;size:64;not null" json:"timezone"`
	WindowSelector string         `gorm:"column:window_selector;size:32;not null" json:"window_selector"`
	WindowParams   datatypes.JSON `gorm:"column:window_params;type:json" json:"window_params"`
	Params         datatypes.JSON `gorm:"column:params;type:json;not null" json:"params"`
	ExportFormats  datatypes.JSON `gorm:"column:export_formats;type:json;not null" json:"export_formats"`
	NextRunAt      *time.Time     `gorm:"column:next_run_at" json:"next_run_at,omitempty"`
	LastRunAt      *time.Time     `gorm:"column:last_run_at" json:"last_run_at,omitempty"`
	CreatedAt      time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (TrafficReportTask) TableName() string { return "traffic_report_tasks" }

type TrafficReportRun struct {
	ID             string                  `gorm:"column:id;primaryKey;size:36" json:"id"`
	TaskID         uint64                  `gorm:"column:task_id;not null;index" json:"task_id"`
	Status         string                  `gorm:"column:status;size:20;not null" json:"status"`
	ProgressPct    int                     `gorm:"column:progress_pct;not null;default:0" json:"progress_pct"`
	ProgressStage  *string                 `gorm:"column:progress_stage;size:200" json:"progress_stage,omitempty"`
	ResolvedWindow datatypes.JSON          `gorm:"column:resolved_window;type:json" json:"resolved_window"`
	ResolvedParams datatypes.JSON          `gorm:"column:resolved_params;type:json" json:"resolved_params"`
	EngineVersion  string                  `gorm:"column:engine_version;size:32;not null" json:"engine_version"`
	RowCount       int64                   `gorm:"column:row_count;not null;default:0" json:"row_count"`
	Summary        datatypes.JSON          `gorm:"column:summary;type:json" json:"summary,omitempty"`
	WarningMessage *string                 `gorm:"column:warning_message;type:text" json:"warning_message,omitempty"`
	ErrorMessage   *string                 `gorm:"column:error_message;type:text" json:"error_message,omitempty"`
	StartedAt      *time.Time              `gorm:"column:started_at" json:"started_at,omitempty"`
	FinishedAt     *time.Time              `gorm:"column:finished_at" json:"finished_at,omitempty"`
	CreatedAt      time.Time               `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	Artifacts      []TrafficReportArtifact `gorm:"foreignKey:RunID;references:ID" json:"artifacts,omitempty"`
}

func (TrafficReportRun) TableName() string { return "traffic_report_runs" }

type TrafficReportArtifact struct {
	ID          uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	RunID       string    `gorm:"column:run_id;size:36;not null;index" json:"run_id"`
	FileName    string    `gorm:"column:file_name;size:255;not null" json:"file_name"`
	MediaType   string    `gorm:"column:media_type;size:128;not null" json:"media_type"`
	StoragePath string    `gorm:"column:storage_path;size:500;not null" json:"-"`
	FileSize    int64     `gorm:"column:file_size;not null;default:0" json:"file_size"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (TrafficReportArtifact) TableName() string { return "traffic_report_artifacts" }
