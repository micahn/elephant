package main

import (
	"log/slog"
	"net"
	"time"

	"github.com/abenz1267/elephant/v2/pkg/pb/pb"
)

func Query(conn net.Conn, query string, _ bool, exact bool) []*pb.QueryResponse_Item {
	start := time.Now()

	entries := []*pb.QueryResponse_Item{}
	actions := []string{ActionOpen, ActionOpenDir, ActionCopyFile, ActionCopyPath}

	results := getFilesByQuery(query, exact)

	for _, v := range results {
		entries = append(entries, &pb.QueryResponse_Item{
			Identifier: v.f.Identifier,
			Text:       v.f.Path,
			Type:       pb.QueryResponse_REGULAR,
			Subtext:    "",
			Provider:   Name,
			Actions:    actions,
			Score:      v.score,
			Fuzzyinfo: &pb.QueryResponse_Item_FuzzyInfo{
				Start:     v.start,
				Field:     "text",
				Positions: v.positions,
			},
		})
	}

	slog.Info(Name, "queryresult", len(entries), "time", time.Since(start))

	return entries
}

func calcScore(v time.Time, now time.Time) int32 {
	if v.IsZero() {
		return 0
	}

	diff := now.Sub(v)

	res := 3600 - diff.Seconds()

	if res < 0 {
		res = 0
	}

	return int32(res)
}
