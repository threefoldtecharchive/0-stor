# What is 0-stor

- 0-stor is a simple object storage where you can write files to and reas files from in an efficient manner.
- It's designed to leverage the power of [Badger][badger], a key value store that works very fast on SSD hard disks. You can read more about Badger [here][badger_post].
- 0-stor use [GRPC][grpc] as the communication protocol between server and client.
- 0-stor depends on [IYO][iyo] for:
    - Authentication & Authorization.
    - Every 0-stor collection is tied to an existing [IYO][iyo] organization, and thus there is a convention that 0stor collection names **MUST** follow:
        - 0-stor collection names convention is `{organization_name}_0stor_{namespace}`, the 
         `0stor` part is fixed and mandatory.
        - That says you **MUST** have `{organization_name}.0stor.{namespace}` sub organization on [IYO][iyo].
        - In most cases it's the responsibility of the client you are using to create proper [IYO][iyo] sub organizations for you
        provided that you have a valid organization


- [IYO][iyo] and how it's used for Authorization:
    - Assume we have an [IYO][iyo] organization called `myth`.
    - Now if you want to put an object in 0stor, the collection name must be `myth_0stor_{any_suffix}`
      - i.e `myth_0stor_namespace2`
      - meaning you must have a sub organization `myth.0stor.namespace2` in [IYO][iyo];
      - also you must have these sub organizations:
        - `myth.0stor.namespace2.read` (read-only access);
        - `myth.0stor.namespace2.write` (write-only access);
        - `myth.0stor.namespace2.delete` (delete-only-delete access);
          ![IYO organization][iyo_example]
    - The reason we have these different sub organizations is because we generate [JSON Web Tokens (JWTs)][jwt] based on them:
        - if a user is member of `myth.0stor.namespace2.read` then user has only read access to `myth.0stor.namespace2` and cannot write or delete;
        - if a user is member of `myth.0stor.namespace2.write` then user has only write access to `myth.0stor.namespace2` and cannot read or delete;
        - if a user is member of `myth.0stor.namespace2.delete` then user has only delete access to `myth.0stor.namespace2` and cannot read or write;

[iyo]: https://itsyou.online/
[jwt]: https://jwt.io/
[iyo_example]: assets/intro_iyo_namespace_example.png
[badger]: https://github.com/dgraph-io/badger
[badger_post]: https://open.dgraph.io/post/badger/
[grpc]: https://grpc.io/
