package cstoresqlite0

import (
	"github.com/gimpldo/ba-prototype-go/sqlschema"
)

// Constants intended to be used in this package's code only; they are
// a kind of local identifiers for database tables and other schema elements.
//
// Most important use: position in array/slice, so we can define collections
// of tables or related things like statement text, prepared statement objects
// for convenient access to individuals and for collective processing.
//
// (1) Access to individuals = get or set item in an array/slice of objects.
//        You want the object instance that corresponds to a specific table:
//        the table is identified by its position in the array/slice.
//
// (2) Collective processing = visit all items
//        (corresponding to the tables and maybe other schema elements).
//      Example: close all prepared statements that have been created.
//        In this case an array of Stmt objects would be used as a simple cache
//        to allow reusing prepared Stmts for the duration of a transaction.
//
// The trailing 'EI' stands for "Element Index"
// (tables, views, etc. are database schema elements)
//
// Element Index constants in this const block should be
// in the order the respective elements should be created
// (taking references / dependencies into account).
//
// This should be the only place where the order is specified!
// Array entries are filled by index (the full 'KeyedElement' syntax),
// not by order, therefore the initialized arrays will use automatically
// the same order as this enumeration.
//
const (
	tableCStoreConfEI = iota
	tableCSetInfoEI
	tableCRecIDObjEI
	tableCRecLangStringEI
	tableCRecLitDatatypeEI
	tableCRecOrdContEI
	viewAllCRecEI
	nElements // must be last ConstSpec in the const block
)

// The number of tables; to get its value in a maintainable way we could
// use from the above const block
// (1) the Element Index for the last table (adding one to it), or
// (2) the constant immediately after the last table (first view element),
//      in this case without '+ 1'.
const nTables = tableCRecOrdContEI + 1

var elementTemplates = [nElements]sqlschema.ElementTemplate{
	tableCStoreConfEI: {ElemType: sqlschema.TableElem,
		BaseName:  tableCStoreConfBN,
		CreateSQL: tableCStoreConfCre,
	},
	tableCSetInfoEI: {ElemType: sqlschema.TableElem,
		BaseName:  tableCSetInfoBN,
		CreateSQL: tableCSetInfoCre,
	},
	tableCRecIDObjEI: {ElemType: sqlschema.TableElem,
		BaseName:  tableCRecIDObjBN,
		CreateSQL: tableCRecIDObjCre,
	},
	tableCRecLangStringEI: {ElemType: sqlschema.TableElem,
		BaseName:  tableCRecLangStringBN,
		CreateSQL: tableCRecLangStringCre,
	},
	tableCRecLitDatatypeEI: {ElemType: sqlschema.TableElem,
		BaseName:  tableCRecLitDatatypeBN,
		CreateSQL: tableCRecLitDatatypeCre,
	},
	tableCRecOrdContEI: {ElemType: sqlschema.TableElem,
		BaseName:  tableCRecOrdContBN,
		CreateSQL: tableCRecOrdContCre,
	},
	viewAllCRecEI: {ElemType: sqlschema.ViewElem,
		BaseName:  viewAllCRecBN,
		CreateSQL: viewAllCRecCre,
	},
}

var insertSQLTemplates = [nTables]string{
	tableCStoreConfEI:      tableCStoreConfIns,
	tableCSetInfoEI:        tableCSetInfoIns,
	tableCRecIDObjEI:       tableCRecIDObjIns,
	tableCRecLangStringEI:  tableCRecLangStringIns,
	tableCRecLitDatatypeEI: tableCRecLitDatatypeIns,
	tableCRecOrdContEI:     tableCRecOrdContIns,
}
