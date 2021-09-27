package missions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

var ErrBadData = errors.New("Bad data")

type MissionsProvider func() []*Mission

// returns if was callback succesful
type MissionClaimedCback func(context.Context, *Mission, string) bool

type MissionsServiceV1 struct {
	rdClient         *redis.Client
	missionsProvider MissionsProvider
	onMissionClaimed MissionClaimedCback
}

var missionsDB = []*Mission{
	&Mission{Counter: "event.tutorial_1", Reward: []*Reward{&Reward{Kind: KindCard}}, Goal: 5, NameKey: "Complete 5 tutorial steps", Id: "mission.bootcamp1"},
	&Mission{Counter: "event.new_user_signup", Reward: []*Reward{&Reward{Kind: KindOrb}}, Goal: 1, NameKey: "New Users Signup", Id: "mission.new_recruit"},
	&Mission{Counter: "event.battle_won", Reward: []*Reward{&Reward{Kind: KindBundle}}, Goal: 1, NameKey: "First battle won", Id: "mission.first_blood"},
	&Mission{Counter: "event.battle_won", Reward: []*Reward{&Reward{Kind: KindBundle}}, Goal: 1, NameKey: "Getting hands dirty", Id: "mission.5_battles_won"},
}

func ProvideMissions() []*Mission {
	return missionsDB
}

func NewMissionServiceV1(rd *redis.Client, provider MissionsProvider) *MissionsServiceV1 {

	if provider == nil {
		provider = ProvideMissions
	}

	return &MissionsServiceV1{
		rdClient:         rd,
		missionsProvider: provider,
	}
}

func (m *MissionsServiceV1) ListMissions(ctx context.Context) []*Mission {
	return m.missionsProvider()
}

func (m *MissionsServiceV1) WipeProgress(ctx context.Context, rq *WipeRequest) (*WipeRequestResponse, error) {

	keys, err := m.rdClient.Keys(ctx, fmt.Sprintf("missions:%s:*", rq.UserID)).Result()

	if err != nil && err != redis.Nil {
		logError(err)
		return nil, errors.New("Database error")
	}

	for _, k := range keys {
		m.rdClient.Del(ctx, k)
	}

	m.rdClient.Del(ctx, fmt.Sprintf("user:%v:missions", rq.UserID))

	return &WipeRequestResponse{NumRemoved: int64(len(keys))}, nil
}

func saveUserMission(ctx context.Context, rd *redis.Client, userID string, mission *MissionProgress) (err error) {

	serializedProgress, _ := json.Marshal(mission)
	if _, err = rd.Set(ctx, fmt.Sprintf("missions:%s:progress_by_id:%v", userID, mission.Id), serializedProgress, 0).Result(); err != nil {
		logError(err)
	}

	return
}

func (m *MissionsServiceV1) collectMission(ctx context.Context, missionID string) *Mission {
	for _, m := range m.ListMissions(ctx) {
		if m.Id == missionID {
			return m
		}
	}

	return nil
}

func collectUserMission(ctx context.Context, rd *redis.Client, userID, missionID string) *MissionProgress {
	var v = rd.Get(ctx, fmt.Sprintf("missions:%s:progress_by_id:%v", userID, missionID)).Val()

	var singleMission = &MissionProgress{}
	if err := json.Unmarshal([]byte(v), singleMission); err != nil {
		logError(err)
	}

	return singleMission
}

// Collect user progress for each started missions
func (m *MissionsServiceV1) MissionsUserProgress(ctx context.Context, rq *MissionProgressRQ) (result []*MissionProgress, err error) {

	progress, err := m.rdClient.SMembers(ctx,
		fmt.Sprintf("user:%s:missions", rq.UserID)).Result()
	if err == redis.Nil {
		return []*MissionProgress{}, nil
	}

	if err != nil {
		logError(err)
	}

	for _, v := range progress {
		if singleMission := collectUserMission(ctx, m.rdClient, rq.UserID, v); singleMission != nil {
			result = append(result, singleMission)
		}
	}

	return result, nil
}

// Reports if any counters been changed. Update user progress, returns any affected progress and rewards granted
func (m *MissionsServiceV1) ReportUserProgress(ctx context.Context, rq *ReportUserProgressRQ) *ReportUserProgressRS {

	missions := m.missionsProvider()

	var reportResult = &ReportUserProgressRS{}

	for _, mission := range missions {
		if mission.Counter == rq.Counter {
			// is this mission completed already for user?

			var userMission = &MissionProgress{}
			var userMissionKey = fmt.Sprintf("missions:%s:progress_by_id:%v", rq.UserID, mission.Id)

			missionJson, err := m.rdClient.Get(ctx, userMissionKey).
				Result()

			if err == nil {
				json.Unmarshal([]byte(missionJson), userMission)
			} else {
				if err != redis.Nil {
					logError(err)
				} else {
					userMission.Goal = mission.Goal
					userMission.Id = mission.Id
					userMission.Meta = mission.Meta
				}
			}

			if userMission.Current < userMission.Goal {
				userMission.Current++

				if userMission.Current >= userMission.Goal {
					userMission.UnlockedAt = time.Now().Unix()
				}

				if err = saveUserMission(ctx, m.rdClient, rq.UserID, userMission); err != nil {
					logError(err)
					continue
				}

				m.rdClient.SAdd(ctx, fmt.Sprintf("user:%s:missions", rq.UserID), mission.Id)

				reportResult.ModifiedProgress = append(reportResult.ModifiedProgress, userMission)
				if userMission.Current >= userMission.Goal {
					reportResult.Rewarded = append(reportResult.Rewarded, mission.Reward...)
				}
			}
		}
	}

	return reportResult
}
