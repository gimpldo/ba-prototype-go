package cstoresqlite0

import (
	"bytes"
	"fmt"
	"log"
	"math"
	"sort"
	"strings"
	"text/template"

	"github.com/gimpldo/ba-prototype-go/geconf"
	"github.com/gimpldo/ba-prototype-go/sqlschema"
)

func generateDefsForAllElements(
	generated []sqlschema.ElementDef,
	templates []sqlschema.ElementTemplate,
	prefix string,
	confEntries []geconf.Entry) {

	err := checkSafeNameChars(prefix)
	if err != nil {
		panic(err)
	}

	matchableEntries := organizeConfEntries(confEntries)

	matchablesText, err := geconf.List(matchableEntries).MarshalText()
	log.Printf("Matchable conf entries (err %v): {%s}\n", err, matchablesText)

	for i := range generated {
		var data sqlTemplateData

		baseName := templates[i].BaseName

		setSpecificSQLTemplateData(&data, matchableEntries, baseName)

		if data.IDTableName != "" {
			data.ReferencesIDTable = " REFERENCES " + data.IDTableName
		}

		data.Prefix = prefix

		generated[i].CreateSQL = generateSQL(templates[i].CreateSQL, data)
		generated[i].ElemType = templates[i].ElemType
		generated[i].BaseName = baseName
		generated[i].Name = prefix + baseName
	}
}

func generateInsertSQLForAllTables(generated, templates []string, prefix string) {
	err := checkSafeNameChars(prefix)
	if err != nil {
		panic(err)
	}
	data := sqlTemplateData{Prefix: prefix}

	for i := range generated {
		generated[i] = generateSQL(templates[i], data)
	}
}

func generateSQL(sqlTemplate string, data sqlTemplateData) string {
	// Template caching does not seem necessary in this case:
	// Usually a given SQL template will be executed/rendered exactly once.
	// Even if needed a second time, it won't be in a performance-critical
	// situation --- could be the case of instantiating two Change Stores
	// using the same 'cstore...' package (same CStore implementation).

	t := template.Must(template.New("").Parse(sqlTemplate))

	var buf bytes.Buffer
	t.Execute(&buf, data)

	return string(bytes.TrimSpace(buf.Bytes()))
}

func setSpecificSQLTemplateData(dest *sqlTemplateData, confEntries []geconf.Entry, elementName string) {
	for _, entry := range confEntries {
		nBytes := len(entry.ConfElement)

		if entry.ConfElement[nBytes-1] == '*' {
			if strings.HasPrefix(elementName, entry.ConfElement[:nBytes-1]) {
				setSQLTemplateDataVal(dest, entry)
			}
		} else {
			if elementName == entry.ConfElement {
				setSQLTemplateDataVal(dest, entry)
			}
		}
	}
}

func setSQLTemplateDataVal(dest *sqlTemplateData, confEntry geconf.Entry) {
	switch confEntry.ConfProperty {
	case "IDTable":
		dest.IDTableName = confEntry.ConfValue
	case "IOTL1":
		dest.IndexOrganizedTableL1 = getConfBool(confEntry.ConfValue, false)
	case "IOTL2":
		dest.IndexOrganizedTableL2 = getConfBool(confEntry.ConfValue, false)
	default:
		log.Printf("Unexpected property in %#v", confEntry)
	}
}

func getConfBool(confValue string, defaultVal bool) bool {
	switch confValue {
	case "Y", "y", "T", "t":
		return true
	case "N", "n", "F", "f":
		return false
	default:
		log.Printf("Returning default (%v) for unexpected bool ConfValue %q",
			defaultVal, confValue)
		return defaultVal
	}
}

func organizeConfEntries(confEntries []geconf.Entry) []geconf.Entry {
	for i := range confEntries {
		rankByElementPrefixLength(&confEntries[i])
	}

	sort.Sort(geconf.RankOrder(confEntries))

	var iSkip int
	for iSkip = range confEntries {
		if confEntries[iSkip].Rank >= 0 {
			break
		}
	}
	return confEntries[iSkip:]
}

func rankByElementPrefixLength(confEntry *geconf.Entry) {
	// Arbitrary limit for configuration element string length in bytes,
	// should be several orders of magnitude smaller than 'math.MaxInt32'.
	const maxBytes = 100

	nBytes := len(confEntry.ConfElement)
	if nBytes > maxBytes {
		// Allow releasing memory in case it was a huge string:
		confEntry.ConfElement = ""

		confEntry.Rank = -99
		return
	}
	if nBytes == 0 {
		confEntry.Rank = -2
		return
	}

	starPos := strings.IndexByte(confEntry.ConfElement, '*')
	if starPos >= 0 { // wildcard (asterisk) found; need to check where:
		if starPos == nBytes-1 { // looks like a well-formed prefix:
			confEntry.Rank = int32(starPos)
		} else { // not a prefix; we only support asterisk at the end:
			confEntry.Rank = -5
		}
	} else { // no wildcard (asterisk); sort all such entries after prefixes:
		confEntry.Rank = math.MaxInt32
	}
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
		// NO: case ch == '.'
		// Could also allow at most one dot but not for SQLite.
		// Maybe would make sense for other SQL DBMS which
		// use schema names / namespaces.
	}
	return nil
}
