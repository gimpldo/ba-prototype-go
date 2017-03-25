package sqlite3schema

import (
	"bytes"
	"database/sql"
	"fmt"
	"math"
	"strings"

	"github.com/gimpldo/ba-prototype-go/sqlschema"
	"github.com/pkg/errors"
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

func ReadFromDB(db *sql.DB, namePrefix, nameSuffix string) ([]ElementFound, error) {
	// Must be the same character as in the SQL 'ESCAPE' clause below:
	const escapeCharForLike = '!'

	var err error

	escPrefix, err := checkEscapeForLike(namePrefix, escapeCharForLike)
	if err != nil {
		return nil, errors.Wrapf(err, "Prefix not acceptable")
	}
	escSuffix, err := checkEscapeForLike(nameSuffix, escapeCharForLike)
	if err != nil {
		return nil, errors.Wrapf(err, "Suffix not acceptable")
	}

	rows, err := db.Query(
		"SELECT type, rootpage, name, tbl_name, sql FROM sqlite_master WHERE name LIKE ? ESCAPE '!'",
		escPrefix+"%"+escSuffix)
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

// checkEscapeForLike uses a very strict (whitelist) approach to validation.
//
// Right now, uppercase letters are not accepted because
// there is no plan or need for this software to use uppercase
// in database schema element/object names, and
// the official SQLite style (D. Richard Hipp's) does not use them either.
// It would be an easy change to allow uppercase, should the need appear.
//
func checkEscapeForLike(nameFragment string, escapeChar rune) (escaped string, err error) {
	var buf bytes.Buffer

	for i, ch := range nameFragment {
		switch {
		case ch == escapeChar: // must be first case
			return "", fmt.Errorf("Name fragment contains the escape char (%x) at %d.", ch, i)
			/*
				If we wanted to escape it, it could have been done as follows:
				buf.WriteRune(escapeChar) // first use: to escape (itself, here)
				buf.WriteRune(escapeChar) // second use: to include it in the LIKE

				But we don't need to support the escape char in names, and
				it's not easy to handle reliably an arbitrary escape char that
				could be one of the characters that we accept or reject below!
			*/
		case 'a' <= ch && ch <= 'z':
			buf.WriteRune(ch)
		case '0' <= ch && ch <= '9':
			buf.WriteRune(ch)
		case ch == '_': // Underscore would be misinterpreted as wildcard, escape it:
			buf.WriteRune(escapeChar)
			buf.WriteRune(ch)
		default:
			return "", fmt.Errorf("Name fragment is not safe: %x at %d.", ch, i)
		}
	}

	return buf.String(), nil
}
