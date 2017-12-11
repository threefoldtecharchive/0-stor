package encoding

import (
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zero-os/0-stor/server"
)

func TestObjectEncodingDecoding(t *testing.T) {
	require := require.New(t)

	validTestCases := []server.Object{
		server.Object{Data: []byte("1")},
		server.Object{Data: []byte("Hello, World!")},
		server.Object{Data: []byte("大家好")},
	}
	for _, validTestCase := range validTestCases {
		data, err := EncodeObject(validTestCase)
		require.NoError(err)
		require.NotNil(data)

		obj, err := DecodeObject(data)
		require.NoError(err)
		require.Equal(validTestCase, obj)
	}
}

func TestInvalidObjectDecoding(t *testing.T) {
	require := require.New(t)

	// invalid encodings as nil data was given,
	// or because not enough data was given, to be even possibly valid
	_, err := DecodeObject(nil)
	require.Equal(ErrInvalidData, err)
	_, err = DecodeObject([]byte{4, 2})
	require.Equal(ErrInvalidData, err)
	_, err = DecodeObject([]byte{1, 2, 3, 4})
	require.Equal(ErrInvalidData, err)

	// invalid crc
	_, err = DecodeObject([]byte{1, 2, 3, 4, 5})
	require.Equal(ErrInvalidChecksum, err)
}

func TestInvalidObjectEncoding(t *testing.T) {
	require := require.New(t)

	// an error is only to be expected in case the
	// given object has no data defined
	_, err := EncodeObject(server.Object{})
	require.Error(err)
	_, err = EncodeObject(server.Object{Data: nil})
	require.Error(err)
}

func TestNamespaceEncodingDecoding(t *testing.T) {
	require := require.New(t)

	validTestCases := []server.Namespace{
		server.Namespace{Label: []byte("1")},
		server.Namespace{Reserved: 1, Label: []byte("1")},
		server.Namespace{Reserved: 42, Label: []byte("42")},
		server.Namespace{Reserved: math.MaxUint64, Label: []byte("大家好")},
	}
	for _, validTestCase := range validTestCases {
		data, err := EncodeNamespace(validTestCase)
		require.NoError(err)
		require.NotNil(data)

		ns, err := DecodeNamespace(data)
		require.NoError(err)
		require.Equal(validTestCase, ns)
	}
}

func TestInvalidNamespaceDecoding(t *testing.T) {
	require := require.New(t)

	// invalid encodings as nil data was given,
	// or because not enough data was given, to be even possibly valid
	_, err := DecodeNamespace(nil)
	require.Equal(ErrInvalidData, err)
	_, err = DecodeNamespace([]byte{4, 2})
	require.Equal(ErrInvalidData, err)
	_, err = DecodeNamespace([]byte{1, 2, 3, 4})
	require.Equal(ErrInvalidData, err)
	_, err = DecodeNamespace([]byte{1, 2, 3, 4, 5, 6, 7, 8})
	require.Equal(ErrInvalidData, err)
	_, err = DecodeNamespace([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12})
	require.Equal(ErrInvalidData, err)

	// invalid crc
	_, err = DecodeNamespace([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13})
	require.Equal(ErrInvalidChecksum, err)
}

func TestInvalidNamespaceEncoding(t *testing.T) {
	require := require.New(t)

	// an error is only to be expected in case the
	// given namespace has no label defined
	_, err := EncodeNamespace(server.Namespace{})
	require.Error(err)
	_, err = EncodeNamespace(server.Namespace{Reserved: 42})
	require.Error(err)
	_, err = EncodeNamespace(server.Namespace{Reserved: 42, Label: nil})
	require.Error(err)
}

func TestStoreStatEncodingDecoding(t *testing.T) {
	require := require.New(t)

	validTestCases := []server.StoreStat{
		server.StoreStat{},
		server.StoreStat{SizeAvailable: 1, SizeUsed: 0},
		server.StoreStat{SizeAvailable: 0, SizeUsed: 1},
		server.StoreStat{SizeAvailable: 1, SizeUsed: 1},
		server.StoreStat{SizeAvailable: math.MaxUint64, SizeUsed: 0},
		server.StoreStat{SizeAvailable: math.MaxUint64, SizeUsed: 42},
		server.StoreStat{SizeAvailable: 0, SizeUsed: math.MaxUint64},
		server.StoreStat{SizeAvailable: 42, SizeUsed: math.MaxUint64},
		server.StoreStat{SizeAvailable: 123456789, SizeUsed: 987654321},
		server.StoreStat{SizeAvailable: math.MaxUint64, SizeUsed: math.MaxUint64},
	}
	for _, validTestCase := range validTestCases {
		data := EncodeStoreStat(validTestCase)
		require.NotNil(data)

		stat, err := DecodeStoreStat(data)
		require.NoError(err)
		require.Equal(validTestCase, stat)
	}
}

func TestInvalidStoreStatDecoding(t *testing.T) {
	require := require.New(t)

	// invalid encodings as nil data was given,
	// or because not enough data was given, to be even possibly valid
	_, err := DecodeStoreStat(nil)
	require.Equal(ErrInvalidData, err)
	_, err = DecodeStoreStat([]byte{4, 2})
	require.Equal(ErrInvalidData, err)
	_, err = DecodeStoreStat([]byte{1, 2, 3, 4})
	require.Equal(ErrInvalidData, err)
	_, err = DecodeStoreStat([]byte{1, 2, 3, 4, 5, 6, 7, 8})
	require.Equal(ErrInvalidData, err)
	_, err = DecodeStoreStat([]byte{1, 2, 3, 4,
		5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19})
	require.Equal(ErrInvalidData, err)

	// invalid crc
	_, err = DecodeStoreStat([]byte{1, 2, 3, 4,
		5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20})
	require.Equal(ErrInvalidChecksum, err)
}

func TestReferenceListEncodingDecoding(t *testing.T) {
	require := require.New(t)

	validTestCases := []server.ReferenceList{
		server.ReferenceList{""},
		server.ReferenceList{"a"},
		server.ReferenceList{"a", "", "b", "", "c"},
		server.ReferenceList{"bar", "大家好", "foo", "1"},
		server.ReferenceList{"a", "bo", "foo", "cup of tea"},
	}
	for _, validTestCase := range validTestCases {
		data, err := EncodeReferenceList(validTestCase)
		require.NoError(err)
		require.NotNil(data)

		list, err := DecodeReferenceList(data)
		require.NoError(err)
		require.Equal(validTestCase, list)
	}
}

func TestInvalidReferenceListEncoding(t *testing.T) {
	require := require.New(t)

	// no references given too encode
	_, err := EncodeReferenceList(server.ReferenceList{})
	require.Error(err)

	// a reference cannot exceed the given max size
	_, err = EncodeReferenceList(server.ReferenceList{string(make([]byte, MaxReferenceIDLength+1))})
	require.Equal(ErrReferenceIDTooLarge, err)
}

func TestInvalidReferenceListDecoding(t *testing.T) {
	require := require.New(t)

	// invalid encodings as nil data was given,
	// or because not enough data was given, to be even possibly valid
	_, err := DecodeReferenceList(nil)
	require.Equal(ErrInvalidData, err)
	_, err = DecodeReferenceList([]byte{4, 2})
	require.Equal(ErrInvalidData, err)
	_, err = DecodeReferenceList([]byte{1, 2, 3, 4})
	require.Equal(ErrInvalidData, err)

	// invalid crc
	_, err = DecodeReferenceList([]byte{1, 2, 3, 4, 5})
	require.Equal(ErrInvalidChecksum, err)

	// invalid reference length
	data := make([]byte, checksumSize+1)
	data[checksumSize] = 42
	packageData(data)
	_, err = DecodeReferenceList(data)
	require.Equal(ErrInvalidData, err)

	// too small reference length
	data = make([]byte, checksumSize+2)
	data[checksumSize] = 2
	data[checksumSize+1] = 'a'
	packageData(data)
	_, err = DecodeReferenceList(data)
	require.Equal(ErrInvalidData, err)
}

func TestAppendToReferenceList(t *testing.T) {
	require := require.New(t)

	// local util functions to easily create a reference list
	rl := func(elements ...string) server.ReferenceList { return server.ReferenceList(elements) }
	rls := func(str string) server.ReferenceList { return rl(strings.Split(str, ",")...) }

	validTestCases := []struct {
		first, second, result server.ReferenceList
	}{
		{rl(""), rl(), rl("")},
		{rl("a"), rl("b"), rls("a,b")},
		{rl("a"), rls("b,a"), rls("a,b,a")},
		{rls("f,foo"), rls("b,bar"), rls("f,foo,b,bar")},
		{rls("foo,bar"), rl("baz", "", "a", "大家好", ""),
			rl("foo", "bar", "baz", "", "a", "大家好", "")},
	}
	for _, validTestCase := range validTestCases {
		data, err := EncodeReferenceList(validTestCase.first)
		require.NoError(err)
		require.NotNil(data)

		data, err = AppendToEncodedReferenceList(data, validTestCase.second)
		require.NoError(err)
		require.NotNil(data)

		list, err := DecodeReferenceList(data)
		require.NoError(err)
		require.Equal(validTestCase.result, list)
	}
}

func TestInvalidAppendToReferenceList(t *testing.T) {
	require := require.New(t)

	// invalid encodings as nil data was given,
	// or because not enough data was given, to be even possibly valid
	_, err := AppendToEncodedReferenceList(nil, server.ReferenceList{})
	require.Equal(ErrInvalidData, err)
	_, err = AppendToEncodedReferenceList([]byte{1, 2, 3, 4}, server.ReferenceList{})
	require.Equal(ErrInvalidData, err)

	// invalid crc
	_, err = AppendToEncodedReferenceList([]byte{1, 2, 3, 4, 5}, server.ReferenceList{})
	require.Equal(ErrInvalidChecksum, err)

	// too big reference
	data := make([]byte, checksumSize+1)
	packageData(data)
	_, err = AppendToEncodedReferenceList(data,
		server.ReferenceList{string(make([]byte, MaxReferenceIDLength+1))})
	require.Equal(ErrReferenceIDTooLarge, err)
}

func TestRemoveFromReferenceList(t *testing.T) {
	require := require.New(t)

	// local util functions to easily create a reference list
	rl := func(elements ...string) server.ReferenceList { return server.ReferenceList(elements) }
	rls := func(str string) server.ReferenceList { return rl(strings.Split(str, ",")...) }

	validTestCases := []struct {
		first, second, result server.ReferenceList
	}{
		{rl(""), rl(), rl("")},
		{rl("a"), rl("b"), rls("a")},
		{rl("a"), rl("a"), rl()},
		{rls("a,b"), rl("a"), rl("b")},
		{rls("a,b"), rl("b"), rl("a")},
		{rls("f,o,o"), rl("f"), rls("o,o")},
		{rls("f,o,o"), rls("o,o,o"), rl("f")},
		{rls("f,o,o"), rls("o,f"), rl("o")},
		{rls("bar,baz,bong,bang"), rls("bar,baz,baz,bar,bong,bin"), rl("bang")},
		{rl("大家好", "", "大家好"), rl("", "大家好", "", "大家好"), rl()},
		{rl("大家好", "", "foo", "大家好"), rl("", "大家好", "", "大家好"), rl("foo")},
	}
	for _, validTestCase := range validTestCases {
		data, err := EncodeReferenceList(validTestCase.first)
		require.NoError(err)
		require.NotNil(data)

		data, count, err := RemoveFromEncodedReferenceList(data, validTestCase.second)
		require.NoErrorf(err, "%v", validTestCase)

		if len(validTestCase.result) == 0 {
			require.Nilf(data, "%v", validTestCase)
			continue
		}
		require.NotNil(data, "%v", validTestCase)

		list, err := DecodeReferenceList(data)
		require.NoError(err)
		require.Equal(validTestCase.result, list)
		require.Len(list, count)
	}
}

func TestInvalidReferenceRemoveDecoding(t *testing.T) {
	require := require.New(t)

	// invalid encodings as nil data was given,
	// or because not enough data was given, to be even possibly valid
	_, _, err := RemoveFromEncodedReferenceList([]byte{1, 2, 3, 4}, server.ReferenceList{})
	require.Equal(ErrInvalidData, err)

	// invalid crc
	_, _, err = RemoveFromEncodedReferenceList([]byte{1, 2, 3, 4, 5}, server.ReferenceList{})
	require.Equal(ErrInvalidChecksum, err)

	// invalid reference length
	data := make([]byte, 5)
	data[checksumSize] = 42
	packageData(data)
	_, _, err = RemoveFromEncodedReferenceList(data, server.ReferenceList{})
	require.Equal(ErrInvalidData, err)
}

func TestValidateData(t *testing.T) {
	// TODO
}

func TestDataPackaging(t *testing.T) {
	require := require.New(t)

	testCases := []string{
		"1",
		"Hello, World!",
		"大家好",
	}

	for _, testCase := range testCases {
		// allocate proper data package
		data := make([]byte, checksumSize+len(testCase))
		copy(data[checksumSize:], testCase[:])

		// package our fresh data
		packageData(data)

		// unpackage it again
		blob, err := unpackageData(data)
		require.NoError(err)
		require.Equal(testCase, string(blob))

		// let's fool around a bit with our data to make it invalid,
		// the only error at this point that is really possible,
		// would be when the blob's crc (packaged as part of the data)
		// does not match the blob
		origByte := data[0]
		data[0] = data[1]
		data[1] = data[2]
		blob, err = unpackageData(data)
		require.Equal(ErrInvalidChecksum, err)
		require.Nil(blob)
		// repair the crc binary data again,
		// let's validate we can still unpackage normally
		data[1] = data[0]
		data[0] = origByte
		blob, err = unpackageData(data)
		require.NoError(err)
		require.Equal(testCase, string(blob))
		// and let's than try to mess with the data
		data[len(data)-1] = 2
		data[len(data)-2] = 4
		blob, err = unpackageData(data)
		require.Equal(ErrInvalidChecksum, err)
		require.Nil(blob)
	}
}
