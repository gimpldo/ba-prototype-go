package sqlite3schema

import (
	"database/sql"
	"fmt"
	"math"
	"strings"

	"github.com/gimpldo/ba-prototype-go/sqlschema"
)

func ReadFromDB(db *sql.DB, namePrefix, nameSuffix string) ([]ElementFound, error) {
	var err error
	err = checkSafeNameChars(namePrefix)
	if err != nil {
		panic(err)
	}
	err = checkSafeNameChars(nameSuffix)
	if err != nil {
		panic(err)
	}
	rows, err := db.Query(
		"SELECT type, rootpage, name, tbl_name, sql FROM sqlite_master WHERE name LIKE ?",
		namePrefix+"%"+nameSuffix)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return readSQLiteMasterRows(rows)
}

func readSQLiteMasterRows(rows *sql.Rows) ([]ElementFound, error) {
	var result []ElementFound

	for rows.Next() {
		var (
			// From 'https://www.sqlite.org/fileformat.html':
			// "For rows that define views, triggers, and
			// virtual tables, the rootpage column is 0 or NULL."
			//
			// The maximum page number is 2147483646 (2**31 - 2)
			// but there is no standard type 'sql.NullInt32'.
			//
			elemRootPage sql.NullInt64

			// From 'https://www.sqlite.org/fileformat.html':
			// "sqlite_master.sql is NULL for the internal indexes
			// that are automatically created by
			// UNIQUE or PRIMARY KEY constraints."
			//
			elemSQL sql.NullString

			elemTypeText string
			elemFound    ElementFound
		)
		err := rows.Scan(&elemTypeText, &elemRootPage,
			&elemFound.Name, &elemFound.TableName, &elemSQL)
		if err != nil {
			return result, err
		}

		if elemRootPage.Valid {
			if elemRootPage.Int64 < 0 {
				return result, fmt.Errorf(
					"Negative root page number %d (type %q, name %q, tbl_name %q)",
					elemRootPage.Int64, elemTypeText, elemFound.Name, elemFound.TableName)
			}
			if elemRootPage.Int64 >= math.MaxInt32 {
				return result, fmt.Errorf(
					"Root page number too big: %d (type %q, name %q, tbl_name %q)",
					elemRootPage.Int64, elemTypeText, elemFound.Name, elemFound.TableName)
			}
			elemFound.RootPage = int32(elemRootPage.Int64)
		} else {
			// 'sqlite_master.rootpage' is NULL:
			elemFound.RootPage = -1
		}

		if elemSQL.Valid {
			elemFound.CreateSQL = elemSQL.String
		} else {
			// 'sqlite_master.sql' is NULL:
			if !(elemTypeText == "index" &&
				strings.HasPrefix(elemFound.Name, "sqlite_autoindex_")) {
				return result, fmt.Errorf(
					"Unexpected NULL in 'sqlite_master.sql' (type %q, name %q, tbl_name %q, rootpage %d)",
					elemTypeText, elemFound.Name, elemFound.TableName, elemFound.RootPage)
			}
		}

		switch elemTypeText {
		case "table":
			elemFound.ElemType = sqlschema.TableElem
			if elemFound.Name != elemFound.TableName {
				return result, fmt.Errorf(
					"Bad table: name %q != tbl_name %q",
					elemFound.Name, elemFound.TableName)
			}
		case "index":
			elemFound.ElemType = sqlschema.IndexElem
		case "view":
			elemFound.ElemType = sqlschema.ViewElem
			if elemFound.Name != elemFound.TableName {
				return result, fmt.Errorf(
					"Bad view: name %q != tbl_name %q",
					elemFound.Name, elemFound.TableName)
			}
		case "trigger":
			elemFound.ElemType = sqlschema.TriggerElem
		default:
			return result, fmt.Errorf(
				"Unexpected element type %q (name %q, tbl_name %q)",
				elemTypeText, elemFound.Name, elemFound.TableName)
		}

		result = append(result, elemFound)
	}
	if err := rows.Err(); err != nil {
		return result, err
	}

	return result, nil
}

// Note: Right now, uppercase letters are not accepted because this software
// does not plan to use them in database schema element/object names,
// and the official SQLite style (D. Richard Hipp's) does not use them.
// It would be an easy change to allow uppercase, should the need appear.
func checkSafeNameChars(nameFragment string) error {
	for i, ch := range nameFragment {
		switch {
		case 'a' <= ch && ch <= 'z':
		case '0' <= ch && ch <= '9':
		case ch == '_':
		default:
			return fmt.Errorf("Name fragment is not safe: %x at %d.", ch, i)
		}
		// Problem: case ch == '_' will be misinterpreted as wildcard.
		// We want to have underscores in name fragments but
		// underscore has special meaning in 'LIKE'.
		// Not (yet) worth the trouble to quote the underscores with
		// an 'ESCAPE' character specified in the query. TODO!
	}
	return nil
}
