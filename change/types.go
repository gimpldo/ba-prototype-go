package change

import (
	"github.com/gimpldo/ba-prototype-go/id"
)

// RecForm = change Record Form / Format; it's about type of content,
// not about the kind of change (operation: add, delete, modify, etc.)
type RecForm uint16

const (
	// UsualTriple means typical RDF triple,
	// not part of an order-preserving container:
	UsualTriple RecForm = 0

	// OrdContItem means Item from an Order-preserving Container:
	OrdContItem = 8
)

// RecType = change Record Type.
// Usually represents one editing operation: add, delete, modify, etc.
// The name emphasizes 'Record' type (it's not simply a 'Change' type) because
// (1) some records might not specify a change (operation), but
// context info or metadata;
// (2) a logical change (replace/modify, for example) might be represented by
// at least two records (Add and Delete) and maybe some context info or metadata records.
//
// The change record flags and the change record context ID might be useful
// for connecting related change records like the Add and Delete
// from the above example, but they cannot guarantee unambiguous connection;
// the old value fields (when available) are more useful for that.
//
// It is not a goal to provide unambiguous identification of
// logical / high-level editing intentions in all cases.
//
type RecType uint16

// The trailing 'RT' in constant names stands for "Record Type".
const (
	AddRT RecType = 0
	DelRT         = 1

	ModifyRT // or have it only for order-preserving container items? see below!

	TestRT
	ContextRT

	// 'Ident' is short for "Identify", or "Identifier";
	// initially considered the name 'DescRT' ("Describe")
	IdentRT

	// 'Meta' is short for "Add Metadata".
	// There is no corresponding "Delete Metadata" because
	// the metadata records (which are completely optional anyway)
	// are intended to help interpreting the changeset that contains them.
	// A metadata record ---by itself--- does NOT
	// (1) provide a value to be stored or (2) cause a change
	// in the data store that receives and applies the changeset.
	MetaRT

	// The following type codes are for change record Form = OrdContItem
	// and maybe other forms not specified yet, like big text or binary.
	//
	// They apply only to items from an order-preserving container or
	// some sequence types like a big text or binary which may be defined
	// later, even if they would not correspond to anything in RDF.

	AppendRT
	PrependRT
	InsertAtRT
	RemoveAtRT
	ReplaceAtRT // initially considered the name 'SetItemRT'

// Or only have the above 'ReplaceAtRT' instead of a general 'ModifyRT'?
// Is a general "Modify" operation meaningful in the world of RDF?
)

// Change record flag values, for 'ChangeRecFlags' below.
// The trailing 'RF' in constant names stands for "Record Flag".
const (
	// Copying Flag means: the change record is part of the representation
	// for a copy operation/edit, which should include at least one (1)
	// 'AddRT', 'AppendRT', 'PrependRT', 'InsertAtRT' or 'ReplaceAtRT' record.
	// There may be a 'TestRT' record to document the source for the copy;
	// it must have this flag set.
	// TODO: document ways to identify corresponding change records
	// that represent together a copy operation/edit.
	CopyingRF = 0x1

	// Reorder = move an item inside an order-preserving container or
	// maybe some sequence type like a big text or binary
	// (if such extensions are defined later).
	// Reordering Flag means: the change record is part of the representation
	// for a reordering operation/edit, which should include
	// one 'RemoveAtRT' and one 'InsertAtRT' record for same subject
	// (order-preserving container), with this flag set on both records.
	ReorderingRF = 0x1

	// Swapping Flag means: the change record is part of the representation
	// for an interchange operation/edit, which should include
	// two 'ReplaceAtRT' records for same subject
	// (order-preserving container), with this flag set on both records.
	// TODO: maybe name it 'SwapItemsRF' ???
	// Does this make it too specific? That is, only for the OrdCont, and
	// not usable for possible sequence type like a big text or binary
	// (if such extensions are defined later).
	// *** Would it be wrong to have flags specific to OrdCont?
	// *** Can't tell now --- February 15th, 2017
	SwappingRF = 0x1

	// ???

	ReplacingRF = 0x1
	MovingRF    = 0x1
	RF          = 0x1
)

// Rec = change Record
type Rec struct {
	Form           RecForm
	ChangeRecType  RecType
	ChangeRecFlags uint32

	ChangeRecContextID id.IntID

	EditOpID id.IntID

	ValueTypeID id.IntID

	ChangeSetID id.IntID

	SubjectID id.IntID
	Prop      id.PosOrID
	ObjectID  id.IntID

	LangTag   string
	StringVal string
}

// RecPullSource = Pull Source of change Records = iterator
type RecPullSource interface {
	// Another good name would be TakeChangeRec().
	//
	// Could also return only error in second position, no bool ('gotRec'),
	// but this would require a sentinel error like 'io.EOF'.
	// None of the ideas looks great; I don't like much the below either:
	//
	GetNextChangeRec() (rec Rec, gotRec bool, err error)
}

// RecPullSourceCloser = Pull Source of change Records + the ability to dispose/finish
type RecPullSourceCloser interface {
	RecPullSource

	Close() error
}

// RecPushSink = Push Sink of change Records
type RecPushSink interface {
	PutChangeRec(Rec) error
}

// RecPushSinkEnder = Push Sink of change Records + the ability to dispose/finish,
// ending with commit or abort --- so a Close() method does not fit well;
// it's not exactly the 'Closer' kind of interface
type RecPushSinkEnder interface {
	RecPushSink

	// We don't need or expect an error return. TODO: is this suitable?
	End()   // Commit? EndCommit?
	Abort() // EndAbort?
}

// Set = changeset
type Set struct {
	// A slice of change records; order should not be relevant
	// (reordering the change records should not change the meaning =
	// result of applying the changeset).
	// The slice of change records need not be allocated;
	// the other fields of the changeset may be valid while
	// the change records may be streamed, not stored
	// (especially when there are too many).
	//
	ChangeRecords []Rec
}
