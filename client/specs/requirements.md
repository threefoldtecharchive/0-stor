# 0-stor-lib

This library is composed of multiple small components.
Components can be chained to create a pipeline which will process data as it goes though.

The Client can configure the pipeline as desired,
how it's configured will depend on the specific use case.

The components shipped with this library are:
- chunker : splits data into smaller blocks of a fixed size
- compressor : compress or decompress the input
- encrypter : crypt/decrypt the input
- hasher : generate different hashes
- replicator : replicate one input onto multiple outputs
- erasure coder: split and replicate data over multiple outputs as to provide data redundancy

## Compoments in detail

This description of the components are written for a write case.
The opposite flow is supported as well, as to be able to read the data back.

### Chunker

Creation of a chunker takes the chunk size.
It returns an iterator that will yield a new chunk of data of the choosen size.
The block can then be sent to the rest of the pipeline

### Compressor

Creation of the compresser takes the type of compression used, supported:
- snappy
- lz4
- gzip

Some compression algorithms might require some extra configuration,
to be defined while creating the compressor.

### Encrypter

Supports both symetric and asymetric encryption/decryption.
Creation of an encrypter takes the required key(s).

### Hasher

Creation of a hasher takes the type of hashing algorithm used, supported:
- sha256
- blake2
- md5

The hasher compoment is a bit different from the other compoment since its outputs are not supposed to be sent into the pipeline. Instead it generate the hash of the input data it gets and send the hash into a variable external to the pipeline to be reused or even resend into the pipeline as an argument to other compoments.

### Replicator

A replicater is created by taking one input and specifying multiple outputs.
All the data that comes in is replicated on all the configured outputs.

### Erasure coder

Creation takes the erasure coding policy (k and n) and a list of outputs where to distribute the generated n blocks.  
Input data is splitted into n blocks, and then written to the ouputs.
This component is most probably going to be the last one of the pipeline.
