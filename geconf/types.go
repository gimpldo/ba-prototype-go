/*
Package geconf defines types for General/Global and per-Element Configuration.
Includes marshaling to/from text.

"Element" here is intended to mean some kind of schema element
(for example, a table in an SQL database) but may be generalized to mean
the ability to specify configuration options for a part of a whole.
*/
package geconf

// Notes on the package name chosen:
// 'geconf' seems to be an abbreviation for Dutch words like
// 'geconfronteerd' = "confronted" (English), or
// 'geconfigureerd' = "configured" (English).
// The last one is particularly fit.
//
// Most important is that these words do NOT seem:
//  - indecent,
//  - offending, or
//  - distracting (not likely to cause giggles or irrelevant discussion).
// This is what we hope, at least.

import (
	"encoding"
	"sort"
)

// Entry = configuration entry (corresponds to a table row if using an SQL DB)
type Entry struct {
	ConfElement  string
	ConfProperty string
	ConfValue    string

	// A configuration entry's "rank" is intended for sorting only,
	// not to be stored or marshaled; should be set by the application
	// based on whatever rule is deemed useful.
	Rank int32
}

// List = named type so we can implement TextMarshaler and TextUnmarshaler
// for a slice of configuration entries
type List []Entry

// CanonicalOrder = named type so we can implement sorting
// for a slice of configuration entries
type CanonicalOrder []Entry

// RankOrder = named type so we can implement sorting by rank
// for a slice of configuration entries
type RankOrder []Entry

func (cord CanonicalOrder) Len() int      { return len(cord) }
func (cord CanonicalOrder) Swap(i, j int) { cord[i], cord[j] = cord[j], cord[i] }

func (cord CanonicalOrder) Less(i, j int) bool {
	switch {
	case cord[i].ConfElement < cord[j].ConfElement:
		return true
	case cord[i].ConfElement > cord[j].ConfElement:
		return false
	default:
		switch {
		case cord[i].ConfProperty < cord[j].ConfProperty:
			return true
		case cord[i].ConfProperty > cord[j].ConfProperty:
			return false
		default:
			return cord[i].ConfValue < cord[i].ConfValue
		}
	}
}

func (rord RankOrder) Len() int      { return len(rord) }
func (rord RankOrder) Swap(i, j int) { rord[i], rord[j] = rord[j], rord[i] }

func (rord RankOrder) Less(i, j int) bool {
	switch {
	case rord[i].Rank < rord[j].Rank:
		return true
	case rord[i].Rank > rord[j].Rank:
		return false
	default:
		switch {
		case rord[i].ConfElement < rord[j].ConfElement:
			return true
		case rord[i].ConfElement > rord[j].ConfElement:
			return false
		default:
			switch {
			case rord[i].ConfProperty < rord[j].ConfProperty:
				return true
			case rord[i].ConfProperty > rord[j].ConfProperty:
				return false
			default:
				return rord[i].ConfValue < rord[i].ConfValue
			}
		}
	}
}

// Explicitly check that the canonical and rank order that
// were defined for a slice of configuration entries can be used by
// the standard sorting routines (the standard Go "sort" package).
var _ sort.Interface = CanonicalOrder{}
var _ sort.Interface = RankOrder{}

// Explicitly check that Entry and List implement completely
// the text encoding interfaces: TextMarshaler and TextUnmarshaler.
var (
	_ encoding.TextMarshaler   = Entry{}
	_ encoding.TextUnmarshaler = (*Entry)(nil)
	_ encoding.TextMarshaler   = List{}
	_ encoding.TextUnmarshaler = (*List)(nil)
)
