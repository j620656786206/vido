package database

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"time"

	"github.com/vido/api/internal/database/dbtime"
	"modernc.org/sqlite"
)

// DriverName is the database/sql driver the app's connection opens with. It
// is modernc's "sqlite" driver with one change: every time.Time bound as a
// query argument is written as a UTC, fixed-width text SQLite can read itself
// (see dbtime.Format).
//
// bugfix-h: left to itself the driver writes time.Time with t.String() —
// "2026-08-24 10:56:26.303854294 +0800 CST m=+26.660278123" — which SQLite's
// date()/datetime()/strftime() cannot parse (every one of them returned NULL
// on 100% of the production rows), and whose local-zone offset made plain
// string comparisons (ORDER BY created_at, WHERE expires_at < ?) wrong by
// hours whenever two rows were written under different TZ settings. Fixing it
// here means the 50-odd repository write sites, and every future one, get it
// without remembering a .UTC().
const DriverName = "sqlite-utc"

func init() {
	// Wrap the driver modernc registered under "sqlite" (sql.Open does not
	// connect) rather than a fresh &sqlite.Driver{}: the registered one carries
	// the package-level user-defined functions, collations and hooks.
	db, err := sql.Open("sqlite", "")
	if err != nil {
		panic(fmt.Sprintf("database: modernc sqlite driver unavailable: %v", err))
	}
	inner := db.Driver()
	_ = db.Close()
	sql.Register(DriverName, &utcDriver{inner: inner})
}

// sqliteConn is everything database/sql (and BackupService's online restore,
// via conn.Raw) uses from a modernc connection. Embedding it in utcConn
// forwards all of it unchanged.
type sqliteConn interface {
	driver.Conn
	driver.ConnBeginTx
	driver.ConnPrepareContext
	driver.ExecerContext
	driver.QueryerContext
	driver.Pinger
	driver.SessionResetter
	driver.Validator
	NewBackup(dstURI string) (*sqlite.Backup, error)
	NewRestore(srcURI string) (*sqlite.Backup, error)
}

type utcDriver struct {
	inner driver.Driver
}

func (d *utcDriver) Open(name string) (driver.Conn, error) {
	c, err := d.inner.Open(name)
	if err != nil {
		return nil, err
	}
	sc, ok := c.(sqliteConn)
	if !ok {
		_ = c.Close()
		return nil, fmt.Errorf("database: sqlite connection %T lacks a method the app relies on", c)
	}
	return &utcConn{sqliteConn: sc}, nil
}

type utcConn struct {
	sqliteConn
}

// CheckNamedValue runs database/sql's default conversion (which also resolves
// driver.Valuer such as sql.NullTime) and then turns any time.Time into the
// stored text. Every other value passes through exactly as before — modernc
// implements no checker of its own, so the default converter is what ran.
func (c *utcConn) CheckNamedValue(nv *driver.NamedValue) error {
	v, err := driver.DefaultParameterConverter.ConvertValue(nv.Value)
	if err != nil {
		return err
	}
	if t, ok := v.(time.Time); ok {
		v = dbtime.Format(t)
	}
	nv.Value = v
	return nil
}
