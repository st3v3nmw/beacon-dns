package ids

import (
	"sync"
	"time"

	"github.com/oklog/ulid/v2"
	"golang.org/x/exp/rand"
)

var (
	entropySource = rand.New(rand.NewSource(uint64(time.Now().UnixNano())))
	entropyMu     sync.Mutex
	zeroValueULID ulid.ULID
)

func GenerateULID() (ulid.ULID, error) {
	entropyMu.Lock()
	defer entropyMu.Unlock()

	now := time.Now()
	entropy := ulid.Monotonic(entropySource, 0)
	id, err := ulid.New(ulid.Timestamp(now), entropy)
	if err != nil {
		return zeroValueULID, err
	}

	return id, nil
}
