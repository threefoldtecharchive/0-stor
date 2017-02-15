#include <stdio.h>
#include <stdlib.h>
#include <hiredis.h>
#include <libg8stor.h>

int main(int argc, char *argv[]) {
    char *input, *output;
    remote_t *remote;
    buffer_t *buffer;
    chunk_t **chunks;
    int chunks_length;

    //
    // arguments checker
    //
    if(argc < 3) {
        fprintf(stderr, "[-] usage: %s [input-filename] [output-filename]\n", argv[0]);
        exit(EXIT_FAILURE);
    }

    input = argv[1];
    output = argv[2];

    printf("[+] uploading file: %s\n", input);
    printf("[+] restoring file: %s\n", output);

    //
    // connect to redis (ardb)
    //
    if(!(remote = remote_connect("172.17.0.2", 16379)))
        exit(EXIT_FAILURE);

    //
    // initialize buffer and chunks
    //
    if(!(buffer = bufferize(input))) {
        fprintf(stderr, "[-] cannot bufferize: %s\n", input);
        exit(EXIT_FAILURE);
    }

    chunks_length = buffer->chunks;
    if(!(chunks = (chunk_t **) malloc(sizeof(chunk_t *) * chunks_length))) {
        perror("[-] chunks malloc");
        exit(EXIT_FAILURE);
    }

    //
    // uploading chunks
    //
    printf("[+] =============================\n");
    printf("[+] uploading %d chunks\n", buffer->chunks);
    printf("[+] =============================\n");

    for(int i = 0; i < buffer->chunks; i++) {
        // uploading chunk
        chunk_t *chunk = upload(remote, buffer);

        printf("-> %s [%s]\n", chunk->id, chunk->cipher);
        chunks[i] = chunk;
    }

    printf("[+] finalsize: %lu bytes\n", buffer->finalsize);
    buffer_free(buffer);

    //
    // downloading chunks
    //
    printf("[+] =============================\n");
    printf("[+] downloading %d chunks\n", chunks_length);
    printf("[+] =============================\n");

    if(!(buffer = buffer_writer(output))) {
        fprintf(stderr, "[-] cannot bufferize: %s\n", output);
        exit(EXIT_FAILURE);
    }

    for(int i = 0; i < chunks_length; i++) {
        size_t chunksize;

        // downloading chunk
        if(!(chunksize = download(remote, chunks[i], buffer)))
            fprintf(stderr, "[-] download failed\n");

        printf("-> chunk restored: %lu bytes\n", chunksize);
    }

    printf("[+] finalsize: %lu bytes read in %d chunks\n", buffer->finalsize, buffer->chunks);
    buffer_free(buffer);

    //
    // cleaning
    //
    for(int i = 0; i < chunks_length; i++)
        chunk_free(chunks[i]);

    free(chunks);
    remote_free(remote);

    return 0;
}
