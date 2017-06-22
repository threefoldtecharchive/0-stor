Internal keys
====================


Internal Objects Formats
========================

**ACLEntry**

| Admin  | Read  | Write  | Delete  |
|--------|-------|--------|---------|
| 1 byte | 1 byte| 1 byte | 1 byte  |


**ACL**

| ACLEntry  | Id      |
|-----------|---------|
| 4 bytes   | variable|

**NamespaceCreate**

|label size   | ACL[] length |Label   |ACL[0]  Size | ACL[0] |
|-------------|--------------|--------|-------------|--------|
| 2 bytes     | 2 bytes      |        |  2 bytes    |        |


**NameSpace**

| SpaceAvailable  | SpaceUsed  |NamespaceCreate  |
|-----------------|------------|-----------------|
| 8 bytes         | 8 bytes    |                 |


**StoreStat**
- Holds The availableSize of the whole store

| AvailableSize|
|--------------|
| 8 bytes      |

**NamespaceStats**

| NrObjects  | NrRwquests| Created |
|-----------|------------|----------
| 8 bytes   | 8 bytes    |


**Reservation**


| SizeReserved  | SizeUsed  |CreationDateSize |UpdateDateSize |
|---------------|-----------|-----------------|---------------|
| 8 bytes       | 8 bytes   |     2 bytes     |    2 bytes    |


| ExpirationDateSize  | IDSize    |AdminIdSize  |CreationDate |
|---------------------|-----------|-------------|-------------|
| 2 bytes             | 2 bytes   |     2 bytes |             |

| UpdateDate  | ExpirationDate  | ID  | AdminId |
|-------------|-----------------|-----|---------|
|             |                 |     |         |


Token formats
=============

**Reservation Token**

| Random bytes  | expirationEpoch  |NamespaceIdSize |ReservationIdSize|
|---------------|------------------|-----------------|---------------|
| 51 bytes       | 8 bytes         |     2 bytes     |    2 bytes    |


| NamespaceID  | ReservationID |
|--------------|---------------|
|              |               |

**DataAccessToken**

| Random bytes  | expirationEpoch  |Admin|Read |Write|Delete|user|
|---------------|------------------|-----|-----|-----|------|----|
| 51 bytes      | 8 bytes          |1byte|1byte|1byte|1byte |    |



Naming conventions
==================

**Store stats**
- Fixed name :: ```0@stats```

**Namespaces Stats**
- Fixed prefix :: ```0@stats_{namespace_id}```

**Namespace Reservations**
- Fixed prefix ``` 1@res_{namespace_id}_{reservation_id}```

**Namespaces**
- Alphanumeric only

