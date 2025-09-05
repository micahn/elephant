package handlers

import (
	"log/slog"
	"net"
	"strings"

	"github.com/abenz1267/elephant/internal/providers"
	"github.com/abenz1267/elephant/pkg/pb/pb"
	"google.golang.org/protobuf/proto"
)

type ActivateRequest struct{}

func (a *ActivateRequest) Handle(cid uint32, conn net.Conn, data []byte) {
	req := &pb.ActivateRequest{}
	if err := proto.Unmarshal(data, req); err != nil {
		slog.Error("activationrequesthandler", "protobuf", err)

		return
	}

	provider := req.Provider

	if strings.HasPrefix(provider, "menus:") {
		provider = strings.Split(provider, ":")[0]
	}

	if p, ok := providers.Providers[provider]; ok {
		p.Activate(uint32(req.Qid), req.Identifier, req.Action, req.Arguments)
	}

	providers.Cleanup(uint32(req.Qid))
}
