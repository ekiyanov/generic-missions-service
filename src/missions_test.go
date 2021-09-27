package missions

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ekiyanov/redisclient"
)

var missionsDBMock = []*Mission{
	&Mission{Counter: "event1", Reward: []*Reward{&Reward{Kind: KindCard}}, Goal: 5, NameKey: "Complete 5 tutorial steps", Id: "test-mission.bootcamp1"},
	&Mission{Counter: "event2", Reward: []*Reward{&Reward{Kind: KindOrb}}, Goal: 1, NameKey: "New Users Signup", Id: "test-mission.new_recruit"},
	&Mission{Counter: "event3", Reward: []*Reward{&Reward{Kind: KindBundle}}, Goal: 5, Id: "test-mission.event3-x5"},
	&Mission{Counter: "event3", Reward: []*Reward{&Reward{Kind: KindOrb}}, Goal: 3, Id: "test-mission.event3-x3"},
}

func ProvideMissionsMock() []*Mission {
	return missionsDBMock
}

func TestMissionProvider(t *testing.T) {
	provider := provideMissionsLocalJson()

	var missions = provider()
	if len(missions) == 0 {
		t.Fatal("should return list of missions")
	}

	// lets see if it would be reused
	missions = provider()
	if len(missions) == 0 {
		t.Fatal("cached missions was not returned")
	}
}

func TestMissions(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ctx = context.TODO()

	service := NewMissionServiceV1(
		redisclient.NewRedisClientCtx(ctx),
		ProvideMissionsMock,
	)

	var userId = fmt.Sprintf("user-%d", time.Now().Unix())

	missions := service.ListMissions(ctx)
	if len(missions) != 4 {
		t.Fatal("Missions count should be 4 for mocked data")
	}

	items, err := service.MissionsUserProgress(ctx, &MissionProgressRQ{UserID: userId})
	if err != nil {
		t.Fatal("Users Progress failed", err)
	}

	if len(items) != 0 {
		t.Fatal("Should return blank items")
	}

	var resp = service.ReportUserProgress(ctx, &ReportUserProgressRQ{UserID: userId, Counter: "event1"})

	if len(resp.ModifiedProgress) != 1 {
		t.Fatal("Should have 1 mission modified", resp.ModifiedProgress)
	}

	if len(resp.Rewarded) != 0 {
		t.Fatal("Should have nothing rewarded", resp.Rewarded)
	}

	if resp.ModifiedProgress[0].Current != 1 ||
		resp.ModifiedProgress[0].Goal != 5 {
		t.Fatal("Modified progress should have 1 of 5", resp.ModifiedProgress)
	}

	items, err = service.MissionsUserProgress(ctx, &MissionProgressRQ{UserID: userId})

	if len(items) != 1 {
		t.Fatal("should have 1 mission in progress", items)
	}

	// Complete 4 more event1

	service.ReportUserProgress(ctx, &ReportUserProgressRQ{UserID: userId, Counter: "event1"})
	service.ReportUserProgress(ctx, &ReportUserProgressRQ{UserID: userId, Counter: "event1"})
	service.ReportUserProgress(ctx, &ReportUserProgressRQ{UserID: userId, Counter: "event1"})
	resp = service.ReportUserProgress(ctx, &ReportUserProgressRQ{UserID: userId, Counter: "event1"})

	if len(resp.Rewarded) != 1 {
		t.Fatal("Should have reward  KindCard", resp.Rewarded)
	}

	if resp.ModifiedProgress[0].Current != 5 ||
		resp.ModifiedProgress[0].Goal != 5 {
		t.Fatal("Modified progress should have 5 of 5", resp.ModifiedProgress)
	}

	if resp.ModifiedProgress[0].ClaimedAt != 0 {
		t.Fatal("Completed missions should not be claimed automatically")
	}

	if time.Now().Unix()-resp.ModifiedProgress[0].UnlockedAt > 2 {
		t.Fatal("Unlock time is not correct. SHould not be older than  2 seconds before now ", resp.ModifiedProgress[0].UnlockedAt)
	}

	// Claim reward
	var missionRewarded = false
	var claimCallback = func(ctx context.Context, m *Mission, userId string) bool {
		missionRewarded = true
		return true
	}

	service.onMissionClaimed = claimCallback

	service.ClaimMission(ctx, &UserMission{UserID: userId, MissionID: resp.ModifiedProgress[0].Id})

	if missionRewarded != true {
		t.Fatal("Claim mission should call the reward")
	}

	missionRewarded = false
	_, err = service.ClaimMission(ctx, &UserMission{UserID: userId, MissionID: resp.ModifiedProgress[0].Id})

	if missionRewarded != false {
		t.Fatal("Repeated reward should not work")
	}

	if err == nil {
		t.Fatal("should return error")
	}

	// Complete event 3 to have 2 missions reported
	resp = service.ReportUserProgress(ctx, &ReportUserProgressRQ{UserID: userId, Counter: "event3"})

	if len(resp.ModifiedProgress) != 2 {
		t.Fatal("Should have reported 2 missions", resp.ModifiedProgress)
	}

	items, err = service.MissionsUserProgress(ctx, &MissionProgressRQ{UserID: userId})

	if len(items) != 3 {
		t.Fatal("Should have reported 3 missions. 1 completed and 2 with event3", items)
	}

	// Report something on another user
	userId2 := fmt.Sprintf("userid-2-%v", time.Now().Unix())
	service.ReportUserProgress(ctx, &ReportUserProgressRQ{UserID: userId2, Counter: "event3"})
	service.ReportUserProgress(ctx, &ReportUserProgressRQ{UserID: userId2, Counter: "event3"})
	service.ReportUserProgress(ctx, &ReportUserProgressRQ{UserID: userId2, Counter: "event3"})
	items, err = service.MissionsUserProgress(ctx, &MissionProgressRQ{UserID: userId2})
	if len(items) != 2 {
		t.Fatal("New user should have completed mission test-mission-event3x3 and event3x5 in progress", items)
	}

	// Wipe Progress
	wipeResp, err := service.WipeProgress(ctx, NewWipeRequest(userId))
	if wipeResp.NumRemoved == 0 {
		t.Fatal("Should have removed Items")
	}

	if err != nil {
		t.Fatal("Should not return error", err, "Wipe Request")
	}

	items, err = service.MissionsUserProgress(ctx, &MissionProgressRQ{UserID: userId})
	if len(items) != 0 {
		t.Fatal("Should return 0 after wipe progres", items)
	}

	items, err = service.MissionsUserProgress(ctx, &MissionProgressRQ{UserID: userId2})
	if len(items) != 2 {
		t.Fatal("Wipe progress should now affect user2")
	}

}
