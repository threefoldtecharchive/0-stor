# 0-stor daemon

## Goal
Giving the client the option to become a daemon with a e.g. grpc interface so that
e.g. python can instruct the daemon to manage data, permissions and namespaces.

We do this to prevent to have to re-implement all the client logic in every language we use.

### future feature
In a second phase we want to add a caching and autodiscovery feature to the deamon so it could be used as some kind of IPFS replacement.

## Requirements:
- The daemon will be part of the normal client, using the same configuration.
- The daemon is spawned by running a seperate command (`zstor daemon`).
- Simple interface, probably grpc. Grpc has a great number of supported language, that should give us the flexibilty we need.
- There is no authentication needed between proxy user and the proxy because the authentication is done in the server side and the proxy is supposed to run locally and thus the user can be trusted.
- IYO commands:
  - CreateJWT(namespace, permission)
  - CreateNamespace(namespace)
  - DeleteNamespace(namespace)
  - GivePermission(namespace, userID, permission)
  - RemovePermission(namespace, userID, permission)
  - GetPermission(namespace, userID)
  
- Object Commands:
  - Write(key/meta, value, referenceList, optional prevKey/prevMeta): write value to given key/meta with optionally specified reference list. If optional prevKey/prevMeta is specified, we build metadata linked list.
    We use this command to write a not so big data. If data is considered big, we should use `WriteStream` or `WriteFile`
    
  - WriteStream(key/meta, value stream, referenceList, optional prevKey/prevMeta): same as `Write` but receive stream of value instead of value. We use this command when we have stream of data which could be from external source like network or any reader. 
  - WriteFile(key/meta, file path, referenceList, optional prevKey/prevMeta): same as `Write` but receive file path instead of value. We use this command to store a file to 0-stor server and don't want the overhead of `WriteStream`. The file also need to be readable from the proxy for this call to work.
  - Read(key/meta) (value, referenceList): read object with given key/meta. We use this command when we expect the value to be not so big.
  - ReadStream(key/meta) (value stream): Same as `Read` but returns stream of value. We use this command when we expect the value to be big.
  - ReadFile(key/meta, file path): Same as `Read` but write the returned value to the given file path directly. The file need to be writable from the proxy for this call to work.
  - Delete(key/meta): Delete object with given key/meta.
  
- Walk commands
  - Walk(startKey, startEpoch, endEpoch) (stream of object): Walk over the metadata.

- Reference List:
  - AppendReferenceList(key/meta, referenceList)
  - RemoveReferenceList(key/meta, referenceList)
 
- Object check and Repair:
  - Check(key): check that object with given key is not corrupted
  - Repair(key): repair object with the given key

