package main

import (
"gopkg.in/validator.v2"
	"fmt"
	"time"
	"math"
	"strings"
	"github.com/zero-os/0-stor/store/core/goraml"
	"github.com/zero-os/0-stor/store/utils"
	"encoding/binary"
	"errors"
	"github.com/zero-os/0-stor/store/core/librairies/reservation"
	"encoding/base64"
	log "github.com/Sirupsen/logrus"
)

/* NamespacesNsidReservationPostRespBody */

type NamespacesNsidReservationPostRespBody struct {
	DataAccessToken  string                  `json:"dataAccessToken" validate:"nonzero"`
	Reservation      reservation.Reservation `json:"reservation" validate:"nonzero"`
	ReservationToken string                  `json:"reservationToken" validate:"nonzero"`
}

func (s NamespacesNsidReservationPostRespBody) Validate() error {
	return validator.Validate(s)
}


/* ACL */

type ACL struct {
	ParentModel
	Acl ACLEntry `json:"acl" validate:"nonzero"`
	Id  string   `json:"id" validate:"regexp=^\w+$,nonzero"`
}

func (s ACL) Validate() error {
	return validator.Validate(s)
}

/*

| ACLEntry  | Id      |
|-----------|---------|
| 4 bytes   | variable|

 */
func (s ACL) ToBytes() []byte{
	Id := []byte(s.Id)
	b := make([]byte, len(Id) + 4)
	copy(b[0:4], s.Acl.ToBytes())
	copy(b[4:], Id)
	return b
}

func (s *ACL) FromBytes(data []byte){
	s.Acl = ACLEntry{}
	s.Acl.FromBytes(data[0:4])
	s.Id = string(data[4:])
}

// ACL entry for a reservation
type ACLEntry struct {
	Admin  bool `json:"admin"`
	Delete bool `json:"delete"`
	Read   bool `json:"read"`
	Write  bool `json:"write"`
}

func (s ACLEntry) Validate() error {

	return validator.Validate(s)
}

/*

| Admin  | Read  | Write  | Delete  |
|--------|-------|--------|---------|
| 1 byte | 1 byte| 1 byte | 1 byte  |

 */
func(s ACLEntry) ToBytes() []byte{
	r := []byte{0, 0, 0, 0}

	if s.Admin{
		r[0] = 1
	}

	if s.Read{
		r[1] = 1

	}

	if s.Write{
		r[2] = 1
	}

	if s.Delete{
		r[3] = 1
	}
	return r
}

func (s *ACLEntry) FromBytes(data []byte){
	if data[0] == 1{
		s.Admin = true
	}else{
		s.Admin = false
	}

	if data[1] == 1{
		s.Read = true
	}else{
		s.Read = false
	}

	if data[2] == 1{
		s.Write = true
	}else{
		s.Write = false
	}

	if data[3] == 1{
		s.Delete = true
	}else{
		s.Delete = false
	}
}


func (s *ACLEntry) SetAdminPrivileges() *ACLEntry{
	s.Admin = true
	s.Delete = true
	s.Read = true
	s.Write = true

	return s
}


type Namespace struct {
	NamespaceCreate
	SpaceAvailable float64 `json:"spaceAvailable,omitempty"`
	SpaceUsed      float64 `json:"spaceUsed,omitempty"`
}

func (s Namespace) Validate() error {

	return validator.Validate(s)
}

func (s Namespace) ToBytes() []byte{
	nsc := s.NamespaceCreate.ToBytes()
	b := make([]byte, len(nsc) + 16)

	copy(b[0:8], utils.Float64bytes(s.SpaceAvailable))
	copy(b[8:16], utils.Float64bytes(s.SpaceUsed))
	copy(b[16:], nsc)

	return b
}

func (s *Namespace) FromBytes(data[]byte){
	s.SpaceAvailable = utils.Float64frombytes(data[0:8])
	s.SpaceUsed = utils.Float64frombytes(data[8:16])

	nsc := NamespaceCreate{}
	nsc.FromBytes(data[16:])

	s.NamespaceCreate = nsc
}

type NamespaceCreate struct {
	Acl   []ACL  `json:"acl"`
	Label string `json:"label" validate:"min=5,max=128,regexp=^[a-zA-Z0-9]+$,nonzero"`
}

func (s NamespaceCreate) Validate() error {
	return validator.Validate(s)
}

/*
 label size   | ACL[] length |Label   |ACL[0]  SizeAvailable | ACL[0] |
|-------------|--------------|--------|-------------|--------|
| 2 bytes     | 2 bytes      |        |    2 bytes |        |

 */

func (s NamespaceCreate) ToBytes() []byte{
	label := []byte(s.Label)
	labelSize := len(label)

	size := len(label)

	acls := [][]byte{}

	for _, acl := range s.Acl{
		bytes := acl.ToBytes()
		acls = append(acls, bytes)
		size += len(bytes)
		size += 2 // 2 bytes to old size of each element
	}

	b := make([]byte, size+4)

	binary.LittleEndian.PutUint16(b[0:2], uint16(labelSize))
	binary.LittleEndian.PutUint16(b[2:4], uint16(len(acls)))

	start := 4
	end := 4 + labelSize
	copy(b[start:end], label)

	for _, a := range acls{
		aSize := len(a)
		start = end
		end = end + 2
		binary.LittleEndian.PutUint16(b[start:end], uint16(aSize))
		start = end
		end = end + aSize
		copy(b[start:end], a)
	}

	return b
}

func (s *NamespaceCreate) FromBytes(data []byte){
	lSize := int16(binary.LittleEndian.Uint16(data[0:2]))
	aSize := int16(binary.LittleEndian.Uint16(data[2:4]))

	start := int16(4)
	end := 4 + lSize
	s.Label = string(data[start: end])

	for i:= 0; i < int(aSize); i++{
		start = end
		end = end + 2
		size := int16(binary.LittleEndian.Uint16(data[start:end]))
		start = end
		end = end + size
		entry := ACL{}
		entry.FromBytes(data[start:end])
		s.Acl = append(s.Acl, entry)
	}
}

func (s NamespaceCreate) UpdateACL(db DB, config *Settings, acl ACL) error{
	aclIndex := -1 // -1 means ACL for that user does not exist

	// Find if ACL for that user already exists
	for i, item := range s.Acl {
		if item.Id == acl.Id {
			aclIndex = i
			break
		}
	}

	// Update User ACL
	if aclIndex != -1 {
		s.Acl[aclIndex] = acl
	} else { // Insert new ACL
		s.Acl = append(s.Acl, acl)
	}

	return s.Save(db, config)

}

func (s NamespaceCreate) Exists(db DB, config *Settings) (bool, error){
	exists, err := db.Exists(s.Label)
	if err != nil{
		log.Errorln(err.Error())
		return exists, err
	}
	return exists, nil
}

func (s NamespaceCreate) Save(db DB, config *Settings) error{
	return db.Set(s.Label, s.ToBytes())
}

func (NamespaceCreate) GetKeyForNamespace(label string, config *Settings) string{
	return fmt.Sprintf("%s%s", config.Namespace.Prefix, label)
}


func (s *NamespaceCreate) Get(db DB, config *Settings) (*NamespaceCreate, error){
	key := s.GetKeyForNamespace(s.Label, config)
	v, err := db.Get(key)
	if err != nil{
		return nil, err
	}

	if len(v) == 0{
		return nil, nil
	}
	s.FromBytes(v)
	return s, nil
}

func (s *NamespaceCreate) GetStats(db DB, config *Settings) (*NamespaceStats, error){
	stats := NamespaceStats{
		Namespace:s.Label,
	}
	return stats.Get(db, config)
}

func (s *NamespaceCreate) GetKeyForReservations(config *Settings) string{
	r := Reservation{
		Namespace: s.Label,
	}
	return r.GetKey(config)
}


type NamespaceStat struct {
	NrObjects      int64 `json:"NrObjects" validate:"nonzero"`
	RequestPerHour int64 `json:"requestPerHour" validate:"nonzero"`
}

func (s NamespaceStat) Validate() error {
	return validator.Validate(s)
}


type NamespaceStats struct{
	NamespaceStat
	Namespace         string
	NrRequests        int64
	TotalSizeReserved float64
	Created           time.Time
}

func NewNamespaceStats(namespace string) *NamespaceStats{
	return &NamespaceStats{
		Namespace: namespace,
		Created: time.Now(),
		NrRequests: 0,
		NamespaceStat : NamespaceStat{
			NrObjects: 0,
		},
	}
}

func (s *NamespaceStats) ToBytes() []byte{
	/*
	------------------------------------
	NrObjects|NrRwquests|SizeUSed|Created
	   8     |   8      | 8bytes
	-------------------------------------

	*/

	created := []byte(time.Time(s.Created).Format(time.RFC3339))
	result := make([]byte, 24+len(created))
	binary.LittleEndian.PutUint64(result[0:8], uint64(s.NrObjects))
	binary.LittleEndian.PutUint64(result[8:16], uint64(s.NrRequests))

	copy(result[16:24], utils.Float64bytes(s.TotalSizeReserved))

	copy(result[24:], created)
	return result
}

func (s *NamespaceStats) FromBytes(data []byte) error{
	s.NrObjects = int64(binary.LittleEndian.Uint64(data[0:8]))
	s.NrRequests = int64(binary.LittleEndian.Uint64(data[8:16]))
	s.TotalSizeReserved = utils.Float64frombytes(data[16:24])
	cTime, err := time.Parse(time.RFC3339, string(data[24:]))

	if err != nil{
		return err
	}

	s.Created = cTime
	s.RequestPerHour = int64(float64(s.NrRequests) / math.Ceil(time.Since(cTime).Hours()))
	return nil
}

func (s NamespaceStats) GetKeyForNameSpace(config *Settings) string{
	label := s.Namespace
	if strings.Index(s.Namespace, config.Namespace.Prefix) != -1{
		label = strings.Replace(s.Namespace, config.Namespace.Prefix, "", 1)
	}
	return fmt.Sprintf("%s%s", config.Namespace.Stats.Prefix, label)
}

func (s NamespaceStats) Save(db DB, config *Settings) error{
	key := s.GetKeyForNameSpace(config)
	return db.Set(key, s.ToBytes())
}

func (s NamespaceStats) Delete(db DB, config *Settings) error{
	key := s.GetKeyForNameSpace(config)
	return db.Delete(key)
}

func (s *NamespaceStats) Get(db DB, config *Settings) (*NamespaceStats, error){
	key := s.GetKeyForNameSpace(config)
	v, err := db.Get(key)
	if err != nil{
		return nil, err
	}

	if v == nil{
		return nil, errors.New("Namespace stats not found")
	}

	s.FromBytes(v)
	return s, nil
}

const (
	FileSize = 1024 * 1024
	CRCSize  = 32
)

type Object struct {
	Data string `json:"data" validate:"nonzero"`
	Id   string `json:"id" validate:"min=5,max=128,regexp=^[a-zA-Z0-9]+$,nonzero"`
	Tags []Tag  `json:"tags,omitempty"`
}

type File struct {
	Reference byte
	CRC       [32]byte
	Payload   []byte
	Tags      []byte
}

func (s Object) Validate() error {

	return validator.Validate(s)
}

func (f *File) ToBytes() []byte {
	size := len(f.Payload) + CRCSize + 1
	result := make([]byte, size)
	// First byte is reference
	result[0] = f.Reference
	// Next 32 bytes CRC

	copy(result[1:CRCSize+1], f.CRC[:])
	// Next 1Mbs (file content)
	copy(result[CRCSize+1:], f.Payload)

	return result
}

func(f *File) Size() float64{
	return math.Ceil((float64(len(f.Payload)) / (1024.0*1024.0)))
}


func (f *File) FromBytes(data []byte) error {
	if len(data) > FileSize+CRCSize {
		return errors.New("Data size exceeds limits")
	} else if len(data) <= CRCSize {
		return errors.New(fmt.Sprintf("Invalid file size (%v) bytes", len(data)))
	}

	var crc [CRCSize]byte

	copy(crc[:], data[1:CRCSize+1])

	var maxIdx int

	if len(data) > FileSize+CRCSize+1 {
		maxIdx = FileSize + CRCSize
	} else {
		maxIdx = len(data) - 1
	}

	var payload = make([]byte, maxIdx-CRCSize)

	copy(payload, data[CRCSize+1:])

	f.Reference = data[0]
	f.CRC = crc
	f.Payload = payload

	return nil
}

func (f *File) ToObject(data []byte, Id string) *Object {
	return &Object{
		Id:   Id,
		Data: string(data[1:]),
	}
}

func (o *Object) ToFile(addReferenceByte bool) (*File, error) {
	file := &File{}
	var data []byte
	bytes := []byte(o.Data)

	// add reference
	if addReferenceByte {
		data = make([]byte, len(bytes)+1)
		data[0] = byte(1)
		copy(data[1:], bytes)
	} else {
		data = bytes
	}

	err := file.FromBytes(data)
	return file, err
}

type ObjectCreate struct {
}

func (s ObjectCreate) Validate() error {

	return validator.Validate(s)
}



type ObjectUpdate struct {
	Data string `json:"data" validate:"nonzero"`
	Tags []Tag  `json:"tags,omitempty"`
}

func (s ObjectUpdate) Validate() error {

	return validator.Validate(s)
}

func (o *ObjectUpdate) ToFile(addReferenceByte bool) (*File, error){
	obj := &Object{
		Data: o.Data,
		Tags: o.Tags,

	}
	return obj.ToFile(true)

}

type Reservation struct{
	Namespace string
	reservation.Reservation
}

func NewReservation(namespace string, admin string, size float64, period int) (*Reservation, error){
	creationDate := time.Now()
	expirationDate := creationDate.AddDate(0, 0, period)

	uuid, err := utils.GenerateUUID(64)

	if err != nil{
		return nil, err
	}

	return &Reservation{namespace,
		reservation.Reservation{
			AdminId: admin,
			SizeReserved: size,
			SizeUsed: 0,
			ExpireAt: goraml.DateTime(expirationDate),
			Created: goraml.DateTime(creationDate),
			Updated: goraml.DateTime(creationDate),
			Id: uuid,
		}}, nil
}


func (s Reservation) Validate() error {

	return validator.Validate(s)
}

func (s Reservation) SizeRemaining() float64{
	return s.SizeReserved - s.SizeUsed

}


func (s Reservation) GetKey(config *Settings) string{
	return fmt.Sprintf("%s%s_%s", config.Namespace.Reservations.Prefix, s.Namespace, s.Id)
}

func (r Reservation) Save(db DB, config *Settings) error{
	key := r.GetKey(config)
	return db.Set(key, r.ToBytes())
}

func (r *Reservation) Get(db DB, config *Settings) (*Reservation, error){
	key := r.GetKey(config)
	v, err := db.Get(key)

	if err != nil{
		return nil, err
	}

	if v == nil{
		return nil, nil
	}

	r.FromBytes(v)
	return r, nil
}



func (s Reservation) ToBytes() []byte {
	/*
	-----------------------------------------------------------------
	SizeReserved| TotalSizeReserved |Size of CreationDate
	 8         |   8      |  2
	-----------------------------------------------------------------

	-----------------------------------------------------------------------
	Size of UpdateDate     |Size of ExpirationDate | Size ID | Size AdminID
 	    2                  |         2             |  2       |   2
	----------------------------------------------------------------------

	------------------------------------------------------------
	 CreationDate | UpdateDate | ExpirationDate | ID | AdminId
	------------------------------------------------------------

	*/

	adminId := s.AdminId
	aSize := int16(len(adminId))

	id := s.Id
	iSize := int16(len(id))

	created := []byte(time.Time(s.Created).Format(time.RFC3339))
	updated := []byte(time.Time(s.Updated).Format(time.RFC3339))
	expiration := []byte(time.Time(s.ExpireAt).Format(time.RFC3339))

	cSize := int16(len(created))
	uSize := int16(len(updated))
	eSize := int16(len(expiration))

	result := make([]byte, 26+cSize+uSize+eSize+aSize+iSize)

	copy(result[0:8], utils.Float64bytes(s.SizeReserved))
	copy(result[8:16], utils.Float64bytes(s.SizeUsed))

	binary.LittleEndian.PutUint16(result[16:18], uint16(cSize))
	binary.LittleEndian.PutUint16(result[18:20], uint16(uSize))
	binary.LittleEndian.PutUint16(result[20:22], uint16(eSize))
	binary.LittleEndian.PutUint16(result[22:24], uint16(iSize))
	binary.LittleEndian.PutUint16(result[24:26], uint16(aSize))

	//Creation Date size and date
	start := 26
	end := 26 + cSize
	copy(result[start:end], created)

	//update Date
	start2 := end
	end2 := end + uSize
	copy(result[start2:end2], updated)

	//ExpirationDate
	start3 := end2
	end3 := start3 + eSize
	copy(result[start3:end3], expiration)

	// ID
	start4 := end3
	end4 := start4 + iSize
	copy(result[start4:end4], []byte(id))

	// Admin ID
	start5 := end4
	end5 := start5 + aSize
	copy(result[start5:end5], []byte(adminId))
	return result
}

func (s *Reservation) FromBytes(data []byte) error{
	s.SizeReserved = utils.Float64frombytes(data[0:8])
	s.SizeUsed = utils.Float64frombytes(data[8:16])

	cSize := int16(binary.LittleEndian.Uint16(data[16:18]))
	uSize := int16(binary.LittleEndian.Uint16(data[18:20]))
	eSsize := int16(binary.LittleEndian.Uint16(data[20:22]))
	iSize := int16(binary.LittleEndian.Uint16(data[22:24]))
	aSize := int16(binary.LittleEndian.Uint16(data[24:26]))

	start := 26
	end := 26 + cSize

	cTime, err := time.Parse(time.RFC3339, string(data[start:end]))

	if err != nil{
		return err
	}

	start2 := end
	end2 := end + uSize

	uTime, err := time.Parse(time.RFC3339, string(data[start2:end2]))

	if err != nil{
		return err
	}

	start3 := end2
	end3 := end2 + eSsize

	eTime, err := time.Parse(time.RFC3339, string(data[start3:end3]))

	if err != nil{
		return err
	}

	start4 := end3
	end4 := start4 + iSize

	start5 := end4
	end5 := start5 + aSize


	s.Created = goraml.DateTime(cTime)
	s.Updated = goraml.DateTime(uTime)
	s.ExpireAt = goraml.DateTime(eTime)

	s.Id = string(data[start4:end4])
	s.AdminId = string(data[start5:end5])
	return nil
}




/*
	Token format
	-----------------------------------------------------------------------------------------------------------
	Random bytes |ReservationExpirationDateEpoch| namespace ID length| reservation ID length| namespaceID|resID
	    51           8                                2 bytes        |     2 bytes
	-----------------------------------------------------------------------------------------------------------
 */

func (s Reservation) GenerateTokenForReservation(db *Badger, namespaceID string)(string, error){
	nID := []byte(namespaceID)
	rID := []byte(s.Id)

	b := make([]byte, 63 + len(nID) + len(rID))

	r, err := utils.GenerateRandomBytes(51)

	if err != nil{
		return "", err
	}

	copy(b[0:51], r)

	epoch := time.Time(s.ExpireAt).Unix()
	binary.LittleEndian.PutUint64(b[51:59], uint64(epoch))

	nSize := len(nID)
	rSize := len(rID)

	binary.LittleEndian.PutUint16(b[59:61], uint16(nSize))
	binary.LittleEndian.PutUint16(b[61:63], uint16(rSize))

	start := 63
	end := 63 + nSize
	copy(b[start:end], nID)

	start = end
	end = start + rSize
	copy(b[start:end], rID)

	token, err := base64.StdEncoding.EncodeToString(b), err

	if err != nil{
		return "", err
	}
	return token, nil
}

/*
| Random bytes  | expirationEpoch  |Admin|Read |Write|Delete|user|
|---------------|------------------|-----|-----|-----|------|----|
| 51 bytes      | 8 bytes          |1byte|1byte|1byte|1byte|    |


 */
func (s Reservation) GenerateDataAccessTokenForUser(user string, namespaceID string, acl ACLEntry) (string, error){
	b := make([]byte, 60 + len(namespaceID) + len(user))

	r, err := utils.GenerateRandomBytes(51)

	if err != nil{
		return "", err
	}

	copy(b[0:51], r)
	epoch := time.Time(s.ExpireAt).Unix()
	binary.LittleEndian.PutUint64(b[51:59], uint64(epoch))
	copy(b[59:63], acl.ToBytes())
	copy(b[63:], []byte(user))
	token, err := base64.StdEncoding.EncodeToString(b), err

	if err != nil{
		return "", err
	}

	return token, nil
}

func (s *Reservation) ValidateReservationToken(token, namespaceID string) (string, error){
	bytes, err := base64.StdEncoding.DecodeString(token)

	if err != nil{
		return "", err
	}


	if len(bytes) < 63{
		return "", errors.New("Reservation token is invalid")
	}

	namespaceSize := int16(binary.LittleEndian.Uint16(bytes[59:61]))
	reseIdSize := int16(binary.LittleEndian.Uint16(bytes[61:63]))

	if len(bytes) < 63 + int(namespaceSize) + int(reseIdSize){
		return "", errors.New("Reservation token is invalid")
	}

	now := time.Now()
	expiration := time.Unix(int64(binary.LittleEndian.Uint64(bytes[51:59])), 0)

	if now.After(expiration){
		return "", errors.New("Reservation token expired")
	}

	start := 63
	end := 63 + namespaceSize
	namespace := string(bytes[start:end])

	if namespace != namespaceID{
		return "", errors.New("Reservation token is invalid")
	}

	reservation := string(bytes[end:end+reseIdSize])

	return reservation, nil
}

func (s Reservation) ValidateDataAccessToken(acl ACLEntry, token string) error{
	bytes, err :=  base64.StdEncoding.DecodeString(token)

	if err != nil{
		return err
	}
	if len(bytes) <= 63{
		return errors.New("Data access token is invalid")
	}
	now := time.Now()
	expiration := time.Unix(int64(binary.LittleEndian.Uint64(bytes[51:59])), 0)

	if now.After(expiration){
		return errors.New("Data access token expired")
	}

	tokenACL := ACLEntry{}
	tokenACL.FromBytes(bytes[59:63])

	// IS Admin
	if tokenACL.Admin{
		return nil
	}

	// HTTP action ACL requires missing permission granted for that user
	if (acl.Admin && !tokenACL.Admin) ||
		(acl.Read && !tokenACL.Read) ||
		(acl.Write && !tokenACL.Write) ||
		(acl.Delete && !tokenACL.Delete){
		return errors.New("Permission denied")
	}

	//tokenUser := string(bytes[63:])

	//if user != tokenUser{
	//	return errors.New("Invalid token for user")
	//}

	return nil

}


type StoreStatRequest struct{
	SizeAvailable float64 `json:"size_available" validate:"min=1,nonzero"`
}

type StoreStat struct {
	StoreStatRequest
	SizeUsed float64 `json:"size_used" validate:"min=1,nonzero"`
}

func (s StoreStatRequest) Validate() error {
	return validator.Validate(s)
}

func (s *StoreStat) ToBytes() []byte{
	bytes := make([]byte, 16)
	copy(bytes[0:8], utils.Float64bytes(s.SizeAvailable))
	copy(bytes[8:16], utils.Float64bytes(s.SizeUsed))
	return bytes
}

func (s *StoreStat) FromBytes(data []byte) error{
	s.SizeAvailable = utils.Float64frombytes(data[0:8])
	s.SizeUsed = utils.Float64frombytes(data[8:16])
	return nil
}

func (s StoreStat) Save(db DB, config *Settings) error{
	key := config.Store.Stats.Collection
	return db.Set(key, s.ToBytes())
}

func (s StoreStat) Exists(db DB, config *Settings) (bool, error){
	return db.Exists(config.Store.Stats.Collection)
}

func (s *StoreStat) Get(db DB, config *Settings) error{
	key := config.Store.Stats.Collection
	v, err :=  db.Get(key)
	if err != nil{
		return err
	}
	s.FromBytes(v)
	return nil
}

type Tag struct {
	Key   string `json:"key" validate:"regexp=^\w+$,nonzero"`
	Value string `json:"value" validate:"nonzero"`
}

func (s Tag) Validate() error {

	return validator.Validate(s)
}
