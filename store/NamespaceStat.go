package main

import (
	"gopkg.in/validator.v2"
	"encoding/binary"
	"time"
	"math"
	"github.com/zero-os/0-stor/store/librairies/reservation"
	"github.com/zero-os/0-stor/store/goraml"
)

type NamespaceStat struct {
	NrObjects      int64 `json:"NrObjects" validate:"nonzero"`
	RequestPerHour int64 `json:"requestPerHour" validate:"nonzero"`
}

func (s NamespaceStat) Validate() error {

	return validator.Validate(s)
}

type Stat struct{
	NamespaceStat
	reservation.Reservation
	NrRequests int64
}

func NewStat() *Stat{
	/*
		Newly created namespaces has 0 MBs size and expired - so it's not accessible
	 */
	now:= time.Now()

	return &Stat{
		NamespaceStat: NamespaceStat{
			NrObjects: 0,
			RequestPerHour: 0,
		},
		Reservation: reservation.Reservation{
			Created: goraml.DateTime(now),
			Updated: goraml.DateTime(now),
			ExpireAt: goraml.DateTime(now),
			SizeReserved: 0,
			SizeUsed: 0,
			AdminId: "",
		},

	}
}

func (s *Stat) toBytes() []byte{
	/*
	-----------------------------------------------------------------
	NrObjects|NrRwquests|SizeReserved| SizeUsed |Size of CreationDate
	   8     |   8      |  8         |   8      |  2
	-----------------------------------------------------------------

	--------------------------------------------------------------
	Size of UpdateDate     |Size of ExpirationDate |  CreationDate
	    2                  |         2             |
	--------------------------------------------------------------

	----------------------------------------------
	UpdateDate | ExpirationDate | AdminId
	----------------------------------------------

	*/
	adminId := s.Reservation.AdminId

	created := []byte(time.Time(s.Reservation.Created).Format(time.RFC3339))
	updated := []byte(time.Time(s.Reservation.Updated).Format(time.RFC3339))
	expiration := []byte(time.Time(s.Reservation.ExpireAt).Format(time.RFC3339))

	cSize := int16(len(created))
	uSize := int16(len(updated))
	eSize := int16(len(expiration))

	result := make([]byte, 38 + cSize + uSize + eSize)

	binary.LittleEndian.PutUint64(result[0:8], uint64(s.NrObjects))
	binary.LittleEndian.PutUint64(result[8:16], uint64(s.NrRequests))
	binary.LittleEndian.PutUint64(result[16:24], uint64(s.Reservation.SizeReserved))
	binary.LittleEndian.PutUint64(result[24:32], uint64(s.Reservation.SizeUsed))
	binary.LittleEndian.PutUint16(result[32:34], uint16(cSize))
	binary.LittleEndian.PutUint16(result[34:36], uint16(uSize))
	binary.LittleEndian.PutUint16(result[36:38], uint16(eSize))


	//Creation Date size and date
	start := 38
	end := 38 + cSize
	copy(result[start:end], created)

	//update Date
	start2 := end
	end2 := end + uSize
	copy(result[start2:end2], updated)

	//ExpirationDate
	start3 := end2
	end3 := start3 + eSize
	copy(result[start3:end3], expiration)

	//AdminId (variable string too)
	copy(result[end3:], []byte(adminId))

	return result
}

func (s *Stat) fromBytes(data []byte) error{
	s.NrObjects = int64(binary.LittleEndian.Uint64(data[0:8]))
	s.NrRequests = int64(binary.LittleEndian.Uint64(data[8:16]))
	s.Reservation.SizeReserved = int64(binary.LittleEndian.Uint64(data[16:24]))
	s.Reservation.SizeUsed = int64(binary.LittleEndian.Uint64(data[24:32]))

	cSize := int16(binary.LittleEndian.Uint16(data[32:34]))
	uSize := int16(binary.LittleEndian.Uint16(data[34:36]))
	eSsize := int16(binary.LittleEndian.Uint16(data[36:38]))


	start := 38
	end := 38 + cSize

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

	s.Reservation.Created = goraml.DateTime(cTime)
	s.Reservation.Updated = goraml.DateTime(uTime)
	s.Reservation.ExpireAt = goraml.DateTime(eTime)

	s.RequestPerHour = int64(float64(s.NrRequests) / math.Ceil(time.Since(cTime).Hours()))
	s.Reservation.AdminId = string(data[end3:])

	return nil
}
