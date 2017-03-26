package geconfsql

import (
	"database/sql"
	"log"
	"strings"

	"github.com/gimpldo/ba-prototype-go/geconf"
	"github.com/pkg/errors"
)

// InsertEntriesIntoDB writes the given configuration entries into a database
// using the given SQL INSERT statement.
//
// Any configuration entries already in the database remain unchanged.
//
// Neither the table name nor the column names are hardcoded, but
// this means that the caller is responsible to specify the columns
// in the right order: first the Element name, then Property, Value last.
//
func InsertEntriesIntoDB(db *sql.DB, insertSQL string, confEntries []geconf.Entry) error {
	stmt, err := db.Prepare(insertSQL)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for i, entry := range confEntries {
		elem := strings.TrimSpace(entry.ConfElement) // may be empty

		prop := strings.TrimSpace(entry.ConfProperty)
		if prop == "" {
			log.Printf("Skipping entry %d/%d (empty ConfProperty): %#v",
				i, len(confEntries), entry)
			continue
		}

		val := strings.TrimSpace(entry.ConfValue)
		if val == "" {
			log.Printf("Skipping entry %d/%d (empty ConfValue): %#v",
				i, len(confEntries), entry)
			continue
		}

		result, insErr := stmt.Exec(elem, prop, val)
		if insErr != nil {
			return errors.Wrapf(insErr,
				"failed to insert conf entry %d/%d (%#v)",
				i, len(confEntries), entry)
		}

		nAffected, err := result.RowsAffected()
		if err != nil {
			return errors.Wrapf(err,
				"Cannot get affected count after inserting conf entry %d/%d (%#v)",
				i, len(confEntries), entry)
		}
		if nAffected != 1 {
			return errors.Wrapf(err,
				"Affected count is %d != 1 after inserting conf entry %d/%d (%#v)",
				nAffected, i, len(confEntries), entry)
		}
	}

	return nil
}
