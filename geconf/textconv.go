package geconf

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

type EntryTextError struct {
	BadEntry       Entry
	OffenderDescr  string
	FieldName      string
	BytePosInField int
}

const noPos = -99999999

func (ete *EntryTextError) Error() string {
	if ete.BytePosInField == noPos {
		return fmt.Sprintf("%s in %s: %#v",
			ete.OffenderDescr, ete.FieldName, ete.BadEntry)
	}
	return fmt.Sprintf("%s in %s at %d: %#v",
		ete.OffenderDescr, ete.FieldName, ete.BytePosInField, ete.BadEntry)
}

func newEntryTextError(badEntry Entry, bytePosInField int, offenderDescr, fieldName string) error {
	return &EntryTextError{
		BadEntry:       badEntry,
		OffenderDescr:  offenderDescr,
		FieldName:      fieldName,
		BytePosInField: bytePosInField,
	}
}

func newEntryTextErrorNoPos(badEntry Entry, offenderDescr, fieldName string) error {
	return &EntryTextError{
		BadEntry:       badEntry,
		OffenderDescr:  offenderDescr,
		FieldName:      fieldName,
		BytePosInField: noPos,
	}
}

func newEntryTextErrorSpaceAround(badEntry Entry, fieldName string) error {
	return &EntryTextError{
		BadEntry:       badEntry,
		OffenderDescr:  "Leading or trailing space",
		FieldName:      fieldName,
		BytePosInField: noPos,
	}
}

func indexNotPrintable(s string) int {
	for i, r := range s {
		if r < 32 || r == 127 {
			return i
		}
	}
	return -1
}

func indexNotPrintableASCII(s string) int {
	for i, r := range s {
		if r < 32 || r > 126 {
			return i
		}
	}
	return -1
}

const entrySeparatorChar = ';'

// TODO-maybe: the const below could be 'var BadChars',
// settable by the code using this package:
const badChars = "\\'[]{}()"

func (ce Entry) MarshalText() (text []byte, err error) {
	var buf bytes.Buffer

	// The leading 'nosp' in the following local variable names
	// stands for "No Space" = string trimmed at both ends.

	nospElement := strings.TrimSpace(ce.ConfElement)
	if nospElement != ce.ConfElement {
		// could ignore this case, it's not essential to return error
		return nil, newEntryTextErrorSpaceAround(ce, "ConfElement")
	}

	if nospElement != "" {
		if pos := strings.IndexByte(ce.ConfElement, entrySeparatorChar); pos >= 0 {
			return nil, newEntryTextError(ce, pos, "Entry separator", "ConfElement")
		}
		if pos := strings.IndexByte(ce.ConfElement, '='); pos >= 0 {
			return nil, newEntryTextError(ce, pos, "Equals sign", "ConfElement")
		}
		if pos := indexNotPrintableASCII(ce.ConfElement); pos >= 0 {
			return nil, newEntryTextError(ce, pos, "Non-ASCII or non-printable char", "ConfElement")
		}
		if pos := strings.IndexAny(ce.ConfElement, badChars); pos >= 0 {
			// could ignore this case, it's not essential to return error
			return nil, newEntryTextError(ce, pos, "Unsupported char", "ConfElement")
		}

		buf.WriteString(nospElement)
		buf.WriteByte('.')
	}

	nospProperty := strings.TrimSpace(ce.ConfProperty)
	if nospProperty != ce.ConfProperty {
		// could ignore this case, it's not essential to return error
		return nil, newEntryTextErrorSpaceAround(ce, "ConfProperty")
	}

	if nospProperty == "" {
		return nil, errors.Errorf("Empty ConfProperty: %#v", ce)
	}
	if pos := strings.IndexByte(ce.ConfProperty, entrySeparatorChar); pos >= 0 {
		return nil, newEntryTextError(ce, pos, "Entry separator", "ConfProperty")
	}
	if pos := strings.IndexByte(ce.ConfProperty, '='); pos >= 0 {
		return nil, newEntryTextError(ce, pos, "Equals sign", "ConfProperty")
	}
	if pos := strings.IndexByte(ce.ConfProperty, '.'); pos >= 0 {
		return nil, newEntryTextError(ce, pos, "Dot", "ConfProperty")
	}
	if pos := indexNotPrintableASCII(ce.ConfProperty); pos >= 0 {
		return nil, newEntryTextError(ce, pos, "Non-ASCII or non-printable char", "ConfProperty")
	}
	if pos := strings.IndexAny(ce.ConfProperty, badChars); pos >= 0 {
		// could ignore this case, it's not essential to return error
		return nil, newEntryTextError(ce, pos, "Unsupported char", "ConfProperty")
	}

	buf.WriteString(nospProperty)
	buf.WriteString(" = ")

	nospValue := strings.TrimSpace(ce.ConfValue)
	if nospValue != ce.ConfValue {
		// could ignore this case, it's not essential to return error
		return nil, newEntryTextErrorSpaceAround(ce, "ConfValue")
	}

	if nospValue == "" {
		return nil, errors.Errorf("Empty ConfValue: %#v", ce)
	}
	if pos := strings.IndexByte(ce.ConfValue, entrySeparatorChar); pos >= 0 {
		return nil, newEntryTextError(ce, pos, "Entry separator", "ConfValue")
	}
	if pos := indexNotPrintable(ce.ConfValue); pos >= 0 {
		return nil, newEntryTextError(ce, pos, "Non-printable char", "ConfValue")
	}
	if pos := strings.IndexAny(ce.ConfValue, badChars); pos >= 0 {
		// could ignore this case, it's not essential to return error
		return nil, newEntryTextError(ce, pos, "Unsupported char", "ConfValue")
	}

	buf.WriteString(nospValue)

	return buf.Bytes(), nil
}

func (ce *Entry) UnmarshalText(text []byte) error {
	eqPos := bytes.IndexByte(text, '=')
	if eqPos < 0 {
		return errors.Errorf("Equals sign not found")
	}
	if eqPos == 0 {
		return errors.Errorf("No text before equals sign")
	}

	ce.ConfValue = string(bytes.TrimSpace(text[eqPos+1:]))

	beforeEq := text[:eqPos]

	dotPos := bytes.LastIndexByte(beforeEq, '.')
	if dotPos < 0 {
		ce.ConfElement = ""
		ce.ConfProperty = string(bytes.TrimSpace(beforeEq))
	} else {
		ce.ConfElement = string(bytes.TrimSpace(beforeEq[:dotPos]))
		ce.ConfProperty = string(bytes.TrimSpace(beforeEq[dotPos+1:]))
	}

	return nil
}

func (clist List) MarshalText() (text []byte, err error) {
	fragments := make([][]byte, 0, len(clist))
	for i, entry := range clist {
		marshalFrag, marshalErr := entry.MarshalText()
		if marshalErr != nil {
			err = errors.Wrapf(marshalErr,
				"failed to marshal entry %d/%d = %v",
				i, len(clist), entry)
			break
		}
		if len(marshalFrag) != 0 {
			fragments = append(fragments, marshalFrag)
		}
	}
	text = bytes.Join(fragments, []byte{entrySeparatorChar, ' '})
	return
}

func (clist *List) UnmarshalText(text []byte) error {
	fragments := bytes.Split(text, []byte{entrySeparatorChar})
	result := make([]Entry, 0, len(fragments))
	for i, frag := range fragments {
		trimmed := bytes.TrimSpace(frag)
		if len(trimmed) == 0 {
			continue
		}
		var entry Entry
		err := entry.UnmarshalText(trimmed)
		if err != nil {
			return errors.Wrapf(err,
				"failed to unmarshal entry %d/%d = {%s}",
				i, len(fragments), frag)
		}
		result = append(result, entry)
	}
	*clist = result
	return nil
}
