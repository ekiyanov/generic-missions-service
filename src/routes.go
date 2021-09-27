// Schemes: http
// Host: localhost:8081
// Version: 0.0.1
// basePath: /api/missionsv1
// Consumes:
// - application/json
// Produces:
// - application/json
//swagger:meta

package missions

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/ekiyanov/redisclient"
)

var service MissionsService

func provideMissionsLocalJson() func() []*Mission {

	var _missions []*Mission

	return func() []*Mission {

		if _missions == nil {
			log.Println("reading missions.json")
			f, err := os.Open("./missions.json")
			if err != nil {
				log.Println("Failed to open ./missions.json", err)
				return nil
			}

			defer f.Close()

			decoder := json.NewDecoder(f)
			if err = decoder.Decode(&_missions); err != nil {
				_missions = nil
				log.Println("Failed to decode ./missions.json", err)
				return nil
			}

		}

		return _missions
	}
}

func HttpMux(prefix string, claimCallback MissionClaimedCback) http.Handler {
	mux := http.NewServeMux()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	defer cancel()

	ms1 := NewMissionServiceV1(redisclient.NewRedisClientCtx(ctx), provideMissionsLocalJson())
	ms1.onMissionClaimed = claimCallback
	service = ms1

	// /missions to return list of available missions in general
	// swagger:operation GET /missions missions
	//
	// ---
	// responses:
	//  '200':
	//   description: success
	//   schema:
	//    $ref: '#/definitions/ListMissionsRS'
	mux.HandleFunc(prefix+"/missions", handleMissions)

	//	/progress POST to mark mission as complete
	// swagger:operation GET /progress progress
	//
	// ---
	// parameters:
	// - name: u
	//   in: query
	//   description: The id for the user that needs to be fetched.
	//   required: true
	//   type: string
	// responses:
	//  '200':
	//   description: success
	//   schema:
	//    $ref: '#/definitions/UserMissionProgressRS'

	// swagger:operation POST /progress progress
	//
	// ---
	// parameters:
	// - in: body
	//   name: progress
	//   description: The userID and counter which to increase
	//   schema:
	//     $ref: '#/definitions/ReportUserProgressRQ'
	// responses:
	//  '200':
	//   description: success
	//   schema:
	//    $ref: '#/definitions/ReportUserProgressRS'
	mux.HandleFunc(prefix+"/progress", handleProgress)

	// swagger:operation POST /claim claim
	//
	// ---
	// parameters:
	// - in: body
	//   name: claim
	//   description: Claim user's mission
	//   schema:
	//     $ref: '#/definitions/UserMission'
	// responses:
	//  '200':
	//   description: success
	//   schema:
	//    $ref: '#/definitions/MissionProgress'
	mux.HandleFunc(prefix+"/claim", handleClaim)

	// swagger:operation POST /wipe wipe
	//
	// ---
	// parameters:
	// - in: body
	//   name: wipe
	//   description: Clears user's progress entirely
	//   schema:
	//   $ref: '#/definitions/WipeRequest'
	// responses:
	//  '200':
	//   description: success
	//   schema:
	//    $ref: '#/definitions/WipeRequestResponse'
	//  '400':
	//   description: bad sign of broken payload
	//
	mux.HandleFunc(prefix+"/wipe", handleWipe)

	return mux
}

func handleClaim(rw http.ResponseWriter, rq *http.Request) {

	if rq.Method == http.MethodPost {
		var serializedRequest = &UserMission{}
		defer rq.Body.Close()
		decoder := json.NewDecoder(rq.Body)
		if err := decoder.Decode(serializedRequest); err != nil {
			logError(err)
			rw.WriteHeader(http.StatusBadRequest)
			return
		}

		serializedResponse, err := service.ClaimMission(rq.Context(), serializedRequest)

		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(err.Error()))
			return
		}
		encoder := json.NewEncoder(rw)
		encoder.Encode(serializedResponse)
	}
}

func handleMissions(rw http.ResponseWriter, rq *http.Request) {
	missions := service.ListMissions(rq.Context())
	encoder := json.NewEncoder(rw)
	encoder.Encode(missions)
}

func handleWipe(rw http.ResponseWriter, rq *http.Request) {
	if rq.Method == http.MethodPost {
		var serRq = WipeRequest{}
		defer rq.Body.Close()

		decoder := json.NewDecoder(rq.Body)
		if err := decoder.Decode(&serRq); err != nil {
			logError(err)
			rw.WriteHeader(http.StatusBadRequest)
			return
		}

		if serRq.SignValid() == false {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}

		resp, err := service.WipeProgress(rq.Context(), &serRq)
		if err != nil {
			logError(err)
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		encoder := json.NewEncoder(rw)
		encoder.Encode(&resp)

	}
}
func handleProgress(rw http.ResponseWriter, rq *http.Request) {

	if rq.Method == http.MethodPost {
		defer rq.Body.Close()

		var reportUserProgressRQ ReportUserProgressRQ
		var reportUserProgressRS *ReportUserProgressRS

		decoder := json.NewDecoder(rq.Body)

		// todo: add error handle
		decoder.Decode(&reportUserProgressRQ)

		reportUserProgressRS = service.ReportUserProgress(rq.Context(), &reportUserProgressRQ)

		encoder := json.NewEncoder(rw)
		encoder.Encode(reportUserProgressRS)
	}

	if rq.Method == http.MethodGet {
		progress, err := service.MissionsUserProgress(rq.Context(), &MissionProgressRQ{UserID: rq.URL.Query().Get("u")})

		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		if progress == nil {
			rw.Write([]byte("[]"))
			return
		}
		encoder := json.NewEncoder(rw)
		encoder.Encode(progress)
	}
}
