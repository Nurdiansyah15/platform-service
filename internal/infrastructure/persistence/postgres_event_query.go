package persistence

import (
	"context"
	"fmt"

	"platform-service/internal/app/port"
	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/webhook/constant"
)

func sprintf(format string, args ...any) string { return fmt.Sprintf(format, args...) }

// List: event yang relevan untuk solution (aturan routing yang sama dengan webhook),
// urut (occurred_at, id) dengan cursor agar event ber-timestamp sama tidak terlewat.
func (r *PostgresEventQuery) List(ctx context.Context, q port.EventListReadQuery) ([]port.EventReadItem, error) {
	var w whereBuilder
	w.args = append(w.args, q.SolutionID, liveStatuses)
	w.conds = append(w.conds, sprintf(eventRoutedToSolution, "$1", "$2"))
	if q.Since != nil {
		w.add(`e.occurred_at >= $%d`, *q.Since)
	}
	if q.EventType != nil {
		w.add(`e.event_type = $%d`, *q.EventType)
	}
	if q.CompanyID != nil {
		w.add(`e.company_id = $%d`, *q.CompanyID)
	}
	if q.AfterOccurredAt != nil {
		w.add(`(e.occurred_at, e.id) > ($%d, $%d)`, *q.AfterOccurredAt, q.AfterID)
	}
	w.args = append(w.args, q.Limit)
	rows, err := execFromContext(ctx, r.db).QueryContext(ctx, sprintf(`
		SELECT e.id, e.event_type, e.company_id, e.solution_id, e.payload, e.occurred_at
		FROM events e %s ORDER BY e.occurred_at, e.id LIMIT $%d`, w.sql(), len(w.args)), w.args...)
	if err != nil {
		return nil, domainerr.Wrap(constant.CodeWebhookQueryFailed, err)
	}
	defer rows.Close()

	items := []port.EventReadItem{}
	for rows.Next() {
		var it port.EventReadItem
		if err := rows.Scan(&it.ID, &it.EventType, &it.CompanyID, &it.SolutionID, &it.Payload, &it.OccurredAt); err != nil {
			return nil, domainerr.Wrap(constant.CodeWebhookQueryFailed, err)
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, domainerr.Wrap(constant.CodeWebhookQueryFailed, err)
	}
	return items, nil
}
