package sqlite3schema

import (
	"github.com/gimpldo/ba-prototype-go/sqlschema"
)

// ElementFound = SQLite database schema element found (after a search).
// Schema elements are also known as "schema objects".
type ElementFound struct {
	ElemType sqlschema.ElemTypeCode

	// page number -1 means that the rootpage column was NULL in 'sqlite_master'
	RootPage int32

	Name      string
	TableName string
	CreateSQL string
}
