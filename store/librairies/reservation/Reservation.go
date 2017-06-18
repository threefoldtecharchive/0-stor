package reservation

import (
	"github.com/zero-os/0-stor/store/goraml"
	"gopkg.in/validator.v2"
	"time"
	"encoding/binary"
)

type Reservation struct {
	AdminId      string          `json:"adminId" validate:"regexp=^[a-zA-Z0-9]+$,nonzero"`
	Created      goraml.DateTime `json:"created" validate:"nonzero"`
	ExpireAt     goraml.DateTime `json:"expireAt" validate:"nonzero"`
	Id           string          `json:"id" validate:"regexp=^[a-zA-Z0-9]+$,nonzero"`
	SizeReserved int64           `json:"sizeReserved" validate:"min=1,multipleOf=1,nonzero"`
	SizeUsed     int64           `json:"sizeUsed" validate:"min=1,nonzero"`
	Updated      goraml.DateTime `json:"updated" validate:"nonzero"`
}

func (s Reservation) Validate() error {

	return validator.Validate(s)
}

func (s Reservation) ToBytes() []byte{
	/*
	-----------------------------------------------------------------
	SizeReserved| SizeUsed |Size of CreationDate
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

	result := make([]byte, 26 + cSize + uSize + eSize + aSize + iSize)

	binary.LittleEndian.PutUint64(result[0:8], uint64(s.SizeReserved))
	binary.LittleEndian.PutUint64(result[8:16], uint64(s.SizeUsed))
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
	start5 := end3
	end5 := start5 + aSize
	copy(result[start5:end5], []byte(adminId))
	return result



}

func (s *Reservation) FromBytes(data []byte) error{
	s.SizeReserved = int64(binary.LittleEndian.Uint64(data[0:8]))
	s.SizeUsed = int64(binary.LittleEndian.Uint64(data[8:16]))

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