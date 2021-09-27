package missions

import "context"

const KindCard = "card"
const KindBundle = "bundle"
const KindOrb = "orb"

// swagger:model Reward
type Reward struct {
	// what kind of item should be rewarded
	Kind string `json:"kind"`

	// How many of items should be rewarded
	Amount int64 `json:"amount"`
}

// swagger:model Mission
type Mission struct {
	// Unique Identifier of the mission
	Counter string `json:"counter"`

	// Reward item which should be granted upon completion
	Reward []*Reward `json:"reward"`

	// Goal of completion
	Goal int64 `json:"goal"`

	// Mission name key. Expected to be used to be localized in human readable name
	NameKey string `json:"name"`

	// Mission Id
	Id string `json:"id"`

	// Tags
	Tags []string `json:"tags"`

	// Client side related Meta
	Meta map[string]interface{} `json:"meta"`
}

// swagger:model ListMissionsRS
type ListMissionsRS []*Mission

// swagger:model MissionProgress
type MissionProgress struct {
	// Mission Id
	Id string `json:"id"`

	// Total amount should be reached
	Goal int64 `json:"goal"`

	// How many counters completed so far. Should be capped by Goal
	Current int64 `json:"current"`

	// When Current become equal or larger than goal ?
	UnlockedAt int64 `json:"unlocked_at"`

	// When mission been claimed. Epoch time. 0 means it was not claimed yet
	ClaimedAt int64 `json:"claimed_at"`

	// Meta from mission
	Meta map[string]interface{} `json:"meta"`
}

// swagger:model UserMissionProgressRS
type UserMissionProgressRS []*MissionProgress

// swagger:model MissionProgressRQ
type MissionProgressRQ struct {
	UserID string `json:"user_id"`
}

// swagger:model ReportUserProgressRQ
type ReportUserProgressRQ struct {
	UserID  string `json:"user_id"`
	Counter string `json:"counter"`
}

// swagger:model UserMission
type UserMission struct {
	// UserId for  whom to look for a mission
	UserID string `json:"user_id"`

	// MissionId which is should be claimed. Mission Current should be equal to the Goal
	MissionID string `json:"mission_id"`
}

// swagger:model ReportUserProgressRS
type ReportUserProgressRS struct {
	ModifiedProgress []*MissionProgress `json:"progress"`
	Rewarded         []*Reward          `json:"reward"`
}

type MissionsService interface {
	ListMissions(context.Context) []*Mission

	// Collect user progress for each started missions
	MissionsUserProgress(ctx context.Context, rq *MissionProgressRQ) ([]*MissionProgress, error)

	// Reports if any counters been changed. Update user progress, returns any affected progress and rewards granted
	ReportUserProgress(ctx context.Context, rq *ReportUserProgressRQ) *ReportUserProgressRS

	// Claim Reward
	ClaimMission(ctx context.Context, rq *UserMission) (*MissionProgress, error)

	// Wipe user's Progress
	WipeProgress(ctx context.Context, rq *WipeRequest) (*WipeRequestResponse, error)
}
