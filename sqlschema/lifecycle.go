package sqlschema

import (
	"database/sql"
	"fmt"

	"github.com/pkg/errors"
)

// InitOpReportElements initializes a SQL schema operations report,
// especially the slice of schema element status records.
//
// This is not a method of *OpReport to avoid any suggestion/temptation
// to perform changes based on the report info in code that receives a report.
// A schema operations report should be considered read-only outside
// this package and packages implementing interfaces like 'cstore.SQLDef'
// (that this package is intended to help).
//
func InitOpReportElements(r *OpReport, defs []ElementDef) {
	r.NumExpected = len(defs)
	r.Elements = make([]ElementStatus, r.NumExpected)

	for i, elem := range defs {
		statusRec := &r.Elements[i]
		statusRec.ElemType = elem.ElemType
		statusRec.BaseName = elem.BaseName
		statusRec.Name = elem.Name
		statusRec.Status = InitializedES
	}
}

// CreateElements tries to create the SQL schema elements that
// have the given status ('onlyStatus') in the corresponding status record
// from the given schema operations report.
// Stop and return error at first failure.
func CreateElements(db *sql.DB, r *OpReport, defs []ElementDef, onlyStatus ElemStatusCode) error {
	r.NumCreated = 0
	r.NumFailed = 0

	r.LastOp = OpCreate

	for i, elem := range defs {
		statusRec := &r.Elements[i]

		if statusRec.ElemType != elem.ElemType {
			panic(fmt.Sprintf("different ElemType at %d", i))
		}
		if statusRec.BaseName != elem.BaseName {
			panic(fmt.Sprintf("different BaseName at %d", i))
		}
		if statusRec.Name != elem.Name {
			panic(fmt.Sprintf("different Name at %d", i))
		}

		if statusRec.Status != onlyStatus {
			continue
		}

		_, execErr := db.Exec(elem.CreateSQL)
		if execErr == nil {
			statusRec.Err = nil
			statusRec.Status = CreatedES
			r.NumCreated++
		} else {
			err := errors.Wrapf(execErr,
				"failed to create element %d/%d = %s [%s] (status was %s)",
				i, len(defs), elem.ElemType.String(), elem.Name,
				statusRec.Status)

			statusRec.Err = err
			statusRec.Status = CreateFailedES
			r.NumFailed++

			return err
		}
	}

	return nil
}

// DropElement tries to drop the given schema element and
// updates its status record.
func DropElement(db *sql.DB, statusRec *ElementStatus) error {
	dropSQL := fmt.Sprintf("DROP %s %s",
		statusRec.ElemType.SQLKeyword(), statusRec.Name)
	_, execErr := db.Exec(dropSQL)
	if execErr == nil {
		statusRec.Status = DroppedES
	} else {
		statusRec.Status = DropFailedES
	}

	statusRec.Err = execErr
	return execErr
}

// DropReportedElements = try to delete the given schema elements.
//
// Tries to drop the given schema elements in reverse order,
// starting with the last element, which should not be a dependency for others.
// The intention is to allow using same sequence of schema elements for
// creating and dropping:
//  - creation goes from first to last,
//  - dropping starts with the last element.
//
// This is not a method of *OpReport to avoid any suggestion/temptation
// to perform changes based on the report info in code that receives a report.
// A schema operations report should be considered read-only outside
// this package and packages implementing interfaces like 'cstore.SQLDef'
// (that this package is intended to help).
//
func DropReportedElements(db *sql.DB, r *OpReport, tryAll bool) error {
	r.NumDropped = 0
	r.NumFailed = 0

	r.LastOp = OpDrop

	for i := len(r.Elements) - 1; i >= 0; i-- {
		statusRec := &r.Elements[i]

		dropSQL := fmt.Sprintf("DROP %s %s",
			statusRec.ElemType.SQLKeyword(), statusRec.Name)
		_, execErr := db.Exec(dropSQL)
		if execErr == nil {
			statusRec.Err = nil
			statusRec.Status = DroppedES
			r.NumDropped++
		} else {
			err := errors.Wrapf(execErr,
				"failed to drop element %d/%d = %s [%s]",
				i, len(r.Elements),
				statusRec.ElemType.SQLKeyword(), statusRec.Name)

			statusRec.Err = err
			statusRec.Status = DropFailedES
			r.NumFailed++

			if !tryAll {
				return err
			}
		}
	}

	return nil
}
