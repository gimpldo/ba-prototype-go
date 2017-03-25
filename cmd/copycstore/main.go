// copycstore

package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"

	"github.com/gimpldo/ba-prototype-go/change/cstore"
	"github.com/gimpldo/ba-prototype-go/geconf"

	// Changes store implementations:
	"github.com/gimpldo/ba-prototype-go/cstoresqlite0"

	// Drivers for the 'database/sql' package:
	_ "github.com/gimpldo/go-sqlite3"
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

type dbOpenInfo struct {
	dbDriverName string
	dbDSN        string // 'DSN' = "Data Source Name"
}

type cstoreInfo struct {
	dbOpenInfo

	label string // for messages

	sqlDefFactory cstore.SQLDefFactory

	prefix        string
	createOptions string

	createStore        bool
	createMissingElems bool

	truncate bool // TODO: not sure the truncation flag is useful now
	// (but I planned to have Truncate... methods in SQLDef interface)
}

const specialDSNSameDB = "." // Must be non-empty string; chose a single dot for this purpose

func main() {
	var (
		sourceInfo    cstoreInfo
		sourceDefName string
	)
	flag.StringVar(&sourceInfo.dbDriverName, "from-db-driver", "", "Source database driver name")
	flag.StringVar(&sourceInfo.dbDSN, "from-db-dsn", "",
		"Source database to read from (DSN means Data Source Name)")
	flag.StringVar(&sourceInfo.prefix, "from-prefix", "",
		"Schema element name prefix for the source changes store (optional: may be empty string)")
	flag.StringVar(&sourceDefName, "from-def", "",
		"Schema definition name: identify the source changes store implementation")

	// There is no need for flags like
	// '--from-options=...' or '--source-options=...'
	// (schema definition options for the source CStore)
	// because the source CStore must exist, therefore
	// its options can be retrieved from its metadata, and
	// a CStore's options cannot be changed anyway ---
	// they were decided when the CStore was created.

	var (
		destInfo    cstoreInfo
		destDefName string
	)
	flag.StringVar(&destInfo.dbDriverName, "to-db-driver", "", "Destination database driver name")
	flag.StringVar(&destInfo.dbDSN, "to-db-dsn", "",
		"Destination database to write to ('.' means: use the source database)")
	flag.StringVar(&destInfo.prefix, "to-prefix", "",
		"Schema element name prefix for the destination changes store (optional: may be empty string)")
	flag.StringVar(&destDefName, "to-def", "",
		"Schema definition name: identify the destination changes store implementation")

	flag.StringVar(&destInfo.createOptions, "create-options", "",
		"Schema definition options for the destination changes store (only used when creating it)")
	flag.BoolVar(&destInfo.createStore, "create-dest-cstore", false,
		"Create the changes store in the destination database")
	flag.BoolVar(&destInfo.createMissingElems, "create-dest-missing", false,
		"Create the missing schema elements in the destination database")

	flag.Parse()

	if destInfo.createOptions != "" {
		// The verb "create" already appears in this argument.
		// Would look stupid to ask the user to repeat that,
		// so let's not require the boolean form ('--create-dest-cstore')
		// in addition to '--create-options'.
		//
		destInfo.createStore = true

		var creationConfList geconf.List

		parsingErr := creationConfList.UnmarshalText([]byte(destInfo.createOptions))
		if parsingErr != nil {
			fmt.Printf("Could not parse the schema definition options: %#+v\n"+
				"  Options text: {%s}\n",
				parsingErr, destInfo.createOptions)
			os.Exit(36)
		}

		regeneratedConf, marshalingErr := creationConfList.MarshalText()
		if marshalingErr != nil {
			fmt.Printf("Could not marshal the parsed schema definition options: %#+v\n"+
				"  Options text: {%s}\n",
				marshalingErr, destInfo.createOptions)
			os.Exit(37)
		}

		fmt.Printf("Schema definition options parsed: {%s}\n",
			regeneratedConf)
	}
	// Note that '--create-options' is not required:
	// an empty string is a valid value for the CStore creation options.

	if sourceInfo.dbDriverName == "" {
		fmt.Println("Source database driver not specified. Use --from-db-driver=...")
		os.Exit(11)
	}
	if sourceInfo.dbDSN == "" {
		fmt.Println("Source DSN not specified. Use --from-db-dsn=...")
		os.Exit(12)
	}
	if sourceDefName == "" {
		fmt.Println("Source definition name not specified. Use --from-def=...")
		os.Exit(13)
	}

	sameDB := false

	switch destInfo.dbDSN {
	case "":
		fmt.Println("Destination DSN not specified. Use --to-db-dsn=...")
		os.Exit(15)
	case specialDSNSameDB:
		if destInfo.dbDriverName != "" && destInfo.dbDriverName != sourceInfo.dbDriverName {
			fmt.Println("Looks like a request for copy to same database: destination and source driver must be identical")
			os.Exit(16)
		}
		destInfo.dbDriverName = ""
		sameDB = true
	default:
		if destInfo.dbDriverName == "" {
			if sourceInfo.dbDriverName == "" {
				panic("Source database driver not specified; should have exited already!")
			}

			destInfo.dbDriverName = sourceInfo.dbDriverName

			fmt.Printf("Using the source database driver ('%s') for the destination DSN '%s'\n",
				destInfo.dbDriverName, destInfo.dbDSN)
			// Showing the _destination_ driver name (that should have been set)
			// to make it easier to notice a bug if it's introduced here.
		}
	}

	if destDefName == "" {
		if sourceDefName == "" {
			panic("Source definition name not specified; should have exited already!")
		}

		destDefName = sourceDefName

		fmt.Printf("Using the source definition ('%s') for the destination\n",
			destDefName)
		// Showing the _destination_ definition name (that should have been set)
		// to make it easier to notice a bug if it's introduced here.
	}

	if sameDB {
		if sourceInfo.prefix == destInfo.prefix {
			fmt.Printf("Source and destination prefix cannot be identical (%q) when copying to same database.\n",
				sourceInfo.prefix)
			os.Exit(20)
		}
	}

	sourceInfo.sqlDefFactory = mapNameToSQLDefFactory(sourceDefName)
	if sourceInfo.sqlDefFactory == nil {

	}

	destInfo.sqlDefFactory = mapNameToSQLDefFactory(destDefName)
	if destInfo.sqlDefFactory == nil {

	}

	os.Exit(dbCopy(destInfo, sourceInfo, sameDB))
}

// Harder to do DB work in main().
// It's better with a separate function because
// 'defer' and 'os.Exit' don't go well together.
//
// DO NOT use 'log.Fatal...' below: remember that it's equivalent to
// Print() followed by a call to os.Exit(1) --- and
// we want to avoid Exit() so 'defer' can do cleanup.
// Use 'log.Panic...' instead.

func dbCopy(destInfo, sourceInfo cstoreInfo, sameDB bool) int {
	var err error

	var sourceDB *sql.DB

	sourceDB, err = sql.Open(sourceInfo.dbDriverName, sourceInfo.dbDSN)
	if err != nil {
		fmt.Printf("Failed to open the source '%s' database with DSN '%s': %#+v\n",
			sourceInfo.dbDriverName, sourceInfo.dbDSN, err)
		return 3
	}
	defer sourceDB.Close()

	err = sourceDB.Ping()
	if err != nil {
		fmt.Printf("Failed to ping the source '%s' database with DSN '%s': %#+v\n",
			sourceInfo.dbDriverName, sourceInfo.dbDSN, err)
		return 4
	}

	var destDB *sql.DB

	if sameDB {
		if destInfo.dbDriverName != "" {
			panic("Destination driver name should be empty when source and destination CStores are in the same database")
		}
		if destInfo.dbDSN != specialDSNSameDB {
			panic("Destination DSN should be special marker when source and destination CStores are in the same database")
		}

		destDB = sourceDB
	} else {
		if destInfo.dbDriverName == "" {
			panic("Destination database driver name should not be empty")
		}
		if destInfo.dbDSN == "" {
			panic("Destination database DSN should not be empty")
		}
		if destInfo.dbDSN == specialDSNSameDB {
			panic("Destination database DSN is wrong (unexpected special marker)")
		}

		destDB, err = sql.Open(destInfo.dbDriverName, destInfo.dbDSN)
		if err != nil {
			fmt.Printf("Failed to open the destination '%s' database with DSN '%s': %#+v\n",
				destInfo.dbDriverName, destInfo.dbDSN, err)
			return 5
		}
		defer destDB.Close()

		err = destDB.Ping()
		if err != nil {
			fmt.Printf("Failed to ping the destination '%s' database with DSN '%s': %#+v\n",
				destInfo.dbDriverName, destInfo.dbDSN, err)
			return 6
		}
	}

	sourceSQLDef, err := sourceInfo.sqlDefFactory.OpenSQLStoreReadOnly(sourceDB, sourceInfo.prefix)
	if err != nil {
		fmt.Printf("Failed to open store read-only and get source SQLDef instance: %#+v\n",
			err)
		return 21
	}

	var destSQLDef cstore.SQLDef

	if destInfo.createStore {
		destSQLDef, err = destInfo.sqlDefFactory.CreateSQLStore(destDB, destInfo.prefix, destInfo.createOptions)
		if err != nil {
			fmt.Printf("Failed to create store and get destination SQLDef instance: %#+v\n",
				err)
			return 22
		}
	} else {
		destSQLDef, err = destInfo.sqlDefFactory.OpenSQLStore(destDB, destInfo.prefix)
		if err != nil {
			fmt.Printf("Failed to open store and get destination SQLDef instance: %#+v\n",
				err)
			return 23
		}
	}

	reportFromSourceCheck, err := sourceSQLDef.CheckCStoreSchema()
	if err != nil {
		fmt.Printf("Check failed for source '%s' database with DSN '%s': %#+v\n",
			sourceInfo.dbDriverName, sourceInfo.dbDSN, err)
		return 25
	}

	reportFromDestCheck, err := destSQLDef.CheckCStoreSchema()
	if err != nil {
		fmt.Printf("Check failed for destination '%s' database with DSN '%s': %#+v\n",
			destInfo.dbDriverName, destInfo.dbDSN, err)
		return 26
	}

	if destInfo.createStore || destInfo.createMissingElems {
		reportFromCreate, err := destSQLDef.CreateCStoreSchemaElements()
		if err != nil {
			fmt.Printf("Schema elements creation failed for destination '%s' database with DSN '%s': %#+v\n",
				destInfo.dbDriverName, destInfo.dbDSN, err)
			return 27
		}
	}

	// 'dop' in this case is a changes store Data Operator instance:
	sourceDop, err := sourceSQLDef.UseCStoreReadOnly()
	if err != nil {
		fmt.Printf("Could not get a Data Operator instance to use '%s' database with DSN '%s': %#+v\n",
			sourceInfo.dbDriverName, sourceInfo.dbDSN, err)
		return 31
	}
	defer sourceDop.Close()

	// 'dop' in this case is a changes store Data Operator instance:
	destDop, err := destSQLDef.UseCStore()
	if err != nil {
		fmt.Printf("Could not get a Data Operator instance to use '%s' database with DSN '%s': %#+v\n",
			destInfo.dbDriverName, destInfo.dbDSN, err)
		return 32
	}
	defer destDop.Close()

	return 0
}
