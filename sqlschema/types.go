package sqlschema

import (
	"bytes"
	"fmt"
	"io"
)

// ElemTypeCode = SQL database schema Element Type Code
type ElemTypeCode int

// Element Type codes
const (
	NoElem ElemTypeCode = iota
	TableElem
	ViewElem
	IndexElem
	TriggerElem
)

// ElemStatusCode = SQL database schema Element Status Code
type ElemStatusCode int

// Element Status codes;
// the trailing 'ES' stands for "Element Status"
const (
	UnknownES ElemStatusCode = iota

	InitializedES

	MissingES
	FoundES
	MatchedES
	MismatchedES

	CreatedES
	CreateFailedES

	DroppedES
	DropFailedES
)

// ElementTemplate = SQL database schema Element Template.
// Schema elements are also known as "schema objects".
type ElementTemplate struct {
	ElemType ElemTypeCode
	BaseName string

	CreateSQL string
}

// ElementDef = SQL database schema Element Definition.
// Schema elements are also known as "schema objects".
type ElementDef struct {
	ElemType ElemTypeCode
	BaseName string
	Name     string

	CreateSQL string
}

// ElementStatus = status record for SQL database schema element (table, etc.)
type ElementStatus struct {
	ElemType ElemTypeCode
	BaseName string
	Name     string

	Status        ElemStatusCode
	ProblemDetail string

	Err error
}

// Suggested values for OpReport.LastOp
const (
	OpCheckFailDB = "check failed to get DB schema"

	OpCheck  = "check"
	OpCreate = "create"
	OpDrop   = "drop"
)

// OpReport = database (SQL) schema operations report,
// to be returned by functions that implement mass operations like
// checking, creating or dropping a collection of SQL schema elements:
// tables, etc.
type OpReport struct {
	Elements []ElementStatus

	Conf string

	// Last Operation that contributed to this report;
	// its value should be a short lowercase verb (especially
	// in case of success; failure can and maybe should look ugly).
	//
	// Suggested values: "check", "create", "drop"; there are constants
	// defined for this purpose (OpCheck, OpCreate, OpDrop).
	//
	LastOp string

	NumExpected   int
	NumFound      int
	NumMatched    int
	NumMismatched int
	NumMissing    int
	NumCreated    int
	NumDropped    int
	NumFailed     int
}

// String method is for display and debugging purpose
func (r OpReport) String() string {
	var buf bytes.Buffer
	r.Dump(&buf, 0)
	return buf.String()
}

// Dump method is for display and debugging purpose
func (r OpReport) Dump(w io.Writer, detailLevel int) {
	if r.LastOp != "" {
		fmt.Fprintf(w, "After %s:", r.LastOp)
	}

	fmt.Fprintf(w, " %d expected", r.NumExpected)

	if r.NumFound != 0 {
		fmt.Fprintf(w, ", %d found", r.NumFound)
	}
	if r.NumMatched != 0 {
		fmt.Fprintf(w, ", %d matched", r.NumMatched)
	}
	if r.NumMismatched != 0 {
		fmt.Fprintf(w, ", %d mismatched", r.NumMismatched)
	}
	if r.NumMissing != 0 {
		fmt.Fprintf(w, ", %d missing", r.NumMissing)
	}
	if r.NumCreated != 0 {
		fmt.Fprintf(w, ", %d created", r.NumCreated)
	}
	if r.NumDropped != 0 {
		fmt.Fprintf(w, ", %d dropped", r.NumDropped)
	}
	if r.NumFailed != 0 {
		fmt.Fprintf(w, ", %d failed", r.NumFailed)
	}

	nElems := len(r.Elements)
	if nElems != 0 {
		fmt.Fprintf(w, ":")
		for i, elem := range r.Elements {
			fmt.Fprintf(w, "\n[%d/%d] ", i, nElems)
			elem.Dump(w, detailLevel)
		}
	} else {
		fmt.Fprintf(w, ".")
	}

	if detailLevel > 1 {
		fmt.Fprintf(w, "\nConf: {%s}\n", r.Conf)
	}
}

// String method is for display and debugging purpose
func (es ElementStatus) String() string {
	var buf bytes.Buffer
	es.Dump(&buf, 0)
	return buf.String()
}

// Dump method is for display and debugging purpose
func (es ElementStatus) Dump(w io.Writer, detailLevel int) {
	fmt.Fprintf(w, "%s: %q = %s %q",
		es.Status.String(), es.BaseName, es.ElemType.String(), es.Name)

	if es.ProblemDetail != "" {
		fmt.Fprintf(w, " {%s}", es.ProblemDetail)
	}

	if es.Err != nil {
		if detailLevel > 1 {
			fmt.Fprintf(w, ": %#+v", es.Err)
		} else {
			fmt.Fprintf(w, ": %#v", es.Err)
		}
	} else {
		fmt.Fprintf(w, ".")
	}
}

// String method is for display and debugging purpose
func (statusCode ElemStatusCode) String() string {
	switch statusCode {
	case UnknownES:
		return "(unknown status)"
	case InitializedES:
		return "Initialized"
	case MissingES:
		return "Missing"
	case FoundES:
		return "Found"
	case MatchedES:
		return "Matched"
	case MismatchedES:
		return "Mismatched"
	case CreatedES:
		return "Created"
	case CreateFailedES:
		return "Create failed"
	case DroppedES:
		return "Dropped"
	case DropFailedES:
		return "Drop failed"
	default:
		return fmt.Sprintf("(unknown schema element status code %x)",
			uint(statusCode))
	}
}

// String method is for display and debugging purpose
func (typeCode ElemTypeCode) String() string {
	switch typeCode {
	case NoElem:
		return "(no schema element)"
	case TableElem:
		return "table"
	case ViewElem:
		return "view"
	case IndexElem:
		return "index"
	default:
		return fmt.Sprintf("(unknown schema element type code %x)",
			uint(typeCode))
	}
}

// SQLKeyword method is for using the element type in generated SQL
func (typeCode ElemTypeCode) SQLKeyword() string {
	switch typeCode {
	case NoElem:
		panic("No schema element")
	case TableElem:
		return "TABLE"
	case ViewElem:
		return "VIEW"
	case IndexElem:
		return "INDEX"
	default:
		panic(fmt.Sprintf("Unknown schema element type code (%x)",
			uint(typeCode)))
	}
}
