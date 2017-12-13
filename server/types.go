package server

import (
	"sort"
)

type (
	// Object is the data structure for all data (blobs) stored in the 0-stor server.
	// It is used to exchange it between the codebase.
	// The decoding and encoding happens in the encoding package.
	Object struct {
		// Data in its raw encoded form
		Data []byte
	}

	// Namespace is the data structure for all namespaces referenced and stored in the 0-stor server.
	// It is used to exchange namespaces between the codebase.
	// The decoding and encoding happens in the encoding package.
	Namespace struct {
		// Reserved (total) space in this namespace (in bytes)
		Reserved uint64
		// Label (or name) of the namespace
		Label []byte
	}

	// StoreStat is the data structure for the global store statistics,
	// stored for a single 0-stor server. One per server and thus one per database.
	// It is used to exchange the stats between the codebase.
	// The decoding and encoding happens in the encoding package.
	StoreStat struct {
		// Available disk space in bytes
		SizeAvailable uint64
		// Space used in bytes
		SizeUsed uint64
	}
)

// ReferenceList is the data type used to store identifiers of entities,
// which reference a specific object, the object which stores this reference list.
//
// A reference list is not a unique element of lists, and it is supported to have
// the same element multiple times in a list.
// Thus if you have /X/ two times in the same reference list,
// and you would remove the list [/X/] from that list,
// you would still end up with one other /X/ in the current list.
//
// A ReferenceList is not thread-safe!
type ReferenceList []string

// AppendReferences appends the elements, of the other list, to the current list.
func (list *ReferenceList) AppendReferences(other ReferenceList) {
	*list = append(*list, other...)
}

// RemoveReferences removes the elements, of the other list, from the current list.
// Both the current list and other list are sorted prior to removal,
// this is required as elements appended to a ReferenceList might
// make that list unsorted.
//
// RemoveReferences returns at the end the elements that were part of other
// but weren't part of this list and thus couldn't be removed.
func (list *ReferenceList) RemoveReferences(other ReferenceList) (rest ReferenceList) {
	otherLength := len(other)
	if otherLength == 0 {
		return // nop-operation
	}

	// sort both the resulting and other list
	sort.Strings(*list)
	sort.Strings(other)

	// store the current list in a result var, to make it easier to use
	result := *list

	// local variables used throughout this function
	var (
		remaining               ReferenceList
		resultIndex, otherIndex int
	)

resultLoop:
	for resultIndex < len(result) {
		element := result[resultIndex]
		// compare the elements
		if element != other[otherIndex] {
			resultIndex++
			if resultIndex >= len(result) {
				continue
			}
			// ensure to skip any elements which are less than the current index
			for other[otherIndex] < result[resultIndex] {
				remaining = append(remaining, other[otherIndex])
				otherIndex++
				if otherIndex >= otherLength {
					break resultLoop
				}
			}
			continue
		}

		// remove the element from the resulting list
		result = append(result[:resultIndex], result[resultIndex+1:]...)

		// move the cursor from other by one
		otherIndex++

		// if either one of the lists are finished, we can stop
		if otherIndex >= otherLength || resultIndex >= len(result) {
			break resultLoop
		}

		// ensure to skip extra elements in other,
		// which are equal to the currently removed result,
		// but which no longer are part of the resulting list
		for element == other[otherIndex] && element != result[resultIndex] {
			remaining = append(remaining, element)
			otherIndex++
			if otherIndex >= otherLength {
				break resultLoop
			}
		}

		// ensure to skip any elements which are less than the current index
		for other[otherIndex] < result[resultIndex] {
			remaining = append(remaining, other[otherIndex])
			otherIndex++
			if otherIndex >= otherLength {
				break resultLoop
			}
		}
	}

	// add all remaining others as well
	remaining = append(remaining, other[otherIndex:]...)

	// use the result as this list, and return the remaining elements,
	// that is elements which were part of the other list,
	// but couldn't be removed from the resulting list.
	*list = result
	return remaining
}

// ObjectStatus represents the status received after checking,
// whether or not an object is ok
type ObjectStatus uint8

const (
	// ObjectStatusMissing indicates the requested object doesn't exist.
	ObjectStatusMissing ObjectStatus = iota
	// ObjectStatusOK indicates the requested object exists and is healthy
	ObjectStatusOK
	// ObjectStatusCorrupted indicates the requested object exists,
	// but its checksum indicates it is corrupted,
	// this can be about the data itself as well as its reference list.
	ObjectStatusCorrupted
)

// String implements Stringer.String
func (status ObjectStatus) String() string {
	str, ok := _ObjectStatusValueStringMapping[status]
	if !ok {
		return ""
	}
	return str
}

const _ObjectStatusStrings = "okcorruptedmissing"

var _ObjectStatusValueStringMapping = map[ObjectStatus]string{
	ObjectStatusOK:        _ObjectStatusStrings[:2],
	ObjectStatusCorrupted: _ObjectStatusStrings[2:11],
	ObjectStatusMissing:   _ObjectStatusStrings[11:],
}
