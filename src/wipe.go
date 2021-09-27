package missions

import (
	"crypto/sha256"
	"fmt"
	"os"
	"time"
)

// swagger:model WipeRequest
type WipeRequest struct {
	// UserID for whom wipe the progress
	UserID string `json:"user_id"`

	// Hex encoded SHA256 hash of {EPOCH}{USERID}{SERVER_WIPE_KEY}
	Sign string `json:"signature"`

	// Time of the sending a request. Should not differs from server more than 25 seconds
	Epoch int64 `json:"epoch"`
}

// swagger:model WipeRequestResponse
type WipeRequestResponse struct {

	// Number of removed missions
	NumRemoved int64 `json:"removed"`
}

func NewWipeRequest(userId string) *WipeRequest {
	wr := WipeRequest{UserID: userId}
	wr.SignRq()
	return &wr
}

func (wr *WipeRequest) SignRq() {
	wr.Epoch = time.Now().Unix()

	base := fmt.Sprintf("%v%v", wr.Epoch, wr.UserID)

	hasher := sha256.New()
	hasher.Write([]byte(base))
	hasher.Write([]byte(os.Getenv("WIPEKEY")))

	wr.Sign = fmt.Sprintf("%x", hasher.Sum(nil))
}

func (wr *WipeRequest) SignValid() bool {
	if (time.Now().Unix() - wr.Epoch) > 25 {
		return false
	}

	base := fmt.Sprintf("%v%v", wr.Epoch, wr.UserID)

	hasher := sha256.New()
	hasher.Write([]byte(base))
	hasher.Write([]byte(os.Getenv("WIPEKEY")))
	validSign := fmt.Sprintf("%x", hasher.Sum(nil))

	if validSign != wr.Sign {
		return false
	}

	return true
}
