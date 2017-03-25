// cstoresqlite0/sqldefimpl.go: changes store SQLDef Implementation for SQLite
// v0 design and schema, intended for SQLite 3

package cstoresqlite0

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"github.com/gimpldo/ba-prototype-go/change/cstore"
	"github.com/gimpldo/ba-prototype-go/cstoreconfsql"
	"github.com/gimpldo/ba-prototype-go/geconf"
	"github.com/gimpldo/ba-prototype-go/sqlschema"
	"github.com/gimpldo/ba-prototype-go/util/geconfsql"
	"github.com/gimpldo/ba-prototype-go/util/sqlite3schema"
	"github.com/pkg/errors"
	"github.com/sergi/go-diff/diffmatchpatch"
)

const cstoreImplName = "cstoresqlite0"

type SQLDefFactory struct{}

type (
	commonDef struct {
		db     *sql.DB
		prefix string

		// The general/global and per-element configuration for the store,
		// as it was retrieved from the database (in most cases), or
		// as it should be written (this is the case of
		// a store being created in the current operation)
		dbConf []geconf.Entry

		elementDefs [nElements]sqlschema.ElementDef

		report sqlschema.OpReport
	}

	cstoreSQLiteReadingDef struct {
		commonDef
	}

	cstoreSQLiteDef struct {
		commonDef

		// "Store creation requested" flag
		createStoreReq bool
	}
)

func (SQLDefFactory) OpenSQLStoreReadOnly(db *sql.DB, storePrefix string) (cstore.ReadingSQLDef, error) {
	err := checkSafeNameChars(storePrefix)
	if err != nil {
		return nil, err
	}

	confEntries, err := cstoreconfsql.ReadConfFromDB(db, storePrefix)
	if err != nil {
		return nil, err
	}
	err = cstoreconfsql.CheckOrder(confEntries)
	if err != nil {
		return nil, err
	}

	sd := &cstoreSQLiteReadingDef{
		commonDef: commonDef{db: db, prefix: storePrefix, dbConf: confEntries},
	}

	err = setupElements(&sd.commonDef)
	if err != nil {
		return nil, err
	}

	return sd, nil
}

func (SQLDefFactory) OpenSQLStore(db *sql.DB, storePrefix string) (cstore.SQLDef, error) {
	err := checkSafeNameChars(storePrefix)
	if err != nil {
		return nil, err
	}

	confEntries, err := cstoreconfsql.ReadConfFromDB(db, storePrefix)
	if err != nil {
		return nil, err
	}
	err = cstoreconfsql.CheckOrder(confEntries)
	if err != nil {
		return nil, err
	}

	sd := &cstoreSQLiteDef{
		commonDef: commonDef{db: db, prefix: storePrefix, dbConf: confEntries},

		createStoreReq: false,
	}

	err = setupElements(&sd.commonDef)
	if err != nil {
		return nil, err
	}

	return sd, nil
}

func (SQLDefFactory) CreateSQLStore(db *sql.DB, storePrefix, schemaCreationOptions string) (cstore.SQLDef, error) {
	err := checkSafeNameChars(storePrefix)
	if err != nil {
		return nil, err
	}

	dbConfEntries, err := cstoreconfsql.ReadConfFromDB(db, storePrefix)
	n := len(dbConfEntries)
	if err != nil { // failure to read CStore configuration; good news...
		// but more checking is needed to avoid overwriting an old store:
		if n != 0 {
			return nil, errors.Wrapf(err,
				"Store probably exists: read failed with partial result (%d conf entries)",
				n)
		}
	} else { // CStore configuration read successfully; bad news for Create...
		return nil, errors.Errorf(
			"Store already exists (%d conf entries)",
			n)
	}

	var creationConfList geconf.List
	err = creationConfList.UnmarshalText([]byte(schemaCreationOptions))
	if err != nil {
		return nil, err
	}

	sort.Sort(geconf.CanonicalOrder(creationConfList))

	extendedConf := prependImplInfoToConf(creationConfList)

	sd := &cstoreSQLiteDef{
		commonDef: commonDef{db: db, prefix: storePrefix, dbConf: extendedConf},

		createStoreReq: true,
	}

	err = setupElements(&sd.commonDef)
	if err != nil {
		return nil, err
	}

	return sd, nil
}

func setupElements(c *commonDef) error {
	// Before using it to generate the expected schema element definitions,
	// validate the configuration by (re)generating its textual form.
	//
	// The textual form is useful for informational and debugging purposes,
	// so it will be saved in the SQL schema operations report.
	//
	// The validation is important when the configuration was just read
	// from the database = the CStore exists, we are not creating it now.

	regeneratedConf, marshalingErr := geconf.List(c.dbConf).MarshalText()
	if marshalingErr != nil {
		return errors.Wrapf(marshalingErr, "Could not marshal DB conf")
	}

	c.report.Conf = string(regeneratedConf)

	generateDefsForAllElements(c.elementDefs[:], elementTemplates[:],
		c.prefix, c.dbConf)

	sqlschema.InitOpReportElements(&c.report, c.elementDefs[:])

	return nil
}

func writeConfToDB(c *commonDef, confEntries []geconf.Entry) error {
	data := sqlTemplateData{Prefix: c.prefix}
	insertSQL := generateSQL(insertSQLTemplates[tableCStoreConfEI], data)

	return geconfsql.InsertEntriesIntoDB(c.db, insertSQL, confEntries)
}

func (sd *cstoreSQLiteDef) CreateCStoreSchemaElements() (sqlschema.OpReport, error) {
	dbElementsFound, err := sqlite3schema.ReadFromDB(sd.db, sd.prefix, "")
	if err == nil { // read OK
		err = checkCStoreSchema(&sd.commonDef, dbElementsFound)
		if err != nil { // unlikely; no error return in checkCStoreSchema() as of March 2017
			return sd.report, errors.Wrapf(err, "strange: checkCStoreSchema() failed")
		}
	} else {
		// In general, when creating a store we should expect
		// schema reading to fail, and we want to ignore this error.
		//
		// It seems that SQLite3 can successfully read schema from
		// an empty file, at least if it just created the database file:
		// acts as if the 'sqlite_master' table exists and has no rows.
		//
		// Therefore we don't expect error here in this file because
		// this is a SQLite-specific implementation of a changes store,
		// but in general the CreateCStoreSchemaElements method should
		// ignore the initial failure to read schema and
		// go on creating the required schema elements.
		//
		if !sd.createStoreReq {
			sd.report.LastOp = "check before create failed to get DB schema"
			return sd.report, err
		}
	}

	needWriteConf := false

	confTableStatus := sd.report.Elements[tableCStoreConfEI].Status
	switch confTableStatus {
	case sqlschema.MissingES:
		if !sd.createStoreReq {
			return sd.report, errors.Errorf(
				"Conf table missing")
		}
		needWriteConf = true
	case sqlschema.MismatchedES:
		return sd.report, errors.Errorf(
			"Conf table mismatched")
	case sqlschema.MatchedES:
		if sd.createStoreReq {
			return sd.report, errors.Errorf(
				"This SQLDef instance was made by CreateSQLStore() but conf table exists in DB")
		}
	default:
		panic(fmt.Sprintf(
			"Unexpected schema element status code %x for conf table after check",
			uint(confTableStatus)))
	}

	err = sqlschema.CreateElements(sd.db, &sd.report, sd.elementDefs[:], sqlschema.MissingES)
	if err != nil {
		return sd.report, err
	}

	if needWriteConf {
		err = writeConfToDB(&sd.commonDef, sd.dbConf)
	}

	return sd.report, err
}

func (sd *cstoreSQLiteDef) DropCStoreSchemaElements() (sqlschema.OpReport, error) {
	err := sqlschema.DropReportedElements(sd.db, &sd.report,
		true /* tryAll: means try to drop all elements, don't stop at first failure */)
	return sd.report, err
}

func (sd *cstoreSQLiteReadingDef) CheckCStoreSchema() (sqlschema.OpReport, error) {
	dbElementsFound, err := sqlite3schema.ReadFromDB(sd.db, sd.prefix, "")
	if err != nil {
		sd.report.LastOp = sqlschema.OpCheckFailDB
		return sd.report, err
	}

	err = checkCStoreSchema(&sd.commonDef, dbElementsFound)
	return sd.report, err
}

func (sd *cstoreSQLiteDef) CheckCStoreSchema() (sqlschema.OpReport, error) {
	dbElementsFound, err := sqlite3schema.ReadFromDB(sd.db, sd.prefix, "")
	if err != nil {
		sd.report.LastOp = sqlschema.OpCheckFailDB
		return sd.report, err
	}

	err = checkCStoreSchema(&sd.commonDef, dbElementsFound)
	return sd.report, err
}

func checkCStoreSchema(c *commonDef, dbElementsFound []sqlite3schema.ElementFound) error {
	report := &c.report

	dbNamesFound := make(map[string]*sqlite3schema.ElementFound)
	for i := range dbElementsFound {
		dbNamesFound[dbElementsFound[i].Name] = &dbElementsFound[i]
	}

	report.NumFound = 0
	report.NumMissing = 0
	report.NumMatched = 0
	report.NumMismatched = 0

	for i := range report.Elements {
		statusRec := &report.Elements[i]
		elem := c.elementDefs[i]

		if statusRec.ElemType != elem.ElemType {
			panic(fmt.Sprintf("different ElemType at %d", i))
		}
		if statusRec.BaseName != elem.BaseName {
			panic(fmt.Sprintf("different BaseName at %d", i))
		}
		if statusRec.Name != elem.Name {
			panic(fmt.Sprintf("different Name at %d", i))
		}

		dbElemFound, ok := dbNamesFound[elem.Name]
		if ok {
			// Just in case no other status is assigned,
			// mark as Found so we know how far it got; normally
			// this should be overriden by Matched or Mismatched.
			statusRec.Status = sqlschema.FoundES
			report.NumFound++

			if dbElemFound.ElemType == elem.ElemType {
				foundSQL := strings.TrimSpace(dbElemFound.CreateSQL)

				// No need to trim 'elem.CreateSQL' because
				// it should be already without leading or trailing space;
				// see generateSQL(): returns string(bytes.TrimSpace(buf.Bytes()))

				if foundSQL == elem.CreateSQL {
					statusRec.Status = sqlschema.MatchedES
					report.NumMatched++
				} else {
					statusRec.Status = sqlschema.MismatchedES
					statusRec.ProblemDetail = diff(
						foundSQL, elem.CreateSQL)
					report.NumMismatched++
				}
			} else {
				statusRec.Status = sqlschema.MismatchedES
				statusRec.ProblemDetail = fmt.Sprintf(
					"type %d in DB != %d in def",
					dbElemFound.ElemType, elem.ElemType)
				report.NumMismatched++
			}
		} else {
			statusRec.Status = sqlschema.MissingES
			report.NumMissing++
		}
	}

	report.LastOp = sqlschema.OpCheck
	return nil
}

func (sd *cstoreSQLiteReadingDef) UseCStoreReadOnly() (cstore.ReadingDop, error) {
	dop := &cstoreSQLiteReadingDop{db: sd.db}
	return dop, nil
}

func (sd *cstoreSQLiteDef) UseCStoreReadOnly() (cstore.ReadingDop, error) {
	dop := &cstoreSQLiteReadingDop{db: sd.db}
	return dop, nil
}

func (sd *cstoreSQLiteDef) UseCStore() (cstore.Dop, error) {
	dop := &cstoreSQLiteDop{db: sd.db}
	generateInsertSQLForAllTables(dop.insertSQL[:], insertSQLTemplates[:], sd.prefix)
	return dop, nil
}

func prependImplInfoToConf(confEntries []geconf.Entry) []geconf.Entry {
	cstoreImplNameEntry := geconf.Entry{
		ConfProperty: "CStoreImplName",
		ConfValue:    cstoreImplName,
	}

	extendedConf := make([]geconf.Entry, 0, len(confEntries)+1)
	extendedConf = append(extendedConf, cstoreImplNameEntry)
	extendedConf = append(extendedConf, confEntries...)

	return extendedConf
}

func diff(text1st, text2nd string) string {
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(text1st, text2nd, false)
	//_= diffs
	//return dmp.DiffPrettyText(diffs)
	return fmt.Sprintf("diffs=%v", diffs)
	//return text1st + "\n---\n" + text2nd
}
