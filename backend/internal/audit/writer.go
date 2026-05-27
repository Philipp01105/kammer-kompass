package audit

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Philipp01105/kammer-kompass/backend/internal/db/sqlc"
	"github.com/Philipp01105/kammer-kompass/backend/internal/netx"
	"github.com/Philipp01105/kammer-kompass/backend/internal/security"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type Writer struct {
	q          *sqlc.Queries
	secretSalt string
}

func NewWriter(q *sqlc.Queries, secretSalt string) *Writer {
	return &Writer{q: q, secretSalt: secretSalt}
}

// CreateAuditLog creates a new audit log entry
func (w *Writer) CreateAuditLog(
	ctx context.Context,
	r *http.Request,
	actorUserID string,
	action string,
	resourceType string,
	resourceID *uuid.UUID,
	scopeType *string,
	scopeID *string,
	oldValue any,
	newValue any,
) error {
	actor, ok := parseUUID(actorUserID)
	if !ok {
		return nil
	}

	var resID pgtype.UUID
	if resourceID != nil {
		resID = pgtype.UUID{Bytes: *resourceID, Valid: true}
	} else {
		resID = pgtype.UUID{Valid: false}
	}

	oldJSON := jsonOrNull(oldValue)
	newJSON := jsonOrNull(newValue)

	ip := netx.ClientIP(r)
	ipHash := security.Sha256Hex(ip + w.secretSalt)

	ua := strings.TrimSpace(r.UserAgent())
	uaHash := ""
	if ua != "" {
		uaHash = security.Sha256Hex(ua + w.secretSalt)
	}

	ipHashPtr := (*string)(nil)
	if ipHash != "" {
		ipHashPtr = &ipHash
	}
	uaHashPtr := (*string)(nil)
	if uaHash != "" {
		uaHashPtr = &uaHash
	}

	_, err := w.q.CreateAuditLog(ctx, sqlc.CreateAuditLogParams{
		ActorUserID:   pgtype.UUID{Bytes: actor, Valid: true},
		Action:        action,
		ResourceType:  resourceType,
		ResourceID:    resID,
		ScopeType:     scopeType,
		ScopeID:       scopeID,
		OldValue:      oldJSON,
		NewValue:      newJSON,
		IpHash:        ipHashPtr,
		UserAgentHash: uaHashPtr,
	})
	return err
}

// CreateReviewEvent creates a new review event
func (w *Writer) CreateReviewEvent(
	ctx context.Context,
	targetType string,
	targetID uuid.UUID,
	actorUserID string,
	action string,
	oldStatus *string,
	newStatus *string,
	comment *string,
) error {
	actor, ok := parseUUID(actorUserID)
	if !ok {
		return nil
	}
	_, err := w.q.CreateReviewEvent(ctx, sqlc.CreateReviewEventParams{
		TargetType:  targetType,
		TargetID:    pgtype.UUID{Bytes: targetID, Valid: true},
		ActorUserID: pgtype.UUID{Bytes: actor, Valid: true},
		Action:      action,
		OldStatus:   oldStatus,
		NewStatus:   newStatus,
		Comment:     comment,
	})
	return err
}

func parseUUID(s string) (uuid.UUID, bool) {
	u, err := uuid.Parse(strings.TrimSpace(s))
	if err != nil {
		return uuid.UUID{}, false
	}
	return u, true
}

func jsonOrNull(v any) []byte {
	if v == nil {
		return nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	return b
}
