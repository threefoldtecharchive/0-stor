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

// Package pipeline is used to write/read content from/to a datastor cluster.
//
// It allows to do this using a pipeline model,
// where data is turned into one or multiple datastor objects,
// and can also be optionally processed prior to storage,
// in a random or distributed manner.
//
// Data written to the storage can be split, using a data splitter,
// into multiple smaller, "fixed-size", blocks.
// No padding will be applied should some (tail) data be too small
// to fit in the fixed-sized buffer. While this data splitting is optional,
// it is recommended, as it will allow you to write and read large files,
// without splitting the data up, this would be impossible,
// due to the fact that for processing all this data has to be read into
// memory of both the client and server, prior to storage in the database.
//
// Each object has a key, which identifies the data.
// In this pipeline model, the key is generated automatically,
// using a cryptographic hasher. Ideally these keys are generated
// as signatures, as to proof ownership,
// but checksum-based keys are supported as well, and are in fact the default.
// This checksum/signature is generated using the raw (split) data slice,
// as it is prior to being processed. When content is read back,
// this key is also validated, as an extra validation (on top of the checksum provided by zstordb),
// as to ensure our data is the one we expect. For this reason in specific,
// it is recommended to use signature-based key generation when possible.
// See the 'crypto' package for more information about this hashing logic,
// and read its documentation especially if you are planning to provide your own hasher.
//
// Data can be processed, which in the context of the pipeline
// means that it can be compressed and/or encrypted prior to storage,
// and decrypted and/or decompressed when the content is read back again.
// For this we make use of the 'processing' sub-package.
// This package defines the interfaces, types and logic used for processing data,
// but users can plugin their own compression/encryption processors as well,
// should this be desired. See the 'processing' package for more information.
//
// Finally when the data is ready for storage, which is the case once its processed and has a key,
// it will be stored using the 'storage' subpackage, into a zstordb cluster of choice.
// An object can simply be stored on a random available shard,
// or it can be distributed using replication or erasure coding.
// While the default is the simple random storage, it is recommended to opt-in
// for erasure-code distribution, as it will give your data
// the greatest resilience offered by this package.
// See the 'storage' package for more information.
//
// The easiest way to create a pipeline, is using the provided Config struct.
// Using a single configuration, and a pre-defined/created cluster,
// you are able that way to create an entire pipeline, ready for usage.
package pipeline
