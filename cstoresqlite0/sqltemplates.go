package cstoresqlite0

// The trailing 'BN' stands for "Base Name"
// The trailing 'Cre' stands for "Create" (SQL DDL statement)
// The trailing 'Ins' stands for "Insert" (SQL DML statement)
// The trailing 'Sel...' stands for "Select" (SQL DML statement)
// 'crec' stands for "Change Record"
// 'cset' stands for "Change Set" (usually written as a single word: changeset)

/*
The SQL DDL and DML templates defined below are intended to be used
with the standard Go package 'text/template'.

Arguments for the SQL templates defined below:

.Prefix: string

Prefix for schema element names (tables, views, etc.).
Could be the beginning of an identifier: letter followed by letters or digits.
    Example: "cst123_"
      (the trailing underscore is just a suggestion, for readability)
Could be also a schema or database name, followed by dot --- must be explicit,
because the templates try to accomodate any possible syntax so they will just
prepend the given prefix to the base name.
    Example: "dbname."
      (the trailing dot is required by syntax)
You could have both schema/database name and a local name prefix.
    Example: "dbname.cst123_"

.ReferencesIDTable: string

Could be the empty string (no "REFERENCES" clause), or
valid syntax for the clause, including the desired ID table's name
(which need not use same prefix as the other tables).
    Example: " REFERENCES " followed by the ID table name
      (note the space before and after "REFERENCES")

.IndexOrganizedTableL1: boolean

If true, create Index-Organized Table (also known as "clustered index")
where applicable = where this option is mentioned.
This is specified in SQLite3 by appending "WITHOUT ROWID"
to the "CREATE TABLE" statement.
This template argument (.IndexOrganizedTableL1) is for the "Level one" of
using index-organized tables = where it seems to be a good fit (clear benefit
and no obvious problem, nothing to recommend against using IOTs).

.IndexOrganizedTableL2: boolean

If true, create Index-Organized Tables (also known as "clustered indexes")
where applicable = where this option is mentioned.
This is specified in SQLite3 by appending "WITHOUT ROWID"
to the "CREATE TABLE" statement.
This template argument (.IndexOrganizedTableL2) is for the "Level two" of
using index-organized tables: cases which might benefit or not from it
(there are possible problems as well as possible benefits).

*/
type sqlTemplateData struct {
	Prefix string

	IDTableName       string
	ReferencesIDTable string

	IndexOrganizedTableL1 bool
	IndexOrganizedTableL2 bool
}

// The head table: by checking it we can say whether we got a valid CStore;
// it contains the definition options used when the CStore was created.
//
// This is an index-organized table ("WITHOUT ROWID" in SQLite3) mainly
// to make sure that the database library used supports "WITHOUT ROWID" =
// it's not a very old SQLite3 version.
//
// The expected benefit of reduced storage requirements
// by avoiding a separate index for the primary key
// is not relevant/significant in this case because
// this table should be very small anyway (expected average size:
// less than 10 rows, with very short strings in the three columns).
//
const tableCStoreConfBN = "cstore_conf"
const tableCStoreConfCre = `CREATE TABLE {{.Prefix}}cstore_conf (
  cstore_element TEXT NOT NULL,
  cstore_property TEXT NOT NULL,
  cstore_value TEXT NOT NULL,
  PRIMARY KEY (cstore_element, cstore_property)
) WITHOUT ROWID
`
const tableCStoreConfIns = `INSERT INTO {{.Prefix}}cstore_conf (
  cstore_element, cstore_property, cstore_value
) VALUES (?, ?, ?)
`

// Main changeset table: one row for each changeset
//
const tableCSetInfoBN = "cset_info"
const tableCSetInfoCre = `CREATE TABLE {{.Prefix}}cset_info (
  cset_id INTEGER NOT NULL{{.ReferencesIDTable}},
  cset_todo_property TEXT NOT NULL,
  cset_todo_value TEXT NOT NULL,
  PRIMARY KEY (cset_id)
)
`
const tableCSetInfoIns = `INSERT INTO {{.Prefix}}cset_info (
  cset_id, cset_todo_property, cset_todo_value
) VALUES (?, ?, ?)
`

const tableBN = ""
const tableCre = `CREATE TABLE {{.Prefix}} (

)
`

// Table for change records with object specified as ID ("resource" TODO)
//
// It's on "Level one" of using index-organized tables =
// seems to be a good fit.
// Expected benefit of IOTs: reduced storage requirements
// by avoiding a separate index for the primary key.
//
const tableCRecIDObjBN = "crec_idobj"
const tableCRecIDObjCre = `CREATE TABLE {{.Prefix}}crec_idobj (
  cset_id INTEGER NOT NULL{{.ReferencesIDTable}},
  subject_id INTEGER NOT NULL{{.ReferencesIDTable}},
  prop_id INTEGER NOT NULL{{.ReferencesIDTable}},
  object_id INTEGER NOT NULL{{.ReferencesIDTable}},
  crec_type INTEGER NOT NULL,
  crec_flags INTEGER NOT NULL,
  crec_context_id INTEGER NOT NULL{{.ReferencesIDTable}},
  edit_op_id INTEGER NOT NULL{{.ReferencesIDTable}},
  PRIMARY KEY (cset_id, subject_id, prop_id, object_id)
) {{if .IndexOrganizedTableL1}} WITHOUT ROWID {{end}}
`
const tableCRecIDObjIns = `INSERT INTO {{.Prefix}}crec_idobj (
  cset_id, subject_id, pos_cn, old_pos_cn,
  crec_type, crec_flags, crec_context_id, edit_op_id,
  val_type_id, item_id,
  lang_tag, string_val
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`

// Table for change records with value = Language-tagged String
// (class 'rdf:langString').
//
// "The class rdf:langString is the class of language-tagged string values.
//  rdf:langString is
//        an instance of rdfs:Datatype and
//        a subclass of rdfs:Literal."
// == Quote from RDF Schema 1.1, W3C Recommendation 25 February 2014
//    https://www.w3.org/TR/rdf-schema/#ch_langstring
//
// "RDF also defines rdf:langString, used for plain text in a natural language,
//  but this is not formally considered a datatype."
// == Quote from RDF 1.1 Concepts and Abstract Syntax,
//        W3C Editor's Draft 20 March 2017
//    https://dvcs.w3.org/hg/rdf/raw-file/default/diffs/rdf-concepts-langstring.html#section-Datatypes-intro
//
// Explanation for the above "[langString] not formally considered a datatype":
// "NOTE: Language-tagged strings have the datatype IRI rdf:langString.
//  No datatype is formally defined for this IRI because
//  the definition of datatypes does not accommodate language tags."
// == Quote from Richard Cyganiak's email (Tue, 13 Sep 2011 06:03:53 +0200)
//        Message-Id: <7C779724-4593-4FA7-A90A-1BAD1046EC0B@cyganiak.de>
//        Subject: Proposed text for language-tagged strings (ISSUE-71)
//    http://lists.w3.org/Archives/Public/public-rdf-wg/2011Sep/0083.html
//
// It's on "Level two" of using index-organized tables:
// (1) Possible problem: row size has no specified limit ---
//      contains string field which can be arbitrarily big
//      ('string_val'); 'lang_tag' is less likely to cause problems
//      (big 'lang_tag' value would be a bug).
// (2) Possible benefits of IOTs: reduced storage requirements
//      by avoiding a separate index for the primary key (relevant because
//      rows are usually small --- expected average size under 100 bytes)
//
const tableCRecLangStringBN = "crec_langstring"
const tableCRecLangStringCre = `CREATE TABLE {{.Prefix}}crec_langstring (
  cset_id INTEGER NOT NULL{{.ReferencesIDTable}},
  subject_id INTEGER NOT NULL{{.ReferencesIDTable}},
  prop_id INTEGER NOT NULL{{.ReferencesIDTable}},
  lang_tag TEXT NOT NULL,
  string_val TEXT NOT NULL,
  crec_type INTEGER NOT NULL,
  crec_flags INTEGER NOT NULL,
  crec_context_id INTEGER NOT NULL{{.ReferencesIDTable}},
  edit_op_id INTEGER NOT NULL{{.ReferencesIDTable}},
  PRIMARY KEY (cset_id, subject_id, prop_id, lang_tag, string_val)
) {{if .IndexOrganizedTableL2}} WITHOUT ROWID {{end}}
`
const tableCRecLangStringIns = `INSERT INTO {{.Prefix}}crec_langstring (
  cset_id, subject_id, pos_cn, old_pos_cn,
  crec_type, crec_flags, crec_context_id, edit_op_id,
  val_type_id, item_id,
  lang_tag, string_val
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`

// Table for change records with value = Literal with Datatype,
// other than Language-tagged String ('rdf:langString', see above
// the langString specific table)
//
// It's on "Level two" of using index-organized tables:
// (1) Possible problem: row size has no specified limit
//      (contains string field which can be arbitrarily big)
// (2) Possible benefits of IOTs: reduced storage requirements
//      by avoiding a separate index for the primary key (relevant because
//      rows are usually small --- expected average size under 100 bytes)
//
const tableCRecLitDatatypeBN = "crec_litdatatype"
const tableCRecLitDatatypeCre = `CREATE TABLE {{.Prefix}}crec_litdatatype (
  cset_id INTEGER NOT NULL{{.ReferencesIDTable}},
  subject_id INTEGER NOT NULL{{.ReferencesIDTable}},
  prop_id INTEGER NOT NULL{{.ReferencesIDTable}},
  val_datatype_id INTEGER NOT NULL{{.ReferencesIDTable}},
  string_val TEXT NOT NULL,
  crec_type INTEGER NOT NULL,
  crec_flags INTEGER NOT NULL,
  crec_context_id INTEGER NOT NULL{{.ReferencesIDTable}},
  edit_op_id INTEGER NOT NULL{{.ReferencesIDTable}},
  PRIMARY KEY (cset_id, subject_id, prop_id, val_datatype_id, string_val)
) {{if .IndexOrganizedTableL2}} WITHOUT ROWID {{end}}
`
const tableCRecLitDatatypeIns = `INSERT INTO {{.Prefix}}crec_litdatatype (
  cset_id, subject_id, pos_cn, old_pos_cn,
  crec_type, crec_flags, crec_context_id, edit_op_id,
  val_type_id, item_id,
  lang_tag, string_val
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`

const tableCRecBN = "crec_"
const tableCRecCre = `CREATE TABLE {{.Prefix}} (
  cset_id INTEGER NOT NULL{{.ReferencesIDTable}},
  subject_id INTEGER NOT NULL{{.ReferencesIDTable}},
  prop_id INTEGER NOT NULL{{.ReferencesIDTable}},
  object_id INTEGER NOT NULL{{.ReferencesIDTable}},
  crec_type INTEGER NOT NULL,
  crec_flags INTEGER NOT NULL,
  crec_context_id INTEGER NOT NULL{{.ReferencesIDTable}},
  val_type_id INTEGER NOT NULL{{.ReferencesIDTable}},
  lang_tag TEXT NOT NULL,
  string_val TEXT NOT NULL,
  PRIMARY KEY (cset_id, subject_id, prop_id)
)
`
const tableCRecIns = `INSERT INTO {{.Prefix}}(
  cset_id, subject_id,
  crec_type, crec_flags, crec_context_id, edit_op_id,
  val_type_id, item_id,
  lang_tag, string_val
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`

// 'ordcont' stands for "Order-preserving Container";
// the trailing '_cn' in field names stands for "Checked Number".
//
// If 'val_type_id' column's value is zero = id.NoID, then
// the 'item_id' column contains the Object part of the triple
// (corresponds to 'object_id' from 'crec_idobj' table).
// The 'lang_tag' and 'string_val' columns could then be used to specify
// a "c-link name" or "label" (this is an extension to RDF containers that
// the 'ba' design included before deciding to follow RDF closely).
//
// If the 'val_type_id' column contains the special ID corresponding to
// 'rdf:langString', the Object/value part is a Language-tagged String:
// the value columns should have same names as in the 'crec_langstring' table
// ('lang_tag' and 'string_val').
// In this case, the 'item_id' column must contain zero (id.NoID).
//
// Otherwise, the row contains a Literal with Datatype:
// 'val_type_id' corresponds to 'val_datatype_id' from 'crec_litdatatype'.
// In this case, the 'item_id' column must contain zero (id.NoID).
//
// It's on "Level two" of using index-organized tables:
// (1) Possible problem: row size has no specified limit ---
//      contains string field which can be arbitrarily big
//      ('string_val'); 'lang_tag' is less likely to cause problems
//      (big 'lang_tag' value would be a bug).
// (2) Possible benefits of IOTs: reduced storage requirements
//      by avoiding a separate index for the primary key (relevant because
//      rows are usually small --- expected average size under 100 bytes)
//
const tableCRecOrdContBN = "crec_ordcont"
const tableCRecOrdContCre = `CREATE TABLE {{.Prefix}}crec_ordcont (
  cset_id INTEGER NOT NULL{{.ReferencesIDTable}},
  subject_id INTEGER NOT NULL{{.ReferencesIDTable}},
  pos_cn INTEGER NOT NULL,
  old_pos_cn INTEGER NOT NULL,
  crec_type INTEGER NOT NULL,
  crec_flags INTEGER NOT NULL,
  crec_context_id INTEGER NOT NULL{{.ReferencesIDTable}},
  edit_op_id INTEGER NOT NULL{{.ReferencesIDTable}},
  val_type_id INTEGER NOT NULL{{.ReferencesIDTable}},
  item_id INTEGER NOT NULL{{.ReferencesIDTable}},
  lang_tag TEXT NOT NULL,
  string_val TEXT NOT NULL,
  PRIMARY KEY (cset_id, subject_id, pos_cn)
) {{if .IndexOrganizedTableL2}} WITHOUT ROWID {{end}}
`
const tableCRecOrdContIns = `INSERT INTO {{.Prefix}}crec_ordcont (
  cset_id, subject_id, pos_cn, old_pos_cn,
  crec_type, crec_flags, crec_context_id, edit_op_id,
  val_type_id, item_id,
  lang_tag, string_val
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`

const viewAllCRecBN = "all_crec"
const viewAllCRecCre = `CREATE VIEW {{.Prefix}}all_crec AS
    SELECT cset_id, subject_id, prop_id AS prop, 0 AS old_prop,
      crec_type, crec_flags, crec_context_id, edit_op_id,
      lang_tag, string_val
    FROM {{.Prefix}}crec_idobj
  UNION ALL
    SELECT cset_id, subject_id, prop_id AS prop, 0 AS old_prop,

      crec_type, crec_flags, crec_context_id, edit_op_id,
      lang_tag, string_val
    FROM {{.Prefix}}crec_langstring
  UNION ALL
    SELECT cset_id, subject_id, prop_id AS prop, 0 AS old_prop,

      crec_type, crec_flags, crec_context_id, edit_op_id,
      lang_tag, string_val
    FROM {{.Prefix}}crec_litdatatype
  UNION ALL
    SELECT cset_id, subject_id, pos_cn, old_pos_cn,
      val_type_id, item_id,
      crec_type, crec_flags, crec_context_id, edit_op_id,
      lang_tag, string_val
    FROM {{.Prefix}}crec_ordcont
`
