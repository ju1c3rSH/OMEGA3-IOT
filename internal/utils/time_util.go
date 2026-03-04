package utils

import (
	"time"
)

type TimeConverter struct{}

func (tc TimeConverter) MsToISO(ms int64) string {
	return time.UnixMilli(ms).Format("2006-01-02T15:04:05")
}

func (tc TimeConverter) SecToISO(sec int64) string {
	return time.Unix(sec, 0).Format("2006-01-02T15:04:05")
}

func (tc TimeConverter) ISOToMs(iso string) (int64, error) {
	t, err := time.Parse("2006-01-02T15:04:05", iso)
	if err != nil {
		return 0, err
	}
	return t.UnixMilli(), nil
}

func (tc TimeConverter) NowMs() int64 {
	return time.Now().UnixMilli()
}

func (tc TimeConverter) NowISO() string {
	return time.Now().Format("2006-01-02T15:04:05")
}
