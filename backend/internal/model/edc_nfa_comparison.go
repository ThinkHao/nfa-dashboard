package model

import "time"

// EDCNFAComparisonMemberInput is the write model used by the mapping editor.
// A nil validity boundary means the mapping is open ended.
type EDCNFAComparisonMemberInput struct {
	EntityID  uint64     `json:"entity_id"`
	ValidFrom *time.Time `json:"valid_from"`
	ValidTo   *time.Time `json:"valid_to"`
	Enabled   *bool      `json:"enabled"`
}

type EDCNFAComparisonGroupInput struct {
	GroupName    string                        `json:"group_name"`
	NFASrcRegion string                        `json:"nfa_src_region"`
	NFACP        string                        `json:"nfa_cp"`
	Enabled      *bool                         `json:"enabled"`
	Remark       string                        `json:"remark"`
	Members      []EDCNFAComparisonMemberInput `json:"members"`
}

// EDCNFAComparisonGroup is the business mapping used by the comparison view.
// NFASrcRegion/NFACP identify schools through nfa_school metadata.
type EDCNFAComparisonGroup struct {
	ID           uint64                        `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	GroupName    string                        `gorm:"column:group_name;not null" json:"group_name"`
	NFASrcRegion string                        `gorm:"column:nfa_src_region;not null" json:"nfa_src_region"`
	NFACP        string                        `gorm:"column:nfa_cp;not null" json:"nfa_cp"`
	Enabled      bool                          `gorm:"column:enabled;not null;default:true" json:"enabled"`
	Remark       string                        `gorm:"column:remark" json:"remark"`
	Members      []EDCNFAComparisonGroupMember `gorm:"foreignKey:GroupID" json:"members"`
	CreatedAt    time.Time                     `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time                     `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (EDCNFAComparisonGroup) TableName() string { return "edc_nfa_comparison_groups" }

type EDCNFAComparisonGroupMember struct {
	ID          uint64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	GroupID     uint64     `gorm:"column:group_id;not null" json:"group_id"`
	EntityID    uint64     `gorm:"column:entity_id;not null" json:"entity_id"`
	EDCName     string     `gorm:"column:edc_name" json:"edc_name"`
	DisplayName string     `gorm:"column:display_name" json:"display_name"`
	ValidFrom   *time.Time `gorm:"column:valid_from" json:"valid_from"`
	ValidTo     *time.Time `gorm:"column:valid_to" json:"valid_to"`
	Enabled     bool       `gorm:"column:enabled;not null;default:true" json:"enabled"`
}

func (EDCNFAComparisonGroupMember) TableName() string { return "edc_nfa_comparison_group_members" }

type EDCNFAComparisonFilter struct {
	GroupID           uint64
	StartTime         time.Time
	EndTime           time.Time
	AllowedEntityIDs  []uint64
	AllowedSchoolKeys []TrafficScopeSchoolKey
}

type EDCNFAComparisonPoint struct {
	Bucket5m        time.Time `json:"bucket_5m"`
	EDCServiceBytes int64     `json:"edc_service_bytes"`
	NFARecvBytes    int64     `json:"nfa_recv_bytes"`
	EDCMbps         float64   `json:"edc_mbps"`
	NFAMbps         float64   `json:"nfa_mbps"`
	DifferenceMbps  float64   `json:"difference_mbps"`
	Ratio           *float64  `json:"ratio"`
	EDCMemberCount  int       `json:"edc_member_count"`
	NFASchoolCount  int       `json:"nfa_school_count"`
	EDCRecordCount  int       `json:"edc_record_count"`
	NFARecordCount  int       `json:"nfa_record_count"`
	Status          string    `json:"status"`
}
