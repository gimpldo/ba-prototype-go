package cstoresqlite0

// The trailing 'BN' stands for "Base Name"
// The trailing 'Cre' stands for "Create" (SQL DDL statement)
// The trailing 'Ins' stands for "Insert" (SQL DML statement)
// The trailing 'Sel...' stands for "Select" (SQL DML statement)
// 'crec' stands for "Change Record"
// 'cset' stands for "Change Set" (usually written as a single word: changeset)

// 'edit_op_cid' = Editing Operation identifier, a kind of Correlation ID,
// local to a Changeset (i.e. references are valid only inside a changeset).
//
// The values appearing in the 'edit_op_cid' column will be taken from
// a pool of unique IDs --- reusable in each changeset that contains
// editing operation metadata.
//
// The 'c' in '_cid' can be understood as "Correlation" or "Changeset-scoped".

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

.ReferencesChangeSetIDTable: string

Could be the empty string (no "REFERENCES" clause), or
valid syntax for the clause, including the desired changeset ID table's name
(which need not use same prefix as the other tables, but most likely will ---
it makes little sense to have a different prefix for the main changeset table).
    Example: " REFERENCES " followed by the changeset ID table name
      (note the space before and after "REFERENCES")

Using the same ID table for all IDs (including changeset IDs) is allowed:
  templateData.ReferencesChangeSetIDTable = templateData.ReferencesIDTable
is perfectly OK, even if a suboptimal use of database integrity checks.

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

	ReferToMainChangeSetTable  bool
	ReferencesChangeSetIDTable string

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

// The main changeset table: contains one row for each changeset
//
// Note that the primary key is *not* followed by 'ReferencesChangeSetIDTable'
// because *this* table is intended to *be* the "ChangeSet ID Table"
// (so it would refer to itself).
//
const tableCSetInfoBN = "cset_info"
const tableCSetInfoCre = `CREATE TABLE {{.Prefix}}cset_info (
  cset_id INTEGER PRIMARY KEY NOT NULL{{.ReferencesIDTable}},
  cset_todo_property TEXT NOT NULL,
  cset_todo_value TEXT NOT NULL
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
  cset_id INTEGER NOT NULL{{.ReferencesChangeSetIDTable}},
  subject_id INTEGER NOT NULL{{.ReferencesIDTable}},
  prop_id INTEGER NOT NULL{{.ReferencesIDTable}},
  object_id INTEGER NOT NULL{{.ReferencesIDTable}},
  crec_type INTEGER NOT NULL,
  crec_flags INTEGER NOT NULL,
  crec_context_id INTEGER NOT NULL{{.ReferencesIDTable}},
  edit_op_cid INTEGER NOT NULL{{.ReferencesIDTable}},
  PRIMARY KEY (cset_id, subject_id, prop_id, object_id)
) {{if .IndexOrganizedTableL1}} WITHOUT ROWID {{end}}
`
const tableCRecIDObjIns = `INSERT INTO {{.Prefix}}crec_idobj (
  cset_id, subject_id, pos_cn, old_pos_cn,
  crec_type, crec_flags, crec_context_id, edit_op_cid,
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
  cset_id INTEGER NOT NULL{{.ReferencesChangeSetIDTable}},
  subject_id INTEGER NOT NULL{{.ReferencesIDTable}},
  prop_id INTEGER NOT NULL{{.ReferencesIDTable}},
  lang_tag TEXT NOT NULL,
  string_val TEXT NOT NULL,
  crec_type INTEGER NOT NULL,
  crec_flags INTEGER NOT NULL,
  crec_context_id INTEGER NOT NULL{{.ReferencesIDTable}},
  edit_op_cid INTEGER NOT NULL{{.ReferencesIDTable}},
  PRIMARY KEY (cset_id, subject_id, prop_id, lang_tag, string_val)
) {{if .IndexOrganizedTableL2}} WITHOUT ROWID {{end}}
`
const tableCRecLangStringIns = `INSERT INTO {{.Prefix}}crec_langstring (
  cset_id, subject_id, pos_cn, old_pos_cn,
  crec_type, crec_flags, crec_context_id, edit_op_cid,
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
  cset_id INTEGER NOT NULL{{.ReferencesChangeSetIDTable}},
  subject_id INTEGER NOT NULL{{.ReferencesIDTable}},
  prop_id INTEGER NOT NULL{{.ReferencesIDTable}},
  val_datatype_id INTEGER NOT NULL{{.ReferencesIDTable}},
  string_val TEXT NOT NULL,
  crec_type INTEGER NOT NULL,
  crec_flags INTEGER NOT NULL,
  crec_context_id INTEGER NOT NULL{{.ReferencesIDTable}},
  edit_op_cid INTEGER NOT NULL{{.ReferencesIDTable}},
  PRIMARY KEY (cset_id, subject_id, prop_id, val_datatype_id, string_val)
) {{if .IndexOrganizedTableL2}} WITHOUT ROWID {{end}}
`
const tableCRecLitDatatypeIns = `INSERT INTO {{.Prefix}}crec_litdatatype (
  cset_id, subject_id, pos_cn, old_pos_cn,
  crec_type, crec_flags, crec_context_id, edit_op_cid,
  val_type_id, item_id,
  lang_tag, string_val
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`

const tableCRecBN = "crec_"
const tableCRecCre = `CREATE TABLE {{.Prefix}} (
  cset_id INTEGER NOT NULL{{.ReferencesChangeSetIDTable}},
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
  crec_type, crec_flags, crec_context_id, edit_op_cid,
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
  cset_id INTEGER NOT NULL{{.ReferencesChangeSetIDTable}},
  subject_id INTEGER NOT NULL{{.ReferencesIDTable}},
  pos_cn INTEGER NOT NULL,
  old_pos_cn INTEGER NOT NULL,
  crec_type INTEGER NOT NULL,
  crec_flags INTEGER NOT NULL,
  crec_context_id INTEGER NOT NULL{{.ReferencesIDTable}},
  edit_op_cid INTEGER NOT NULL{{.ReferencesIDTable}},
  val_type_id INTEGER NOT NULL{{.ReferencesIDTable}},
  item_id INTEGER NOT NULL{{.ReferencesIDTable}},
  lang_tag TEXT NOT NULL,
  string_val TEXT NOT NULL,
  PRIMARY KEY (cset_id, subject_id, pos_cn)
) {{if .IndexOrganizedTableL2}} WITHOUT ROWID {{end}}
`
const tableCRecOrdContIns = `INSERT INTO {{.Prefix}}crec_ordcont (
  cset_id, subject_id, pos_cn, old_pos_cn,
  crec_type, crec_flags, crec_context_id, edit_op_cid,
  val_type_id, item_id,
  lang_tag, string_val
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`

// Table for change records for IDentified Literal nodes containing Text data.
//
// The goal is to have an ID corresponding to a potentially big text
// so the "big literal" can be:
//  - shared = referred by multiple triples (in the object position);
//  - annotated, tagged (by triples having it in the subject position);
//  - modified incrementally:
//      a change record does *not* need to always specify the whole value ---
//      can insert/append/prepend, remove, or replace *parts* of the literal.
//
// The node/thing identified by the value in the 'subject_id' column
// is *not* necessarily specialized or restricted to "be" a "big literal":
// the same ID may appear, even in same changeset, in the 'subject_id' column
// of other tables with change records (for example, 'crec_ordcont' ---
// our node may "be" simultaneously an Order-preserving Container).
//
// This is why it would be *misleading* to say that the goal is to
// "provide an ID for a potentially big text": it would be too limiting.
//
// On the other hand, it would make little sense to use the same ID
// for a big text and a big binary = to have for the *long* term
// same 'subject_id' in both 'crec_id_lit_text' and 'crec_id_lit_bin' tables.
// *Short-term* coexistence (example: moving/converting from one to the other)
// seems useful; a changeset could contain records saying:
//  - delete text literal and
//  - insert equivalent data into a new binary literal using the same ID.
//
// Trying to prevent unexpected uses seems complicated and unnecessary.
// There is no clear risk/problem that would require such restrictions.
// TODO: document risks/problems related to this (if any).
//
const tableCRecIDLitTextBN = "crec_id_lit_text"
const tableCRecIDLitTextCre = `CREATE TABLE {{.Prefix}}crec_id_lit_text (
  cset_id INTEGER NOT NULL{{.ReferencesChangeSetIDTable}},
  subject_id INTEGER NOT NULL{{.ReferencesIDTable}},
  pos_cn INTEGER NOT NULL,
  old_pos_cn INTEGER NOT NULL,
  crec_type INTEGER NOT NULL,
  crec_flags INTEGER NOT NULL,
  crec_context_id INTEGER NOT NULL{{.ReferencesIDTable}},
  edit_op_cid INTEGER NOT NULL{{.ReferencesIDTable}},
  val_datatype_id INTEGER NOT NULL{{.ReferencesIDTable}},
  lang_tag TEXT NOT NULL,
  string_val TEXT NOT NULL,
  PRIMARY KEY (cset_id, subject_id, pos_cn)
)
`
const tableCRecIDLitTextIns = `INSERT INTO {{.Prefix}}crec_id_lit_text (
  cset_id, subject_id, pos_cn, old_pos_cn,
  crec_type, crec_flags, crec_context_id, edit_op_cid,
  val_type_id, item_id,
  lang_tag, string_val
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`

// Table for change records for IDentified Literal nodes containing Binary data.
//
// The goal is to have an ID corresponding to a potentially big binary
// so the "big literal" can be:
//  - shared = referred by multiple triples (in the object position);
//  - annotated, tagged (by triples having it in the subject position);
//  - modified incrementally:
//      a change record does *not* need to always specify the whole value ---
//      can insert/append/prepend, remove, or replace *parts* of the literal.
//
// The node/thing identified by the value in the 'subject_id' column
// is *not* necessarily specialized or restricted to "be" a "big literal":
// the same ID may appear, even in same changeset, in the 'subject_id' column
// of other tables with change records (for example, 'crec_ordcont' ---
// our node may "be" simultaneously an Order-preserving Container).
//
// This is why it would be *misleading* to say that the goal is to
// "provide an ID for a potentially big binary": it would be too limiting.
//
// On the other hand, it would make little sense to use the same ID
// for a big text and a big binary = to have for the *long* term
// same 'subject_id' in both 'crec_id_lit_text' and 'crec_id_lit_bin' tables.
// *Short-term* coexistence (example: moving/converting from one to the other)
// seems useful; a changeset could contain records saying:
//  - delete text literal and
//  - insert equivalent data into a new binary literal using the same ID.
//
// Trying to prevent unexpected uses seems complicated and unnecessary.
// There is no clear risk/problem that would require such restrictions.
// TODO: document risks/problems related to this (if any).
//
const tableCRecIDLitBinBN = "crec_id_lit_bin"
const tableCRecIDLitBinCre = `CREATE TABLE {{.Prefix}}crec_id_lit_bin (
  cset_id INTEGER NOT NULL{{.ReferencesChangeSetIDTable}},
  subject_id INTEGER NOT NULL{{.ReferencesIDTable}},
  pos_cn INTEGER NOT NULL,

  crec_type INTEGER NOT NULL,
  crec_flags INTEGER NOT NULL,
  crec_context_id INTEGER NOT NULL{{.ReferencesIDTable}},
  edit_op_cid INTEGER NOT NULL{{.ReferencesIDTable}},

  bin_val BLOB NOT NULL,
  PRIMARY KEY (cset_id, subject_id, pos_cn)
)
`

// TODO: crec_id_lit_bin

const viewAllCRecBN = "all_crec"
const viewAllCRecCre = `CREATE VIEW {{.Prefix}}all_crec AS
    SELECT cset_id, subject_id, prop_id AS prop, 0 AS old_prop,
      crec_type, crec_flags, crec_context_id, edit_op_cid,
      lang_tag, string_val
    FROM {{.Prefix}}crec_idobj
  UNION ALL
    SELECT cset_id, subject_id, prop_id AS prop, 0 AS old_prop,

      crec_type, crec_flags, crec_context_id, edit_op_cid,
      lang_tag, string_val
    FROM {{.Prefix}}crec_langstring
  UNION ALL
    SELECT cset_id, subject_id, prop_id AS prop, 0 AS old_prop,

      crec_type, crec_flags, crec_context_id, edit_op_cid,
      lang_tag, string_val
    FROM {{.Prefix}}crec_litdatatype
  UNION ALL
    SELECT cset_id, subject_id, pos_cn, old_pos_cn,
      val_type_id, item_id,
      crec_type, crec_flags, crec_context_id, edit_op_cid,
      lang_tag, string_val
    FROM {{.Prefix}}crec_ordcont
`
