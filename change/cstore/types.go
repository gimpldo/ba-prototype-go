package cstore

import (
	"database/sql"

	"github.com/gimpldo/ba-prototype-go/change"
	"github.com/gimpldo/ba-prototype-go/id"
	"github.com/gimpldo/ba-prototype-go/sqlschema"
)

type SQLDefFactory interface {
	OpenSQLStoreReadOnly(db *sql.DB, storePrefix string) (ReadingSQLDef, error)
	OpenSQLStore(db *sql.DB, storePrefix string) (SQLDef, error)

	// CreateSQLStore should return an error if the store already exists =
	// there seems to be a changes store configuration table ('cstore_conf')
	// in the given database, with the given prefix.
	// In other words, the intended behavior is similar to
	// the POSIX function open(..., O_CREAT | O_EXCL | O_RDWR, ...)
	// minus the atomicity guarantee.
	//
	CreateSQLStore(db *sql.DB, storePrefix, schemaCreationOptions string) (SQLDef, error)
}

// ReadingSQLDef = Read access interface for the changes store Definition
// (assuming SQL database).
//
// Abstracts and encapsulates the specifics of data definition language (DDL)
// and any special needs deriving from the schema used.
//
type ReadingSQLDef interface {
	CheckCStoreSchema() (sqlschema.OpReport, error)

	UseCStoreReadOnly() (ReadingDop, error)
}

// SQLDef = read and write access interface for the changes store Definition
// (assuming SQL database).
//
// Abstracts and encapsulates the specifics of data definition language (DDL)
// and any special needs deriving from the schema used.
//
type SQLDef interface {
	ReadingSQLDef

	CreateCStoreSchemaElements() (sqlschema.OpReport, error)
	DropCStoreSchemaElements() (sqlschema.OpReport, error)

	UseCStore() (Dop, error)
}

// ReadingDop = interface for the changes store Data Operator (read access).
//
// Abstracts and encapsulates the specifics of data manipulation language (DML)
// and any special needs deriving from the schema used.
//
type ReadingDop interface {
	MakeCRecPullSourceCloser() (change.RecPullSourceCloser, error)

	Close()
}

// Dop = interface for the changes store Data Operator (read and write access)
//
// Abstracts and encapsulates the specifics of data manipulation language (DML)
// and any special needs deriving from the schema used.
//
type Dop interface {
	ReadingDop

	MakeInternalIDGetCloser() (id.InternalIDGetCloser, error)
	MakeExternalIDGetCloser() (id.ExternalIDGetCloser, error)

	MakeCRecPushSink(*sql.Tx) (change.RecPushSink, error)
	MakeCRecPushSinkEnder(*sql.Tx) (change.RecPushSinkEnder, error)
}
