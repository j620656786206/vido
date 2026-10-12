package migrations

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// disc-2026-10-genre-names-simplified AC #3

func genresAndUpdatedAt(t *testing.T, db *sql.DB, table, id string) (genres, updatedAt string) {
	t.Helper()
	require.NoError(t, db.QueryRow(`SELECT genres, updated_at FROM `+table+` WHERE id = ?`, id).Scan(&genres, &updatedAt))
	return genres, updatedAt
}

func runMigration046(t *testing.T, db *sql.DB) {
	t.Helper()
	tx, err := db.Begin()
	require.NoError(t, err)
	require.NoError(t, (&canonicalizeGenreNames{}).Up(tx))
	require.NoError(t, tx.Commit())
}

func TestMigration046_RewritesGenreNamesInTaiwanWording(t *testing.T) {
	db := newFullyMigratedDB(t)

	// Raw INSERTs: the repository would already canonicalize. The NAS rows
	// (2026-10-12) look like the first two.
	for _, r := range []struct{ table, id, genres string }{
		{"series", "fanren", `["动画","动作冒险","剧情"]`},
		{"series", "see", `["剧情","Sci-Fi & Fantasy","War & Politics"]`},
		{"series", "mixed", `["剧情","劇情"]`}, // a hand edit next to a TMDb name
		{"movies", "tw", `["劇情","動作"]`},
		{"movies", "old-editor", `["drama","sci-fi"]`},
		{"movies", "unknown", `["武俠"]`},
		{"movies", "broken", `not json`},
		{"movies", "empty", `[]`},
	} {
		title := r.id
		var err error
		if r.table == "movies" {
			_, err = db.Exec(`INSERT INTO movies (id, title, release_date, genres, updated_at) VALUES (?, ?, '', ?, '2026-10-01 00:00:00')`, r.id, title, r.genres)
		} else {
			_, err = db.Exec(`INSERT INTO series (id, title, first_air_date, genres, updated_at) VALUES (?, ?, '', ?, '2026-10-01 00:00:00')`, r.id, title, r.genres)
		}
		require.NoError(t, err)
	}

	before := map[string]string{}
	for _, key := range []string{"series/fanren", "series/see", "series/mixed", "movies/tw", "movies/old-editor", "movies/unknown", "movies/broken", "movies/empty"} {
		table, id, _ := strings.Cut(key, "/")
		_, before[key] = genresAndUpdatedAt(t, db, table, id)
	}

	runMigration046(t, db)

	want := map[string][2]string{
		"series/fanren":     {"series", `["動畫","動作冒險","劇情"]`},
		"series/see":        {"series", `["劇情","科幻奇幻","戰爭政治"]`},
		"series/mixed":      {"series", `["劇情"]`},
		"movies/tw":         {"movies", `["劇情","動作"]`},
		"movies/old-editor": {"movies", `["劇情","科幻"]`},
		"movies/unknown":    {"movies", `["武俠"]`},
		"movies/broken":     {"movies", `not json`},
		"movies/empty":      {"movies", `[]`},
	}
	for key, w := range want {
		id := key[len(w[0])+1:]
		genres, updatedAt := genresAndUpdatedAt(t, db, w[0], id)
		assert.Equal(t, w[1], genres, key)
		assert.Equal(t, before[key], updatedAt, "%s: a spelling fix is not an edit", key)
	}

	// A second run changes nothing.
	n, err := func() (int, error) {
		tx, err := db.Begin()
		require.NoError(t, err)
		defer tx.Rollback() //nolint:errcheck
		return canonicalizeGenreColumn(tx, "series")
	}()
	require.NoError(t, err)
	assert.Zero(t, n)
}
