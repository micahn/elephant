package handlers

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"log/slog"
	"net"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/abenz1267/elephant/internal/providers"
	"github.com/abenz1267/elephant/pkg/pb/pb"
	"google.golang.org/protobuf/proto"
)

const (
	QueryDone      = 255
	QueryNoResults = 254
	QueryItem      = 0
	QueryAsyncItem = 1
)

type queryData struct {
	Query     string
	Iteration atomic.Uint32
	cancel    context.CancelFunc
	sync.Mutex
}

var (
	qid                              atomic.Uint32
	queries                          = make(map[uint32]map[uint32]*queryData)
	queryMutex                       sync.Mutex
	MaxGlobalItemsToDisplayWebsearch = 0
	WebsearchPrefixes                = make(map[string]string)
)

type QueryRequest struct{}

func (h *QueryRequest) Handle(cid uint32, conn net.Conn, data []byte) {
	start := time.Now()

	req := &pb.QueryRequest{}
	if err := proto.Unmarshal(data, req); err != nil {
		slog.Error("queryhandler", "protobuf", err)

		return
	}

	wsprefix := ""

	if slices.Contains(req.Providers, "websearch") {
		for k, v := range WebsearchPrefixes {
			if strings.HasPrefix(req.Query, k) {
				wsprefix = v
			}
		}
	}

	queryMutex.Lock()
	if _, ok := queries[cid]; !ok {
		queries[cid] = make(map[uint32]*queryData)
	}
	queryMutex.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	isCncld := func() bool {
		select {
		case <-ctx.Done():
			return true
		default:
			return false
		}
	}

	var currentQID uint32
	var currentIteration uint32

	if req.Query != "" {
		lastLength := 0

		for k, v := range queries[cid] {
			if v.cancel != nil {
				v.cancel()
			}

			if strings.HasPrefix(req.Query, v.Query) && len(v.Query) > lastLength {
				currentQID = k
				lastLength = len(v.Query)
				v.Iteration.Add(1)
				currentIteration = v.Iteration.Load()
				v.cancel = cancel
			}
		}

		if currentQID == 0 {
			qid.Add(1)
			currentQID = qid.Load()

			queryMutex.Lock()
			providers.QueryProviders[currentQID] = req.Providers
			data := &queryData{
				Query:  req.Query,
				cancel: cancel,
			}
			data.Iteration.Add(1)
			currentIteration = data.Iteration.Load()
			queries[cid][currentQID] = data
			queryMutex.Unlock()

			slog.Info("providers", "query", "new", "qid", currentQID, "iid", currentIteration, "text", req.Query)
		} else {
			slog.Info("providers", "query", "resuming", "qid", currentQID, "iid", currentIteration, "text", req.Query)
		}
	} else {
		qid.Add(1)
		currentQID = qid.Load()
		currentIteration = 1
		slog.Info("providers", "query", "new", "qid", currentQID, "iid", currentIteration, "text", "<empty>")
	}

	var mut sync.Mutex

	var wg sync.WaitGroup
	wg.Add(len(req.Providers))

	providers.Timestampedqueries.Data[currentQID] = time.Now()

	entries := []*pb.QueryResponse_Item{}

	for _, v := range req.Providers {
		query := req.Query

		if strings.HasPrefix(v, "menus:") {
			split := strings.Split(v, ":")
			v = split[0]
			query = fmt.Sprintf("%s:%s", split[1], query)
		}

		go func(text string, wg *sync.WaitGroup) {
			defer wg.Done()
			if p, ok := providers.Providers[v]; ok {
				res := p.Query(currentQID, currentIteration, text, len(req.Providers) == 1, req.Exactsearch)

				mut.Lock()
				entries = append(entries, res...)
				mut.Unlock()
			}
		}(query, &wg)
	}

	wg.Wait()

	if isCncld() {
		return
	}

	slices.SortFunc(entries, sortEntries)

	if len(entries) == 0 {
		writeStatus(QueryNoResults, conn)
		writeStatus(QueryDone, conn)
		slog.Info("providers", "results", len(entries), "time", time.Since(start))
		return
	}

	if len(entries) > int(req.Maxresults) {
		entries = entries[:req.Maxresults]
	}

	hideWebsearch := len(req.Providers) > 1 && len(entries) > MaxGlobalItemsToDisplayWebsearch

	for _, v := range entries {
		if isCncld() {
			return
		}

		if v.Provider == "websearch" && hideWebsearch && v.Text != wsprefix {
			continue
		}

		if req.Query != "" && currentIteration != queries[cid][currentQID].Iteration.Load() {
			slog.Info("queryrequesthandler", "results", "aborting", "qid", currentQID, "iid", currentIteration)
			return
		}

		req := pb.QueryResponse{
			Qid:  int32(qid.Load()),
			Iid:  int32(currentIteration),
			Item: v,
		}

		b, err := proto.Marshal(&req)
		if err != nil {
			slog.Error("queryrequesthandler", "marshal", err)
			continue
		}

		var buffer bytes.Buffer
		buffer.Write([]byte{QueryItem})

		lengthBuf := make([]byte, 4)
		binary.BigEndian.PutUint32(lengthBuf, uint32(len(b)))
		buffer.Write(lengthBuf)
		buffer.Write(b)

		_, err = conn.Write(buffer.Bytes())
		if err != nil {
			slog.Error("queryrequesthandler", "write", err)
			return
		}
	}

	writeStatus(QueryDone, conn)

	slog.Info("providers", "results", len(entries), "time", time.Since(start))
}

func sortEntries(a *pb.QueryResponse_Item, b *pb.QueryResponse_Item) int {
	if a.Score > b.Score {
		return -1
	}

	if b.Score > a.Score {
		return 1
	}

	return strings.Compare(a.Text, b.Text)
}
