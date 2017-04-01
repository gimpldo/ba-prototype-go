package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"

	"github.com/gimpldo/ba-prototype-go/change/cstore"
	"github.com/gimpldo/ba-prototype-go/geconf"
	"github.com/gimpldo/sqlite3-util-go/sqlite3tracemask"

	// Changes store implementations:
	"github.com/gimpldo/ba-prototype-go/cstoresqlite0"

	// Drivers for the 'database/sql' package:
	sqlite3 "github.com/gimpldo/go-sqlite3"
	_ "github.com/lib/pq"
)

func mapNameToSQLDefFactory(defName string) cstore.SQLDefFactory {
	switch defName {
	case "cstoresqlite0":
		return cstoresqlite0.SQLDefFactory{}
	default:
		return nil
	}
}

func traceCallback(info sqlite3.TraceInfo) int {
	// Not very readable but may be useful; uncomment next line in case of doubt:
	//fmt.Printf("Trace: %#v\n", info)

	// Suggestions for trace formatting
	//
	// DO NOT show only the fields that should be set (per spec),
	// show what *is* in the current instance, regardless of event/type code!
	//
	// Formatting decisions should be as local as possible,
	// that is, based on the value of the field you are printing and
	// maybe a closely related field.
	// Most of the time it's better to avoid formatting decisions based
	// on the record's type code ('EventCode' in this case).

	var dbErrText string
	if info.DBError.Code != 0 || info.DBError.ExtendedCode != 0 {
		dbErrText = fmt.Sprintf("; DB error: %#v", info.DBError)
	} else {
		dbErrText = "."
	}

	// Show the Statement-or-Trigger text in curly braces ('{', '}')
	// since from the *paired* ASCII characters they are
	// the least used in SQL syntax, therefore better visual delimiters.
	// Maybe show 'ExpandedSQL' the same way as 'StmtOrTrigger'.
	//
	// A known use of curly braces (outside strings) is
	// for ODBC escape sequences. Not likely to appear here.
	//
	// Template languages, etc. don't matter, we should see their *result*
	// at *this* level.
	// Strange curly braces in SQL code that reached the database driver
	// suggest that there is a bug in the application.
	// The braces are likely to be either template syntax or
	// a programming language's string interpolation syntax.

	var stmtOrTrigText string
	if info.StmtOrTrigger != "" {
		stmtOrTrigText = fmt.Sprintf(" {%q}", info.StmtOrTrigger)
	} else {
		stmtOrTrigText = ""
	}

	var expandedText string
	if info.ExpandedSQL != "" {
		if info.ExpandedSQL == info.StmtOrTrigger {
			expandedText = " = exp"
		} else {
			expandedText = fmt.Sprintf(" expanded {%q}", info.ExpandedSQL)
		}
	} else {
		expandedText = ""
	}

	// SQLite docs as of September 6, 2016: Tracing and Profiling Functions
	// https://www.sqlite.org/c3ref/profile.html
	//
	// The profile callback time is in units of nanoseconds, however
	// the current implementation is only capable of millisecond resolution
	// so the six least significant digits in the time are meaningless.
	// Future versions of SQLite might provide greater resolution on the profiler callback.

	var runTimeText string
	if info.RunTimeNanosec == 0 {
		if info.EventCode == sqlite3.TraceProfile {
			//runTimeText = "; no time" // seems confusing
			runTimeText = "; time 0" // no measurement unit
		} else {
			//runTimeText = "; no time" // seems useless and confusing
		}
	} else {
		const nanosPerMillisec = 1000000
		if info.RunTimeNanosec%nanosPerMillisec == 0 {
			runTimeText = fmt.Sprintf("; time %d ms", info.RunTimeNanosec/nanosPerMillisec)
		} else {
			// unexpected: better than millisecond resolution
			runTimeText = fmt.Sprintf("; time %d ns!!!", info.RunTimeNanosec)
		}
	}

	var modeText string
	if info.AutoCommit {
		modeText = "-AC-"
	} else {
		modeText = "+Tx+"
	}

	fmt.Printf("Trace: ev %d %s conn 0x%x, stmt 0x%x%s%s%s%s\n",
		info.EventCode, modeText, info.ConnHandle, info.StmtHandle,
		stmtOrTrigText, expandedText,
		runTimeText,
		dbErrText)
	return 0
}

type dbOpenInfo struct {
	dbDriverName string
	dbDSN        string // 'DSN' = "Data Source Name"
}

type impRequest struct {
	parser        string
	inputFilename string
}

type expRequest struct {
	serializer     string
	outputFilename string
}

// Changes Store Actions specification,
// filled based on command line options, for now.
//
// The order of actions is:
//    - create,
//    - expFirstReq (export first, before other steps),
//    - impReq (import),
//    - expLastReq (export last, after all other steps),
//    - drop.
//
// Intended/typical use is: only one or two of the five actions are selected.
//
// The trailing 'Req' stands for "Request" or "Requested".
//
type cstActions struct {
	prefix        string
	createOptions string

	createStore        bool
	createMissingElems bool
	dropAllElems       bool

	expFirstReq *expRequest
	expLastReq  *expRequest
	impReq      *impRequest
}

// Special (dummy) export format to avoid generating output:
// it's not the exact equivalent of using '/dev/null' (in Unix)
// instead of a regular output file because
// the exporter does not have any specific format to generate, so
// it will not format output strings in memory, skipping that step altogether.
//
// Should be faster than something like '--export-first-file=/dev/null'.
//
// The exporter must retrieve all relevant data from the database though ---
// this is what we test with the special 'discardingExport' format.
//
const discardingExport = "discard"

func main() {
	var (
		maskStr  string
		maskConf sqlite3tracemask.Config
	)
	flag.StringVar(&maskStr, "sqlite3-trace", "",
		"Supported SQLite3 trace event codes: s=Stmt, p=Profile, r=Row, c=Close")

	var (
		actions       cstActions
		cstoreDefName string
	)
	flag.StringVar(&actions.prefix, "cstore-elem-prefix", "",
		"Schema element name prefix for the changes store (optional: may be empty string)")
	flag.StringVar(&cstoreDefName, "cstore-def", "",
		"Schema definition name: identify the changes store implementation")

	flag.StringVar(&actions.createOptions, "cstore-create-options", "",
		"Schema definition options for the changes store (only used when creating a store)")
	flag.BoolVar(&actions.createStore, "create-cstore", false,
		"Create the changes store")
	flag.BoolVar(&actions.createMissingElems, "create-missing", false,
		"Create the missing schema elements for the changes store implementation")
	flag.BoolVar(&actions.dropAllElems, "drop-all", false,
		"Drop all the schema elements known by the changes store implementation")

	var (
		expFirstStruct expRequest
		expLastStruct  expRequest
		impStruct      impRequest
	)
	flag.StringVar(&expFirstStruct.serializer, "export-first-serializer", "", "")
	flag.StringVar(&expFirstStruct.outputFilename, "export-first-file", "", "")
	flag.StringVar(&expLastStruct.serializer, "export-last-serializer", "", "")
	flag.StringVar(&expLastStruct.outputFilename, "export-last-file", "", "")
	flag.StringVar(&impStruct.parser, "import-parser", "", "")
	flag.StringVar(&impStruct.inputFilename, "import-file", "", "")

	var openInfo dbOpenInfo
	flag.StringVar(&openInfo.dbDriverName, "db-driver", "", "Database driver name")
	flag.StringVar(&openInfo.dbDSN, "db-dsn", "", "Database to use (DSN means Data Source Name)")

	flag.Parse()

	if actions.createOptions != "" {
		// The verb "create" already appears in this argument.
		// Would look stupid to ask the user to repeat that,
		// so let's not require the boolean form ('--create-cstore')
		// in addition to '--cstore-create-options'.
		//
		actions.createStore = true

		var creationConfList geconf.List

		parsingErr := creationConfList.UnmarshalText([]byte(actions.createOptions))
		if parsingErr != nil {
			fmt.Printf("Could not parse the schema definition options: %#+v\n"+
				"  Options text: {%s}\n",
				parsingErr, actions.createOptions)
			os.Exit(36)
		}

		regeneratedConf, marshalingErr := creationConfList.MarshalText()
		if marshalingErr != nil {
			fmt.Printf("Could not marshal the parsed schema definition options: %#+v\n"+
				"  Options text: {%s}\n",
				marshalingErr, actions.createOptions)
			os.Exit(37)
		}

		fmt.Printf("Schema definition options parsed: {%s}\n",
			regeneratedConf)
	}
	// Note that '--cstore-create-options' is not required:
	// an empty string is a valid value for the CStore creation options.

	if openInfo.dbDriverName == "" {
		fmt.Println("Database driver not specified. Use --db-driver=...")
		os.Exit(11)
	}
	if openInfo.dbDSN == "" {
		fmt.Println("Data Source Name not specified. Use --db-dsn=...")
		os.Exit(12)
	}

	if expFirstStruct.serializer == discardingExport {
		if expFirstStruct.outputFilename != "" {
			fmt.Println("Unnecessary '--export-first-file': don't need file name when discarding exports")
			expFirstStruct.outputFilename = ""
			// Just warn and clear/forget the file name, do not exit: os.Exit(21)
		}
		actions.expFirstReq = &expFirstStruct
	} else if expFirstStruct.serializer != "" {
		if expFirstStruct.outputFilename == "" {
			fmt.Println("")
			os.Exit(22)
		}
		actions.expFirstReq = &expFirstStruct
	}

	if expFirstStruct.outputFilename != "" {
		if expFirstStruct.serializer == "" {
			fmt.Printf("Missing '--export-first-serializer=...': need serializer for output file '%s'\n",
				expFirstStruct.outputFilename)
			os.Exit(23)
		}
	}

	if expLastStruct.serializer == discardingExport {
		if expLastStruct.outputFilename != "" {
			fmt.Println("Unnecessary '--export-last-file': don't need file name when discarding exports")
			expLastStruct.outputFilename = ""
			// Just warn and clear/forget the file name, do not exit: os.Exit(25)
		}
		actions.expLastReq = &expLastStruct
	} else if expLastStruct.serializer != "" {
		if expLastStruct.outputFilename == "" {
			fmt.Println("")
			os.Exit(26)
		}
		actions.expLastReq = &expLastStruct
	}

	if expLastStruct.outputFilename != "" {
		if expLastStruct.serializer == "" {
			if expFirstStruct.serializer == "" {
				fmt.Printf("Missing '--export-last-serializer=...': need serializer for output file '%s'\n",
					expLastStruct.outputFilename)
				os.Exit(27)
			} else {
				expLastStruct.serializer = expFirstStruct.serializer

				fmt.Printf("Using the first serializer ('%s') for last output file '%s'\n",
					expLastStruct.serializer, expLastStruct.outputFilename)
				// Showing the _last_ serializer (that should have been set)
				// to make it easier to notice a bug if it's introduced here.

				actions.expLastReq = &expLastStruct
			}
		}
	}

	if impStruct.parser != "" {
		if impStruct.inputFilename == "" {
			fmt.Println("")
			os.Exit(28)
		}
		actions.impReq = &impStruct
	}
	if impStruct.inputFilename != "" {
		if impStruct.parser == "" {
			fmt.Printf("Missing '--import-parser=...': need parser for input file '%s'\n",
				impStruct.inputFilename)
			os.Exit(29)
		}
	}

	sqlite3tracemask.DecodeStringArg(&maskConf, maskStr)

	eventMask := maskConf.EventMask()
	if openInfo.dbDriverName == "sqlite3" && eventMask != 0 {
		openInfo.dbDriverName = "sqlite3_tracing"
		sql.Register(openInfo.dbDriverName,
			&sqlite3.SQLiteDriver{
				ConnectHook: func(conn *sqlite3.SQLiteConn) error {
					err := conn.SetTrace(&sqlite3.TraceConfig{
						Callback:         traceCallback,
						EventMask:        eventMask,
						WantExpandedSQL:  true,
						WantSQLiteErrMsg: true,
					})
					return err
				},
			})
	}

	if cstoreDefName == "" {
		fmt.Printf("Missing '--cstore-def=...': need schema definition name\n")
		os.Exit(31)
	}

	sqlDefFactory := mapNameToSQLDefFactory(cstoreDefName)
	if sqlDefFactory == nil {
		fmt.Printf("Unknown schema definition '%s'\n",
			cstoreDefName)
		os.Exit(32)
	}

	os.Exit(dbMain(openInfo, sqlDefFactory, actions))
}

// Harder to do DB work in main().
// It's better with a separate function because
// 'defer' and 'os.Exit' don't go well together.
//
// DO NOT use 'log.Fatal...' below: remember that it's equivalent to
// Print() followed by a call to os.Exit(1) --- and
// we want to avoid Exit() so 'defer' can do cleanup.
// Use 'log.Panic...' instead.

func dbMain(openInfo dbOpenInfo, sqlDefFactory cstore.SQLDefFactory, actions cstActions) int {
	db, err := sql.Open(openInfo.dbDriverName, openInfo.dbDSN)
	if err != nil {
		fmt.Printf("Failed to open '%s' database with DSN '%s': %#+v\n",
			openInfo.dbDriverName, openInfo.dbDSN, err)
		return 3
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		fmt.Printf("Failed to ping '%s' database with DSN '%s': %#+v\n",
			openInfo.dbDriverName, openInfo.dbDSN, err)
		return 4
	}

	var sqlDef cstore.SQLDef

	if actions.createStore {
		sqlDef, err = sqlDefFactory.CreateSQLStore(db, actions.prefix, actions.createOptions)
		if err != nil {
			fmt.Printf("Failed to create store and get SQLDef instance: %#+v\n",
				err)
			return 5
		}
	} else {
		sqlDef, err = sqlDefFactory.OpenSQLStore(db, actions.prefix)
		if err != nil {
			fmt.Printf("Failed to open store and get SQLDef instance: %#+v\n",
				err)
			return 6
		}
	}

	reportFromCheck, err := sqlDef.CheckCStoreSchema()
	if err != nil {
		fmt.Printf("Check failed for '%s' database with DSN '%s': %#+v\n",
			openInfo.dbDriverName, openInfo.dbDSN, err)
		return 7
	}
	fmt.Println("Report from check")
	reportFromCheck.Dump(os.Stdout, 2)

	if actions.createStore || actions.createMissingElems {
		reportFromCreate, createErr := sqlDef.CreateCStoreSchemaElements()
		if createErr != nil {
			fmt.Printf("Schema elements creation failed for '%s' database with DSN '%s': %#+v\n",
				openInfo.dbDriverName, openInfo.dbDSN, createErr)
			return 7
		}
		fmt.Println("Report from schema elements creation")
		reportFromCreate.Dump(os.Stdout, 2)
	}

	// 'dop' in this case is a changes store Data Operator instance:
	dop, err := sqlDef.UseCStore()
	if err != nil {
		fmt.Printf("Could not get a Data Operator instance to use '%s' database with DSN '%s': %#+v\n",
			openInfo.dbDriverName, openInfo.dbDSN, err)
		return 7
	}
	defer dop.Close()

	if actions.expFirstReq != nil {
		ret := doExport(*actions.expFirstReq, dop)
		if ret != 0 {
			return ret
		}
	}
	if actions.impReq != nil {
		ret := doImport(*actions.impReq, dop)
		if ret != 0 {
			return ret
		}
	}
	if actions.expLastReq != nil {
		ret := doExport(*actions.expLastReq, dop)
		if ret != 0 {
			return ret
		}
	}

	if actions.dropAllElems {
		reportFromDrop, dropErr := sqlDef.DropCStoreSchemaElements()
		if dropErr != nil {
			fmt.Printf("Schema elements drop failed for '%s' database with DSN '%s': %#+v\n",
				openInfo.dbDriverName, openInfo.dbDSN, dropErr)
			return 9
		}
		fmt.Println("Report from drop")
		reportFromDrop.Dump(os.Stdout, 2)
	}

	return 0
}

func doExport(req expRequest, dop cstore.ReadingDop) int {

	return 0
}

func doImport(req impRequest, dop cstore.Dop) int {

	return 0
}
