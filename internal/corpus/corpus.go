// Package corpus stores and loads text units and their translations.
package corpus

import (
	"context"
	"encoding/json"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Source describes where a translation comes from and on what terms.
type Source struct {
	ID, Title, Edition, URL, License, Note string
}

// Unit is a text unit with one translation, ready to be stored.
type Unit struct {
	ID      string
	Book    string
	Kind    string
	Tone    int // 0 = none
	Podoben string
	Meta    map[string]any

	Lang     string
	Version  string
	Body     string
	Incipit  string // derived from Body if empty
	SourceID string
}

// Text is a translation loaded for rendering.
type Text struct {
	UnitID  string
	Kind    string
	Tone    int
	Lang    string
	Body    string
	Incipit string
	Meta    map[string]any
}

// Store wraps the database pool.
type Store struct{ Pool *pgxpool.Pool }

// UpsertSource inserts or updates a source.
func (s *Store) UpsertSource(ctx context.Context, src Source) error {
	_, err := s.Pool.Exec(ctx, `INSERT INTO source (id, title, edition, url, license, note)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE SET title = EXCLUDED.title, edition = EXCLUDED.edition,
			url = EXCLUDED.url, license = EXCLUDED.license, note = EXCLUDED.note`,
		src.ID, src.Title, nullable(src.Edition), nullable(src.URL), src.License, nullable(src.Note))
	return err
}

// UpsertUnits stores units and their translations in one transaction.
func (s *Store) UpsertUnits(ctx context.Context, units []Unit) error {
	return pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		b := &pgx.Batch{}
		for _, u := range units {
			meta := u.Meta
			if meta == nil {
				meta = map[string]any{}
			}
			mj, err := json.Marshal(meta)
			if err != nil {
				return err
			}
			var tone any
			if u.Tone > 0 {
				tone = u.Tone
			}
			b.Queue(`INSERT INTO text_unit (id, book, kind, tone, podoben, meta) VALUES ($1, $2, $3, $4, $5, $6)
				ON CONFLICT (id) DO UPDATE SET book = EXCLUDED.book, kind = EXCLUDED.kind, tone = EXCLUDED.tone,
					podoben = EXCLUDED.podoben, meta = EXCLUDED.meta`,
				u.ID, u.Book, u.Kind, tone, nullable(u.Podoben), mj)
			version := u.Version
			if version == "" {
				version = "default"
			}
			inc := u.Incipit
			if inc == "" {
				inc = Incipit(u.Body)
			}
			b.Queue(`INSERT INTO text_translation (unit_id, lang, version, body, incipit, source_id)
				VALUES ($1, $2, $3, $4, $5, $6)
				ON CONFLICT (unit_id, lang, version) DO UPDATE SET body = EXCLUDED.body,
					incipit = EXCLUDED.incipit, source_id = EXCLUDED.source_id`,
				u.ID, u.Lang, version, u.Body, inc, u.SourceID)
		}
		return tx.SendBatch(ctx, b).Close()
	})
}

// DeleteBook removes all units of a book (and their translations), so that a
// re-import does not leave stale keys behind.
func (s *Store) DeleteBook(ctx context.Context, book string) (int64, error) {
	tag, err := s.Pool.Exec(ctx, `DELETE FROM text_unit WHERE book = $1`, book)
	return tag.RowsAffected(), err
}

// Load returns translations of the given units in lang (version 'default').
// Missing units are simply absent from the map.
func (s *Store) Load(ctx context.Context, lang string, ids []string) (map[string]Text, error) {
	rows, err := s.Pool.Query(ctx, `SELECT u.id, u.kind, coalesce(u.tone, 0), t.lang, t.body, t.incipit, u.meta
		FROM text_unit u JOIN text_translation t ON t.unit_id = u.id
		WHERE u.id = ANY($1) AND t.lang = $2 AND t.version = 'default'`, ids, lang)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]Text{}
	for rows.Next() {
		var t Text
		var tone int16
		if err := rows.Scan(&t.UnitID, &t.Kind, &tone, &t.Lang, &t.Body, &t.Incipit, &t.Meta); err != nil {
			return nil, err
		}
		t.Tone = int(tone)
		out[t.UnitID] = t
	}
	return out, rows.Err()
}

// ListIDs returns the ids of units whose id starts with prefix, sorted.
func (s *Store) ListIDs(ctx context.Context, prefix string) ([]string, error) {
	rows, err := s.Pool.Query(ctx, `SELECT id FROM text_unit WHERE starts_with(id, $1) ORDER BY id`, prefix)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}

// Incipit returns the first few words of a text, up to the first stop.
func Incipit(body string) string {
	body = strings.TrimSpace(strings.SplitN(body, "\n", 2)[0])
	words := strings.Fields(body)
	var b strings.Builder
	for i, w := range words {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(w)
		last, _ := utf8.DecodeLastRuneInString(w)
		if (i >= 2 && strings.ContainsRune(",:;.", last)) || i >= 5 {
			break
		}
	}
	return strings.TrimRight(b.String(), ",:;")
}

func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// UnitTones returns the tones of the given units (0 when none).
func (s *Store) UnitTones(ctx context.Context, ids []string) (map[string]int, error) {
	rows, err := s.Pool.Query(ctx, `SELECT id, coalesce(tone, 0) FROM text_unit WHERE id = ANY($1)`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var id string
		var tone int16
		if err := rows.Scan(&id, &tone); err != nil {
			return nil, err
		}
		out[id] = int(tone)
	}
	return out, rows.Err()
}

// UpsertTranslations stores translations of existing units (Book/Kind of the
// given units are ignored). Units unknown to the corpus are skipped and
// returned.
func (s *Store) UpsertTranslations(ctx context.Context, units []Unit) (skipped []string, err error) {
	ids := make([]string, len(units))
	for i, u := range units {
		ids[i] = u.ID
	}
	rows, err := s.Pool.Query(ctx, `SELECT id FROM text_unit WHERE id = ANY($1)`, ids)
	if err != nil {
		return nil, err
	}
	known, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, err
	}
	have := map[string]bool{}
	for _, id := range known {
		have[id] = true
	}
	err = pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		b := &pgx.Batch{}
		for _, u := range units {
			if !have[u.ID] {
				skipped = append(skipped, u.ID)
				continue
			}
			version := u.Version
			if version == "" {
				version = "default"
			}
			inc := u.Incipit
			if inc == "" {
				inc = Incipit(u.Body)
			}
			b.Queue(`INSERT INTO text_translation (unit_id, lang, version, body, incipit, source_id)
				VALUES ($1, $2, $3, $4, $5, $6)
				ON CONFLICT (unit_id, lang, version) DO UPDATE SET body = EXCLUDED.body,
					incipit = EXCLUDED.incipit, source_id = EXCLUDED.source_id`,
				u.ID, u.Lang, version, u.Body, inc, u.SourceID)
		}
		return tx.SendBatch(ctx, b).Close()
	})
	return skipped, err
}
