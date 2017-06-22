package main

import (
	"gopkg.in/validator.v2"
)

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

	copy(b[0:8], Float64bytes(s.SpaceAvailable))
	copy(b[8:16], Float64bytes(s.SpaceUsed))
	copy(b[16:], nsc)

	return b
}

func (s *Namespace) FromBytes(data[]byte){
	s.SpaceAvailable = Float64frombytes(data[0:8])
	s.SpaceUsed = Float64frombytes(data[8:16])

	nsc := NamespaceCreate{}
	nsc.FromBytes(data[16:])

	s.NamespaceCreate = nsc
}
