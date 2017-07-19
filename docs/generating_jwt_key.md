## How to generate a jwt signing key for the store

In the script folder you'll find a small go program that generate a random key. You can use it to generate key that the 0-stor will use to generate JWT.

```shell
go run generate_jwt_key.go -out jwt.key
```
