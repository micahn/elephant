package main

import (
	"log/slog"
	"net"
	"time"

	"github.com/abenz1267/elephant/v2/internal/util"
	"github.com/abenz1267/elephant/v2/pkg/common"
	"github.com/abenz1267/elephant/v2/pkg/pb/pb"
)

func Query(conn net.Conn, query string, _ bool, exact bool, _ uint8) []*pb.QueryResponse_Item {
	start := time.Now()

	entries := []*pb.QueryResponse_Item{}
	actions := []string{ActionOpen, ActionOpenDir, ActionCopyFile, ActionCopyPath}

	results := getFilesByQuery(query, exact)

	for k, v := range results {
		entry := &pb.QueryResponse_Item{
			Identifier:  v.Identifier,
			Text:        v.Path,
			Preview:     v.Path,
			PreviewType: util.PreviewTypeFile,
			Type:        pb.QueryResponse_REGULAR,
			Subtext:     "",
			Score:       int32(1000000000 - k),
			Provider:    Name,
			Actions:     actions,
		}

		if query != "" {
			score, pos, start := common.FuzzyScore(query, v.Path, exact)
			entry.Score = score
			entry.Fuzzyinfo = &pb.QueryResponse_Item_FuzzyInfo{
				Start:     start,
				Field:     "text",
				Positions: pos,
			}
		}

		entries = append(entries, entry)
	}

	slog.Info(Name, "queryresult", len(entries), "time", time.Since(start))

	return entries
}
