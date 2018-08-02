/*
 * Copyright (C) 2017-2018 GIG Technology NV and Contributors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package proto

import (
	"github.com/threefoldtech/0-stor/client/metastor/metatypes"
)

// MarshalMetadata returns the gogo-proto encoding of the metadata parameter.
// It is important to use this function with the `UnmarshalMetadata` function of this package.
func MarshalMetadata(md metatypes.Metadata) ([]byte, error) {
	s := Metadata{
		Namespace:      md.Namespace,
		Key:            md.Key,
		TotalSize:      md.Size,
		StorageSize:    md.StorageSize,
		ChunkSize:      md.ChunkSize,
		CreationEpoch:  md.CreationEpoch,
		LastWriteEpoch: md.LastWriteEpoch,
		PreviousKey:    md.PreviousKey,
		NextKey:        md.NextKey,
		UserDefined:    md.UserDefined,
	}

	if length := len(md.Chunks); length > 0 {
		s.Chunks = make([]Chunk, length)
		for index, input := range md.Chunks {
			chunk := &s.Chunks[index]
			chunk.SizeInBytes = input.Size
			chunk.Hash = input.Hash
			if length := len(input.Objects); length > 0 {
				chunk.Objects = make([]Object, length)
				for index, input := range input.Objects {
					object := &chunk.Objects[index]
					object.Key = input.Key
					object.ShardID = input.ShardID
				}
			}
		}
	}

	return s.Marshal()
}

// UnmarshalMetadata parses the gogo-proto encoded metadata
// and stores the result in the value pointed to by the metadata parameter.
// It is important to use this function with a the `MashalMetadata` function of this package.
func UnmarshalMetadata(b []byte, md *metatypes.Metadata) error {
	if b == nil {
		panic("no bytes given to unmarshal to metadata")
	}
	if md == nil {
		panic("no metadata given to unmarshal to")
	}

	var s Metadata
	err := s.Unmarshal(b)
	if err != nil {
		return err
	}

	md.Namespace = s.Namespace
	md.Key = s.Key
	md.Size = s.TotalSize
	md.StorageSize = s.StorageSize
	md.ChunkSize = s.ChunkSize
	md.CreationEpoch = s.CreationEpoch
	md.LastWriteEpoch = s.LastWriteEpoch
	md.NextKey = s.NextKey
	md.PreviousKey = s.PreviousKey
	md.UserDefined = s.UserDefined

	if length := len(s.Chunks); length > 0 {
		md.Chunks = make([]metatypes.Chunk, length)
		for index, input := range s.Chunks {
			chunk := &md.Chunks[index]
			chunk.Size = input.SizeInBytes
			chunk.Hash = input.Hash
			if length := len(input.Objects); length > 0 {
				chunk.Objects = make([]metatypes.Object, length)
				for index, input := range input.Objects {
					object := &chunk.Objects[index]
					object.Key = input.Key
					object.ShardID = input.ShardID
				}
			}
		}
	}

	return nil
}
