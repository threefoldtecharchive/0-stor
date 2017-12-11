package server

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReferenceList(t *testing.T) {
	// local util functions to easily create a reference list
	rl := func(elements ...string) ReferenceList { return ReferenceList(elements) }
	rls := func(str string) ReferenceList { return rl(strings.Split(str, ",")...) }

	// local test variables used in this test
	var (
		listA   ReferenceList
		require = require.New(t)
	)

	// removing from an empty/nil list should be possible
	listA.RemoveReferences(rl())
	listA.RemoveReferences(rls("a,b,c"))
	require.Nil(listA)

	// Let's append stuff to our list
	listA.AppendReferences(rl())
	require.Nil(listA)
	listA.AppendReferences(rl("a"))
	require.Equal(rl("a"), listA)
	listA.AppendReferences(rls("a,b"))
	require.Equal(rls("a,a,b"), listA)

	// now let's remove one "a"
	require.Empty(listA.RemoveReferences(rl("a")))
	require.Equal(rls("a,b"), listA)

	// not let's remove "a,b,a"
	require.Equal(rl("a"), listA.RemoveReferences(rls("a,b,a")))
	require.Empty(listA)
	// add "a,b" again
	listA.AppendReferences(rls("a,b"))
	require.Equal(rls("a,b"), listA)

	// now let's remove "a,a"
	require.Equal(rl("a"), listA.RemoveReferences(rls("a,a")))
	require.Equal(rl("b"), listA)
	// add "a" again and sort it
	listA.AppendReferences(rl("a"))
	require.Equal(rls("b,a"), listA)
	require.Equal(rls("f,o,o"), listA.RemoveReferences(rls("f,o,o")))
	require.Equal(rls("a,b"), listA)

	// let's add some more "a"s, "b"s and one "c"
	listA.AppendReferences(rls("b,b,a,b,a,a,c"))
	require.Equal(rls("a,b,b,b,a,b,a,a,c"), listA)

	// now let's remove "d", this doesn't exist,
	// it will however make our list be sorted once again
	require.Equal(rl("d"), listA.RemoveReferences(rl("d")))
	require.Equal(rls("a,a,a,a,b,b,b,b,c"), listA)

	// now let's remove "b,a,d,a,b,a"
	require.Equal(rl("d"), listA.RemoveReferences(rls("b,a,d,a,b,a")))
	require.Equal(rls("a,b,b,c"), listA)

	// now let's remove "b" and add it again
	require.Empty(listA.RemoveReferences(rl("b")))
	require.Equal(rls("a,b,c"), listA)
	listA.AppendReferences(rl("b"))
	require.Equal(rls("a,b,c,b"), listA)

	// now let's remove "c" only
	require.Empty(listA.RemoveReferences(rl("c")))

	// now let's add "d,o", and remove "b,a,d,a,b,a" again
	listA.AppendReferences(rls("d,o"))
	require.Equal(rls("a,b,b,d,o"), listA)
	require.Equal(rls("a,a"), listA.RemoveReferences(rls("b,a,d,a,b,a")))
	require.Equal(rl("o"), listA)

	// let's give it a "c"
	listA.AppendReferences(rl("c"))
	require.Equal(rls("o,c"), listA)

	// if we remove using a nil-list,
	// we won't have sorted our current list either,
	// as to be as lazy as possible
	require.Empty(listA.RemoveReferences(rl()))
	require.Equal(rls("o,c"), listA)

	// let's remove "f,o,o", won't do anything except sorting it
	require.Equal(rls("f,o,o"), listA.RemoveReferences(rls("f,o,o")))
	require.Equal(rls("c,o"), listA)

	// appending and removing nothing doesn't change anything
	listA.AppendReferences(rl())
	require.Equal(rls("c,o"), listA)
	require.Empty(listA.RemoveReferences(rl()))
	require.Equal(rls("c,o"), listA)
	listA.AppendReferences(rl())
	require.Equal(rls("c,o"), listA)

	// now let's remove "c,o,o,c", and we're done
	require.Equal(rls("c,o"), listA.RemoveReferences(rls("c,o,c,o")))
	require.Empty(listA)

	// let's do some testing with empty strings
	listA.AppendReferences(rl("大家好", "", "大家好"))
	require.Equal(rl("大家好", "", "大家好"), listA)
	require.Equal(rl(""), listA.RemoveReferences(rl("", "大家好", "", "大家好")))
	require.Empty(listA)

	// let's test some more appending and removing
	listA.AppendReferences(rls("bar,baz,bong,bang"))
	require.Equal(rls("bar,baz,bong,bang"), listA)
	require.Equal(rls("bar,baz,bin"),
		listA.RemoveReferences(rls("bar,baz,baz,bar,bong,bin")))
	require.Equal(rl("bang"), listA)
}

func TestDefaultObjectStatus(t *testing.T) {
	var status ObjectStatus
	require.Equal(t, ObjectStatusMissing, status)
}

func TestObjectStatusString(t *testing.T) {
	require := require.New(t)

	require.Equal("ok", ObjectStatusOK.String())
	require.Equal("missing", ObjectStatusMissing.String())
	require.Equal("corrupted", ObjectStatusCorrupted.String())
}
