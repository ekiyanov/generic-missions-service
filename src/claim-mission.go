package missions

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"net/http"
)

var httpMu sync.Mutex
var _client *http.Client

func httpClient() *http.Client {

	httpMu.Lock()
	defer httpMu.Unlock()

	if _client == nil {

		_client = &http.Client{
			Timeout: time.Second * 10,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 100,
				MaxConnsPerHost:     100,
			},
		}
	}

	return _client
}

// TODO: Should be a job on job manager/redis?
func ClaimCallback(ctx context.Context, mission *Mission, userId string) bool {

	var err error
	if mission.Reward[0].Amount == 0 {
		log.Println("Ignore claim reward with zero amounts", mission.Reward)
		return true
	}

	rewardUrl := os.Getenv("REWARD_HOOK_URL")
	if rewardUrl == "" {
		log.Println("REWARD_HOOK_URL is missing")
		return false
	}

	reader := bytes.NewReader([]byte(
		fmt.Sprintf(`{"userid":"%v", "reward":"%v", "amount": %v}`, userId, mission.Reward[0].Kind, mission.Reward[0].Amount),
	))


	rq, _ := http.NewRequest(http.MethodPost, rewardUrl, reader)

	rq.Header.Set("X-API-KEY", os.Getenv("SPHERE_API_KEY"))
	rq.Header.Set("Content-Type", "application/json")

	_, err = httpClient().Do(rq)

	if err != nil {
		log.Println("Failed to report as claimed", err.Error())
		return false
	}

	return true
}

func (m *MissionsServiceV1) ClaimMission(ctx context.Context, rq *UserMission) (userMission *MissionProgress, err error) {
	userMission = collectUserMission(ctx, m.rdClient, rq.UserID, rq.MissionID)
	if userMission.Current >= userMission.Goal {
		if userMission.ClaimedAt == 0 {
			userMission.ClaimedAt = time.Now().Unix()

			if m.onMissionClaimed != nil {

				if mission := m.collectMission(ctx, rq.MissionID); mission != nil {
					if m.onMissionClaimed(ctx, mission,rq.UserID) == false {
						return nil, errors.New("bad pipeline")
					}
				}
			}

			saveUserMission(ctx, m.rdClient, rq.UserID, userMission)

			return userMission, nil
		}
		return nil, errors.New("claimed already")
	}

	return nil, errors.New("not completed")
}
