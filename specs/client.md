# 0-stor client

This spec is a bit different than other specs,
as at the time of writing this spec the 0-stor client already exists.
The aim of this spec isn't to rewrite the client from scratch,
but to simplify and improve it into a rock-solid 0-stor client.

Our design principles at hand that will guide is through the work to be done are:

+ Code should be as minimalistic and simple as possible;
+ Abstractions should only be used where required;
+ The client should be as parallel and streaming as possible and desired;

## Design

Currently the main public API consists out of one big API, Client.
This has to change and instead I recommend that we have the following 3 clients
(interfaces might be defined or not, but are given here just to describe the expected API)

`client/itsyouonline`:

```go
type Client interface {
    GetToken(namespace string, perm Permission) (string, error)

    SetNamespace(namespace string) error
    DeleteNamespace(namespace string) error

    SetPermission(namespace, userID string, perm Permission) error
    GetPermission(namespace, userID string) (Permission, error)
    DeletePermission(namespace, userID string, perm Permission) error
    
    Close() error
}
```

`client/meta`:

```go
// Keywords have been normalized,
// but not much else has changed.
type Client interface {
    SetMeta(key string, meta *Data) error
    GetMeta(key string) (*Data, error)
    DeleteMeta(key string) error
    
    Close() error
}

// Data used to be called Meta
type Data struct {
    Epoch     int64
    Key       []byte
    Chunks    []*ChunkMeta
    Previous  []byte
    Next      []byte
    ConfigPtr []byte
}
// ChunkData is new
type ChunkData struct {
    Key, Data []byte
    RefList []string
}
// ChunkMeta used to be called Chunk
type ChunkMeta struct {
    Size   uint64
    Key    []byte
    Shards []string
}
```

> ENCODING INTERMEZZO
>
> Right now we use Capn'Proto for the encoding/decoding of Metadata.
> However I would recommend we switch over to https://github.com/gogo/protobuf
> 
> gogoprotobuf performs much better than Capnp'Proto
> (and the standard protobuf library), checkout:
> 
> https://github.com/alecthomas/go_serialization_benchmarks
> 
> If we like it, we should also use it for our GRPC server,
> as it can also be used together with the standard GRPC library.
>
> Switching over to gogoprotobuf (for both our encoding and GRPC needs)
> also means that we'll only require one encoding tool/deps (proto),
> compared to the current 2 (proto+capnp).

`client/data`:

```go
// The Client, which used to live under `client/stor` now lives under `client/data`,
// and has been improved.
//
// The comments below only talk about the (param) signature changes and
// whether or not the method (type) is already supported.
// Regardless of that, all methods have changed name,
// to Normalize their name (have the keyword at the front)
// and to normalize the keyword used.
type Client interface {
    // SetObject works still like in old/current client.
    // Modified the signature though, but that's a client-side change only.
    SetObject(object Object) error

    // GetObject works still like in old/current client,
    // just has a different return param, but that's a client-side change only.
    GetObject(key []byte) (*Object, error)
    // This is a server-stream method,
    // and doesn't exist currently, will have to be supported by the server.
    GetObjectIterator(ctx context.Context, keys [][]byte) (<-chan ObjectResult)

    // DeleteObjects used to just take a single keys,
    // not it can take multiple keys, to save round-trips where needed,
    // is an unary method that will have to be supported by the server.
    DeleteObjects(key ...[]byte) (int, error)

    // Used to be called Check
    GetObjectStatus(key []byte) (ObjectStatus, error)
    // Method doesn't exist, similar to GetObjectStatus,
    // but as a server-stream method, which will have to be supported by the server.
    GetObjectStatusIterator(ctx context.Context, keys [][]byte) (<-chan ObjectStatusResult, error)

    // method already exists,
    // but will have to change it from an unary to a server-stream method
    ListObjects(ctx context.Context) (<-chan string, error)

    // Only signature changes, thus a small client-side change only.
    GetNamespace() (*Namespace, error)

    // Nothing changed compared to the old/current client.
    SetReferenceList(key []byte, refList []string) error
    // Method didn't exist yet,
    // will have to be added as unary function and supported by the server.
    GetReferenceList(key []byte) ([]string, error)
    // Nothing changed compared to the old/current client.
    AppendReferenceList(key []byte, refList []string) error
    // Nothing changed compared to the old/current client.
    DeleteReferenceList(key []byte, refList []string) error
}

// NOTE: these types should really be stand-alone types,
// we do not want to expose the proto types directly in our Client API,
// as those are only used for the GRPC client (which for now is the only choice, but still).
type Namespace struct {
    Label               string 
    ReadRequestPerHour  int64 
    WriteRequestPerHour int64 
    NrObjects           int64
}
type ObjectStatus uint8 // (enum with String() support)
type Object struct {
    key, Data []byte
    RefList []string
}
type ObjectStatusResult struct {
    Key []byte
    Status ObjectStatus
    Error error
}
type ObjectResult struct {
    Object Object
    Error error
}

// Cluster is used in the (main) public `client.Client`,
// in order to 
type Cluster interface {
    // NOTE (QUESTION FOR CHRISTOPHE):
    // > In the old/current 0-stor client design,
    // > the given shard does not have to be part of the known shard-list,
    // > it will simply accept any shard it receives (from a chunk config).
    // >> IS THIS OK?!?!?!
    GetClient(shard string) (Client, error)
    // GetRandomClient gets any cluster available in this cluster:
    // it only ever returns a client created from a shard
    // which comes from the pre-defined shard-list
    // (given at creation time of this cluster):
    GetRandomClient() (Client, error)

    // Dereference a client,
    // which was retrieved using `GetClient` or `GetRandomClient`.
    DerefClient(shard string) error

    // Close will panic,
    // if there is still a reference to a pooled client.
    Close() error
}
```

These are the 3 core clients:

+ `client/itsyouonline.Client`: Deals with IYO namespaces and permissions;
+ `client/metadata.Client`: Deals with metadata;
+ `client/data.Client`: Deals with data and is really only meant to be used in combination with an 0-stor server;

All 3 clients can be used separately, and in the case of the IYO
and metadata client that will probably also happen for some common use cases.

On top of that we also provide one main `client.Client` which will be the client
we really promote to use and is the one we expect people to use for sure.

`client.Client`:

```go
type Client interface {
    // There is no need to have a SetWithMeta and Set,
    // as the meta input is optional, and can be nil where you previously used Write as well.
    Set(key []byte, r io.Reader, refList []string, meta *meta.Meta) (*meta.Meta, error)
    SetWithLink(key, prevKey []byte, r io.Reader, refList []string, meta, prevMeta *meta.Meta) (*meta.Meta, error)

    // meta is optional again, just as with Set.
    // NOTE: meta _IS_ required in case there is no underlying `client/metadata.Client`.
    Get(key []byte, w io.Writer, meta *meta.Meta) error

    // meta is optional again, just as with Set.
    // NOTE: meta _IS_ required in case there is no underlying `client/metadata.Client`.
    Delete(key []byte, meta *meta.Meta) error

    // meta is optional again, just as with Set.
    // NOTE: meta _IS_ required in case there is no underlying `client/metadata.Client`.
    Repair(key []byte, meta *meta.Meta) (*meta.Meta, error)

    // The Traverse methods are only supported if there is an underlying `client/metadata.Client`.
    Traverse(ctx context.Context, startKey []byte, fromEpoch, toEpoch int64) (<-chan interface{}, error)
    TraversePostOrder(ctx context.Context, startKey []byte, fromEpoch, toEpoch int64) (<-chan interface{}, error)

    // used to be called Check
    // meta is optional again, just as with Set.
    // NOTE: meta _IS_ required in case there is no underlying `client/metadata.Client`.
    GetStatus(key []byte, meta *meta.Meta) (ObjectStatus, error)

    // meta is optional once again...
    // NOTE: meta _IS_ required in case there is no underlying `client/metadata.Client`.
    SetReferenceList(key []byte, refList []string, meta *meta.Meta) error
    GetReferenceList(key []byte, meta *meta.Meta) ([]string, error)
    AppendReferenceList(key []byte, refList []string, meta *meta.Meta) error
    DeleteReferenceList(key []byte, refList []string, meta *meta.Meta) error

    Close() error
}

// Each returned result starts with an `TraverseElementHeader`,
// and than continues with one or multiple TraverseData.
// An element ends when the channel is closed or when a new `TraverseElementHeader` has been received.
// At any point a TraverseError could be returned, in case an error has been occured,
// if this happens the channel will be closed and the Traversel will be stopped immediately.
//
// This means that the user will have to use a switch case such as:
//
//    switch input.(type) {
//    case TraverseElementHeader:
//       // complete/process/use previous object (if any)
//       // start new object
//    
//    case TraverseData:
//       // use/append data for current element,
//       // ideally we can write directly to an `io.Writer` (e.g. a file)
//    
//    case TraverseError:
//       // Something went wrong, cleanup, and stop it,
//       // as soon as a value of this type has been received,
//       // the Traverse channel is automatically closed
//    }
//
// This means that using this function is a tiny bit more difficult,
// but the advantage is that we don't have to block on a per-object basis,
// which would be very non-performent when walking over a group of files,
// and even for a single performent that would not be ideal.
// Instead, using this traverse design,
// we can write/process the object as it comes in.
type TraverseError struct {
    Error error
}
type TraverseElementHeader struct{
    key []byte
    Meta  *meta.Meta
    RefList []string
}
type TraverseData struct{
    Data []byte
}
```

The main (high-level) client makes use of one optional `client/metadata.Client`, `client/itsyouonline.Client`
and one required `client/data.Client` per shard.

In contrast with the current/old 0-stor client, we do not expose the IYO and metadata client's features directly,
if you need those, you'll have to use the clients directly,
there is no point in us wrapping an existing interface just for the sake of wrapping.

When the `client.Client` has no `client/metadata.Client`, you _have_ to manage the metadata yourself.
You'll also _have_ to give the metadata to all functions except `Set`,
as information such as the shards information will otherwise be unknown.

All client interfaces will actually be explicitly defined, except for the `client.Client`,
that last (and most high-level) one will only be available as a public Struct.

Even though we expose the other client interfaces, the underlying concrete types
will still be public and the constructors will still return the actual concrete type rather than the interface.

As a final note, also note how all keywords of the clients' methods have been normalized.
There are now only 5 normal keywords: Set, Get, Delete, Append, Exist.
The amount of specialized keywords remained more or less stable: Traverse, List, and Repair.

### Creating the Client

Creating a client would become as simple as calling the following constructor:

```go
// NewClient creates a new (high-level) 0-stor client.
// If `blockSize <= 0`, no chunking logic is supported and used (when writing data).
// This is however not recommended, as it would mean big files have to be
// loaded completely in memory for processing and storage.
// 
// JobCount defines how much (processor) "jobs" or pipelines we run in parallel.
// Note that we'll have more goroutines open than this count,
// as one job/pipeline has several goroutines. On top of that we double the amount of jobs,
// at the end of the pipeline, for storage.
// JobCount is to be concidered optional, $NUM_CPU is the default value used,
// in case this value is zero/nil/negative.
func NewClient(blockSize, jobCount int, components Components) (*Client, error)

// Components used to create a Client
type Components struct {
    // Required!
    StorageCluster data.Cluster

    // Optional,
    // A nopProcessor will be used if not defined.
    DataProcessor processing.Processor

    // Optional,
    // Some Client Functions might not be supported (and return an error),
    // if this client is not defined though (e.g. Walk).
    MetaClient meta.Client

    // Optional,
    // although required if (one of) the used 0-stor servers require it.
    IYOClient itsyouonline.Client

    // Optional,
    // SHA256 will be used if not defined.
    Hasher crypto.Hasher
}
```

Note that there is also an (even) easier (and standard) way of creating a Client.
See the [zstor config](#zstor-config) chapter for more information.

### Components

Package: `/client/components`

The root package doesn't contain anything.

This package contains following sub packages:

+ [`/client/components/storage`](#storage): final storage type(s) for storing (a) chunk(s)
    + (internally these use the `/client/data.Cluster+Client` types for the actual storage of the chunk(s));
+ [`/client/components/processing`](#processing): processing (crypto/compression) components;
+ [`/client/components/crypto`](#crypto): non-processing crypto components (e.g. hashers);

Here is how all these components are used together:

```
client.Client.Write (With Chunking)
+------------------------------------------------------------------+
| +---------+                                                      |
| | Chunker +---> Processor.Write +   +-----------+                |
| |   +     |          ...        +--->  bufchan  +--------+       |
| | Hasher  +---> Processor.Write +   +-----------+        |       |
| +---------+                                              |       |
|                 +-----------------+----------------------+       |
|                 v                 v                      v       |
|           Storage.Write     Storage.Write    ...   Storage.Write |
|                 +                 +                      +       |
|                 |                 |                      |       |
|                 |   (ChunkMeta)   |      (ChunkMeta)     |       |
|                 |                 |                      |       |
|         +-------v-----------------v----------------------v-----+ |
|         |                 meta.Meta (Node)                     | |
|         +------------------------------------------------------+ |
+------------------------------------------------------------------+
```

Notes:

+ I would recommend to have more `storage.Write` goroutines
  than `Processor.Write` goroutines (twice as much?),
  for the simple reason that the latter will require more time per chunk;
+ The processed data chunks should be buffered,
  such that processing can continue even when experiencing network delays;

```
client.Client.Write (No Chunking)
+-----------------------------------------------------------------------+
| io.Reader+Hasher +-> Processor.Write +-> Storage.Write +-> meta.Meta  |
+-----------------------------------------------------------------------+
```

Notes:

When no chunking is involved, there is no need to spawn goroutines and channels,
as these wouldn't be utilized anyhow, thus for non-chunked data
the sequential version outperforms the parallel version.

```
client.Client.Read (With Multiple Chunks)
+--------------------------------------------------------------------------------+
| +--------------+                                                               |
| |[]*ChunkMeta  +----> Storage.Read +-+                                         |
| |     to       |                     |                                         |
| |chan ChunkMeta+----> Storage.Read +-+     +---------+                         |
| +--------------+                     +-----> channel +---------------+         |
|            |   ...                   |     +---------+               |         |
|            |                         |          |         ...        |         |
|            +--------> Storage.Read +-+          |                    |         |
|                                         +-------v--------+  +--------v-------+ |
|                                         | Processor.Read |  | Processor.Read | |
|                                         |       +        |  |       +        | |
|                                         |   Hash/Data    |  |   Hash/Data    | |
|                                         |   Validation   |  |   Validation   | |
|                                         +----------------+  +----------------+ |
|                                                 |                    |         |
|                                                 |                    |         |
|                                             +---v--------------------v---+     |
|                                             |                            |     |
|                                             |     Data Composer          |     |
|                           io.Writer <-------+ (with internal buffer)     |     |
|                         (input param)       |                            |     |
|                                             +----------------------------+     |
+--------------------------------------------------------------------------------+
```

Notes:

The data composer (and its internal buffer) is used,
to ensure we write the raw chunks in the correct order to the io.Writer.

```
client.Client.Read (Just a single chunk)
+-------------------------------------------------------------+
|                                    +----------------------+ |
| meta.ChunkMeta +-> storage.Read +--> Processor.Read +     | |
|                                    | Hash/Data Validation | |
|                                    +---------------------+  |
|                                                |            |
|                                io.Writer <-----+            |
+-------------------------------------------------------------+
```

Notes:

When some data has only one chunk, we can simply read, process and validate
in a plain old sequential fashion. Goroutines wouldn't be utilized in this case,
and spawning them (+ the infrastructure around it), would only cause overhead,
hurting the performance of this case.

#### Storage

Package: `/client/components/storage`

In it we define the single, distributed and replicated storage logic,
which is in the current/old design part of the Client itself.

`client/components/storage.Storage`:

```go
type Storage interface {
    Write(data ChunkData) (ChunkMeta, error)
    Read(meta ChunkMeta) (ChunkData, error)
    Repair(meta ChunkMeta) (ChunkMeta, error)
}
```

Will be used to implement:

+ RandomStorage: stores it on a random storage server;
+ DistributeStorage: erasures encodes the chunk prior to storage
  (using RandomStorage internally);
+ ReplicateStorage: replicates a chunk N amount of times over the available servers
  (using RandomStorage internally);

#### Processing

The `client/components/processing` package will be
the redesigned (subset) version of what we currently call the `client/lib` package.

The main type of that package is `client/components/processing.Processor`:

```go
// methods block, so they should be spawned on their own goroutine,
// you can call each method as much and simultaneously as you want,
// it will create a different instance for each call.
type Processor interface {
    Read(ctx context.Context, in chan<- []byte, out <-chan []byte) error
    Write(ctx context.Context, in chan<- []byte, out <-chan []byte) error
}
```

Users will be able to define their own processor,
but we ship following (std) processors:

+ `client/api/processing/crypto`: `AES`;
+ `client/api/processing/compress`: `Snappy`, `GZip`, `XZ`, `LZ4`;

As you can see, each type of processor is also a reverse processor,
such that you can read the data that has been written.

Processors can also be chained, and we provide a utility `Processor` for this.

`client/components/processing.Chain`:

```go
type Chain struct {
    // guaranteed to have at least one processor by constructor
    processors []Processor
}

type (chain Chain) Write(ctx context.Context, in chan<- []byte, out <-chan []byte) error {
    group, ctx := errgroup.WithContext(ctx)
    var processor Processor

    index, limit := 0, len(chain.processors)-1
    for {
        processor = chain.processors[index]
        group.Go(func() error {
            return processor.Write(ctx, in, out)
        }
        if index == limit {
            break
        }
        index++
        in = out
        out = make(chan []byte, 1)
    }

    return group.Wait()
}

type (chain Chain) Read(ctx context.Context, in chan<- []byte, out <-chan []byte) error {
    group, ctx := errgroup.WithContext(ctx)
    var processor Processor

    index := len(chain.Processors)-1
    for {
        processor = chain.processors[index]
        group.Go(func() error {
            return processor.Read(ctx, in, out)
        }
        if index == 0 {
            break
        }
        index--
        in = out
        out = make(chan []byte, 1)
    }

    return group.Wait()
}
```

Chained processors will look like a single Processor to the `client.Client`.

Just like a single processor, a chain has both a read and write function.

#### Crypto

In the crypto package we'll implement any crypto-components,
which isn't a (chunk) processor.

Currently it will only contain the Hasher code.

We name this package `crypto`, rather than `hash,
as we want to underline that we only support and recommend
cryptographic hash algorithms, for security reasons.

Normal (non-cryptographic) hashing algorithms might leak
information about the content of the chunks stored,
and are therefore not implemented, nor recommended.

Package: `/client/components/crypto`

```go
// Hasher defines the interface of a crypto-hasher
type Hasher interface {
    HashBytes(data []byte) (hash []byte)
}
```

This hasher is used to compute the Keys for (data-) chunks,
stored in the 0-stor server, using the `client.Client` (Write).

The following Hashers will be supported by (as in shipped with) this library:

+ `blake2b`: using `github.com/minio/blake2b-simd`;
+ `sha256`: using `crypto/sha256` (default);

The original library supported (but didn't use `md5`) as well.
In the new design we drop this hasher, as its crypto is broken and
therefore this hasher shouldn't be used any longer.

The user can obviously also define its own (non-crypto) hasher,
however only the std hashes listed above will be available,
when creating the `client.Client` using the configuration approach.

#### Processor Chain Configuration

The standard (complete) Processor Chain can be created with a single call,
by using the standard Configuration object:

```go
type Configuration struct {
    // Compressor Processor Configuration,
    // disabled by default.
    Compression struct {
        // Default: NopCompress
        Type compress.Type `json:"type" yaml:"type"`

        // NOTE: Not all compression types might support this one,
        // so see this configuration more as suggestive
        // Default: DefaultCompression (other: BestSpeed/BestCompression)
        Mode compress.Mode `json:"mode" yaml:"mode"`
    } `json:"compression" yaml:"compression"`

    // Encrypter Processor Configuration,
    // disabled by default (enable it by giving a PrivateKey)
    Encrypton struct {
        // Private
        // Key lengths supported: 0, 16, 24 and 32 bytes
        // (this will turn into no-encryption, AES-128, AES-192 and AES-256)
        PrivateKey string `json:"private_key" yaml:"private_key"`
        // For now we don't support Asymmetric algorithms,
        // so there is no need for a public key.
    } `json:"encryption" yaml:"encryption"`

    // Distributor Processor Configuration,
    // enabled only when both Count and Redundancy
    // are more then 0.
    // Disabled by default.
    Distribution struct {
        // Number of data block to create during distribution
        Count int   `json:"data" yaml:"data"`
        // Number of parity block to create during distribution
        Redundancy int `json:"parity" yaml:"parity"`
    } `json:"distribution" yaml:"distribution"`

    // Replicator Processor Configuration,
    // disabled by default.
    Replication struct {
        // Number of replication to create when writting.
        // Disabled when number is 0 (default) or less.
        // When the number is more then 1, a replicator will be created with this configuration.
        Count int `json:"count" yaml:"count"`
    } `json:"replication" yaml:"replication"`
}
```

Using this configuration you would be able to create a standard chain using following function:

```go
// An error would be returned if the config is in some way invalid,
// or if for any other reason a component construction failed.
func NewChainFromConfig(cfg Configuration) (*Chain, error)
```

### zstor config

For the zstor we'll use a composition of configurations and some extra properties:

```go
type Configuration struct {
    // config defined in our client/itsyouonline package,
    // and used to create a `client/itsyouonline.Client`.
    // Required only when the used Data servers require it.
    IYO itsyouonline.Config `json:"iyo" yaml:"iyo"`

    // Hasher Configuration (used for chunk-key generation),
    // enabled by default (and cannot be disabled).
    Hasher struct {
        // Default: sha256
        // Others available: blake2b
        Type crypto.HashType `json:"type" yaml:"type"`
    } `json:"hasher" yaml:"hasher"`
    
    // Data Processing (chain) configuration (Optional)
    Processing processing.Config `json:"processing" yaml:"processing"`

    Shards struct {
        // Addresses to zstor servers used to store date (Required)
        Data []string `json:"data" yaml:"data"`
        // Addresses of the etcd cluster (Optional)
        Meta []string `json:"meta" yaml:"meta"`
    } `json:"shards" yaml:"shards"`

    // If the data written to the store is bigger then BlockSize, the data is splitted into blocks of size BlockSize. (Optional)
    // set to 0 to never split data (or don't define it all, as 0 is the default).
    // It is recommended to set a non-0 small block-size,
    // as otherwise the entire data (file) will have to be loaded in memory,
    // for processing and storage.
    BlockSize int `json:"block_size" yaml:"block_size"`

    // JobCount defines how much (processor) "jobs" or pipelines we run in parallel.
    // Note that we'll have more goroutines open than this count,
    // as one job/pipeline has several goroutines. On top of that we double the amount of jobs,
    // at the end of the pipeline, for storage.
    // This value is optional, $NUM_CPU is the default value used,
    // in case this value is zero/nil/negative.
    JobCount int `json:"job_count" yaml:"job_count"`
}
```

This configuration can also be used by another Golang package to create a client using:

```go
func NewClientFromConfig(cfg Configuration) (*Client, error)
```

That same function is also the one used by zstor to create the client.

### Required changes in the 0-stor server

+ Update method names (as we normalized keywords and renamed methods in that same line of thinking);
+ Support missing methods:
    + `GetObjectIterator`;
    + `GetObjectStatusIterator`;
    + `GetReferenceList`;
+ Change `DeleteObject` to `DeleteObjects` (and support return param);
