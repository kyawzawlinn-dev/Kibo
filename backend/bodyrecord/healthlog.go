package bodyrecord

import "context"

// -------------------------------
// Health log CRUD
// -------------------------------

// ListHealthLog returns a profile's episodes, most recent day first.
func (r *Repository) ListHealthLog(ctx context.Context, userID int64) ([]HealthLogEntry, error) {
	rows, err := r.DB.QueryContext(ctx,
		"SELECT id, user_id, date, title, severity, notes FROM health_log WHERE user_id = ? ORDER BY date DESC, id DESC",
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []HealthLogEntry
	for rows.Next() {
		var e HealthLogEntry
		if err := rows.Scan(&e.ID, &e.UserID, &e.Date, &e.Title, &e.Severity, &e.Notes); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// AddHealthLogEntry inserts a new episode and returns its id.
func (r *Repository) AddHealthLogEntry(ctx context.Context, e HealthLogEntry) (int64, error) {
	res, err := r.DB.ExecContext(ctx,
		"INSERT INTO health_log(user_id, date, title, severity, notes) VALUES(?,?,?,?,?)",
		e.UserID, e.Date, e.Title, e.Severity, e.Notes,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// UpdateHealthLogEntry edits an episode in place (scoped to its owner).
func (r *Repository) UpdateHealthLogEntry(ctx context.Context, e HealthLogEntry) error {
	_, err := r.DB.ExecContext(ctx,
		"UPDATE health_log SET date = ?, title = ?, severity = ?, notes = ? WHERE id = ? AND user_id = ?",
		e.Date, e.Title, e.Severity, e.Notes, e.ID, e.UserID,
	)
	return err
}

// DeleteHealthLogEntry removes an episode (scoped to its owner).
func (r *Repository) DeleteHealthLogEntry(ctx context.Context, userID, id int64) error {
	_, err := r.DB.ExecContext(ctx,
		"DELETE FROM health_log WHERE id = ? AND user_id = ?", id, userID,
	)
	return err
}
