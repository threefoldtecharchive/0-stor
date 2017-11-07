package db

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io/ioutil"
)

const (
	polynomial  = 0xD5828281
	RefIDCount  = 160
	RefIDLenght = 16

	PrefixData    = "data"
	PrefixRefList = "reflist"
)

var (
	// ErrReferenceTooLong is returned when the reference ID is longer then 16 bytes
	ErrReferenceTooLong = fmt.Errorf("too long reference ID, max = %v", RefIDLenght)
	// ErrReferenceListTooBig is returned when the reference list is bigger then 160 reference ID
	ErrReferenceListTooBig = fmt.Errorf("too much reference ID, max = %v", RefIDCount)
)

var (
	tabPolynomial *crc32.Table
	nilRefList    [RefIDLenght]byte
)

func init() {
	tabPolynomial = crc32.MakeTable(polynomial)
}

// Object is the data structure used to encode, decode object on the disk
type Object struct {
	db        DB
	Namespace string
	Key       []byte
	// A fixed size slice of reference list
	// If not full, all zero reference is used as the sentinel of valid reference list.
	// All reference after that value should be ignored
	referenceList [RefIDCount][RefIDLenght]byte
	crc           uint32
	data          []byte
}

func NewObject(namesapce string, key []byte, db DB) *Object {
	return &Object{
		db:        db,
		Namespace: namesapce,
		Key:       key,
	}
}

func (o *Object) dataKey() []byte {
	return []byte(fmt.Sprintf("%s:%s:%s", o.Namespace, PrefixData, o.Key))
}
func (o *Object) refListKey() []byte {
	return []byte(fmt.Sprintf("%s:%s:%s", o.Namespace, PrefixRefList, o.Key))
}

func (o *Object) SetData(b []byte) {
	o.data = b
	o.crc = crc32.Checksum([]byte(o.data), tabPolynomial)
}

func (o *Object) Data() ([]byte, error) {
	if o.data != nil {
		return o.data, nil
	}

	b, err := o.db.Get(o.dataKey())
	if err != nil {
		return nil, err
	}

	r := bytes.NewReader(b)

	// read CRC
	err = binary.Read(r, binary.LittleEndian, &o.crc)
	if err != nil {
		return nil, err
	}

	// read the rest of the data from the read
	o.data, err = ioutil.ReadAll(r)
	return o.data, err
}

func (o *Object) ReferenceList() ([][16]byte, error) {
	if !isNilReference(o.referenceList[0][:]) {
		return o.referenceList[:], nil
	}

	b, err := o.db.Get(o.refListKey())
	if err != nil {
		return nil, err
	}

	r := bytes.NewReader(b)

	refBuf := make([]byte, 16)
	for i := range o.referenceList {
		err = binary.Read(r, binary.LittleEndian, refBuf)
		if err != nil {
			return nil, err
		}
		if isNilReference(refBuf) {
			break
		}
		n := copy(o.referenceList[i][:], refBuf[:])
		if n != 16 {
			return nil, fmt.Errorf("error decoding reference list")
		}
	}

	return o.referenceList[:], nil
}

func (o *Object) CRC() (uint32, error) {
	if o.crc == 0 {
		data, err := o.Data()
		if err != nil {
			return 0, err
		}
		o.crc = crc32.Checksum([]byte(data), tabPolynomial)
	}
	return o.crc, nil
}

func (o *Object) SetReferenceList(refList []string) error {
	if len(refList) > RefIDCount {
		return ErrReferenceListTooBig
	}

	// copy
	for i := range refList {
		if len(refList[i]) > RefIDLenght {
			return ErrReferenceTooLong
		}
		copy(o.referenceList[i][:], []byte(refList[i]))
	}

	// create sentinel
	if len(refList) < RefIDCount {
		copy(o.referenceList[len(refList)][:], nilRefList[:])
	}

	return nil
}

// AppendReferenceList appends given reference list to the object.
// The append operation doesn't check whether the existing reference
// already exist
func (o *Object) AppendReferenceList(refList []string) error {
	numRefList, err := o.countRefList()
	if err != nil {
		return err
	}

	// make sure the append operation doesn't make our
	// ref list exceed the max len
	if len(refList)+numRefList > RefIDCount {
		return ErrReferenceListTooBig
	}

	// append it
	for i, ref := range refList {
		copy(o.referenceList[i+numRefList][:], []byte(ref))
	}

	currNumRefList := numRefList + len(refList)
	if currNumRefList < RefIDCount {
		copy(o.referenceList[currNumRefList][:], nilRefList[:])
	}
	return nil
}

// RemoveReferenceList removes given reference list from the object.
// It won't return error in case of some or all elements of the `refList`
// are not exist in the object.
func (o *Object) RemoveReferenceList(refList []string) error {
	// creates map of existing reference list for easier search
	// it might good for big refList but seems bad
	// for the little one
	refMap := make(map[string]struct{}, len(refList))
	for _, ref := range refList {
		refMap[ref] = struct{}{}
	}

	newRefList := make([]string, 0, RefIDCount)
	currentRefList, err := o.ReferenceList()
	if err != nil {
		return err
	}
	for _, ref := range currentRefList {
		strRef := referenceToStr(ref)
		if _, exists := refMap[strRef]; exists {
			continue
		}
		newRefList = append(newRefList, strRef)
	}
	return o.SetReferenceList(newRefList)
}

// GetreferenceListStr returns reference list in the form of []string
func (o *Object) GetreferenceListStr() ([]string, error) {
	refListBytes, err := o.ReferenceList()
	if err != nil {
		return nil, err
	}
	refListStr := make([]string, 0, len(refListBytes))
	for _, ref := range o.referenceList {
		if isNilReference(ref[:]) {
			return refListStr, nil
		}
		refListStr = append(refListStr, referenceToStr(ref))
	}

	return refListStr, nil
}

func referenceToStr(ref [RefIDLenght]byte) string {
	return string(bytes.TrimRight(ref[:], "\x00"))
}

// count number of reference list in this object
func (o *Object) countRefList() (int, error) {
	refList, err := o.ReferenceList()
	if err != nil {
		return 0, err
	}
	for i, ref := range refList {
		if isNilReference(ref[:]) {
			return i, nil
		}
	}
	return len(refList), nil
}

func isNilReference(ref []byte) bool {
	return bytes.Compare(ref, nilRefList[:]) == 0
}

// Validcrc compare the content of the data and the crc, return true if crc match data, false otherwrise
func (o *Object) Validcrc() (bool, error) {
	data, err := o.Data()
	if err != nil {
		return false, err
	}
	crc, err := o.CRC()
	if err != nil {
		return false, err
	}
	return crc32.Checksum(data, tabPolynomial) == crc, nil
}

// Save store the object onto disk
func (o *Object) Save() error {
	if err := o.saveData(); err != nil {
		return err
	}
	return o.saveRefList()
}

func (o *Object) Delete() error {
	// TODO: some left over if error during first delete
	if err := o.db.Delete(o.dataKey()); err != nil {
		return err
	}
	return o.db.Delete(o.refListKey())
}

func (o *Object) Exists() (bool, error) {
	return o.db.Exists(o.dataKey())
}

func (o *Object) saveData() error {
	if o.data == nil {
		// we didn't loaded the data, so no update, no need to save it
		return nil
	}

	var err error
	buf := &bytes.Buffer{}

	crc, err := o.CRC()
	if err != nil {
		return err
	}
	err = binary.Write(buf, binary.LittleEndian, crc)
	if err != nil {
		return err
	}

	err = binary.Write(buf, binary.LittleEndian, o.data)

	return o.db.Set(o.dataKey(), buf.Bytes())
}

func (o *Object) saveRefList() error {
	if o.referenceList[:] == nil {
		// reference list not loaded, so no update, no need to save it
		return nil
	}

	var err error
	buf := &bytes.Buffer{}

	for i := range o.referenceList {
		err = binary.Write(buf, binary.LittleEndian, o.referenceList[i][:])
		if err != nil {
			return err
		}
	}

	return o.db.Set(o.refListKey(), buf.Bytes())
}
