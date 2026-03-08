package utils

import (
	"hash/fnv"
	"sync"
	"time"
)

const (
	epoch             = int64(1577836800000)
	workerBits        = uint(5)
	datacenterBits    = uint(5)
	sequenceBits      = uint(12)
	workerMax         = int64(-1) ^ (int64(-1) << workerBits)
	datacenterMax     = int64(-1) ^ (int64(-1) << datacenterBits)
	sequenceMask      = int64(-1) ^ (int64(-1) << sequenceBits)
	workerShift       = sequenceBits
	datacenterShift   = sequenceBits + workerBits
	timestampShift    = sequenceBits + workerBits + datacenterBits
)

type Snowflake struct {
	mu            sync.Mutex
	timestamp     int64
	workerID      int64
	datacenterID  int64
	sequence      int64
}

var (
	snowflakeInstance *Snowflake
	snowflakeOnce     sync.Once
)

func NewSnowflake(workerID, datacenterID int64) (*Snowflake, error) {
	if workerID < 0 || workerID > workerMax {
		panic("worker ID out of range")
	}
	if datacenterID < 0 || datacenterID > datacenterMax {
		panic("datacenter ID out of range")
	}
	return &Snowflake{
		timestamp:    0,
		workerID:     workerID,
		datacenterID: datacenterID,
		sequence:     0,
	}, nil
}

func InitSnowflake(workerID, datacenterID int64) error {
	var err error
	snowflakeOnce.Do(func() {
		snowflakeInstance, err = NewSnowflake(workerID, datacenterID)
	})
	return err
}

func GenerateSnowflakeID() int64 {
	if snowflakeInstance == nil {
		InitSnowflake(1, 1)
	}
	return snowflakeInstance.NextID()
}

func (s *Snowflake) NextID() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UnixNano() / 1000000

	if s.timestamp == now {
		s.sequence = (s.sequence + 1) & sequenceMask
		if s.sequence == 0 {
			for now <= s.timestamp {
				now = time.Now().UnixNano() / 1000000
			}
		}
	} else {
		s.sequence = 0
	}

	s.timestamp = now

	id := ((now - epoch) << timestampShift) |
		(s.datacenterID << datacenterShift) |
		(s.workerID << workerShift) |
		s.sequence

	return id
}

func ParseUserIDFromUUID(uuid string) int64 {
	h := fnv.New64a()
	h.Write([]byte(uuid))
	hash := h.Sum64()
	return int64(hash % 9223372036854775807)
}
