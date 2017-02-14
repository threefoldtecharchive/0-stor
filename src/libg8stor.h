#ifndef LIBG8STOR_H
    #define LIBG8STOR_H

    typedef struct remote_t {
        redisContext *redis;

    } remote_t;

    typedef struct buffer_t {
        FILE *fp;
        unsigned char *data;
        size_t length;
        size_t current;
        size_t chunksize;
        size_t finalsize;
        int chunks;

    } buffer_t;

    typedef struct chunk_t {
        unsigned char *hash;
        unsigned char *key;

    } chunk_t;

    // remote manager
    remote_t *remote_connect(const char *host, int port);
    void remote_free(remote_t *remote);

    // file buffer
    buffer_t *bufferize(char *filename);
    const unsigned char *buffer_next(buffer_t *buffer);
    void buffer_free(buffer_t *buffer);

    // chunk
    chunk_t *chunk_new(unsigned char *hash, unsigned char *key);
    void chunk_free(chunk_t *chunk);

    // uploader
    chunk_t *upload(remote_t *remote, buffer_t *buffer);
#endif
