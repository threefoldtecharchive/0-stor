#include <stdio.h>
#include <stdlib.h>
#include "libg8stor.h"

int main(int argc, char *argv[]) {
    remote_t *remote;
    buffer_t *buffer;
    char *file;

    //
    // arguments checker
    //
    if(argc < 2) {
        fprintf(stderr, "[-] usage: %s [filename]\n", argv[0]);
        exit(EXIT_FAILURE);
    }

    file = argv[1];
    printf("[+] uploading: %s\n", file);

    //
    // connect to redis (ardb)
    //
    if(!(remote = remote_connect("172.17.0.2", 16379)))
        exit(EXIT_FAILURE);

    //
    // initialize buffer
    //
    if(!(buffer = bufferize(file))) {
        fprintf(stderr, "[-] cannot bufferize the file\n");
        exit(EXIT_FAILURE);
    }

    //
    // chunks
    //
    printf("[+] uploading %d chunks\n", buffer->chunks);
    for(int i = 0; i < buffer->chunks; i++) {
        // uploading chunk
        char *chunk = upload(remote, buffer);

        printf("-> %s\n", chunk);
        free(chunk);
    }

    printf("[+] finalsize: %lu bytes\n", buffer->finalsize);

    //
    // cleaning
    //
    remote_free(remote);
    buffer_free(buffer);

    return 0;
}
