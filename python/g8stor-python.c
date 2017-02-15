#include <Python.h>
#include <stdio.h>
#include <stdlib.h>
#include <hiredis.h>
#include "libg8stor.h"

remote_t *remotes = NULL;

static PyObject *g8storclient_connect(PyObject *self, PyObject *args) {
    (void) self;
    remote_t *remote;
    const char *host;
    int port;

    if (!PyArg_ParseTuple(args, "si", &host, &port))
        return NULL;

    printf("[+] python binding: connecting %s (port: %d)\n", host, port);
    if(!(remote = remote_connect(host, port)))
        return NULL;

    // capsule will encapsulate our remote_t which contains
    // the redis client object
    return PyCapsule_New(remote, "RemoteClient", NULL);
}

static PyObject *g8storclient_upload(PyObject *self, PyObject *args) {
    (void) self;
    remote_t *remote;
    buffer_t *buffer;
    PyObject *capsule;
    char *file;

    if(!PyArg_ParseTuple(args, "Os", &capsule, &file))
        return NULL;

    // extracting the capsule from the first argument
    // this will gives us the redis client
    if(!(remote = PyCapsule_GetPointer(capsule, "RemoteClient")))
        return NULL;

    // initialize buffer
    if(!(buffer = bufferize(file)))
        return Py_None;

    // chunks
    PyObject *hashes = PyList_New(buffer->chunks);

    printf("[+] uploading %d chunks\n", buffer->chunks);
    for(int i = 0; i < buffer->chunks; i++) {
        // uploading chunk
        chunk_t *chunk = upload(remote, buffer);

        // inserting hash to the list
        PyObject *pychunk = PyDict_New();
        PyDict_SetItemString(pychunk, "hash", Py_BuildValue("s", chunk->id));
        PyDict_SetItemString(pychunk, "key", Py_BuildValue("s", chunk->cipher));
        PyList_SetItem(hashes, i, pychunk);

        chunk_free(chunk);
    }

    printf("[+] finalsize: %lu bytes\n", buffer->finalsize);

    // cleaning
    buffer_free(buffer);

    return hashes;
}

static PyObject *g8storclient_download(PyObject *self, PyObject *args) {
    (void) self;
    remote_t *remote;
    buffer_t *buffer;
    PyObject *capsule;
    PyObject *hashes;
    char *file;

    if(!PyArg_ParseTuple(args, "OOs", &capsule, &hashes, &file))
        return NULL;

    // extracting the capsule from the first argument
    // this will gives us the redis client
    if(!(remote = PyCapsule_GetPointer(capsule, "RemoteClient")))
        return NULL;

    // initialize buffer
    if(!(buffer = buffer_writer(file)))
        return Py_None;

    // parsing dictionnary
    int chunks = (int) PyList_Size(hashes);

    // chunks
    printf("[+] downloading %d chunks\n", chunks);
    for(int i = 0; i < chunks; i++) {
        size_t chunksize;
        unsigned char *id, *cipher;

        PyObject *item = PyList_GetItem(hashes, i);

        PyArg_Parse(PyDict_GetItemString(item, "hash"), "s", &id);
        PyArg_Parse(PyDict_GetItemString(item, "key"), "s", &cipher);

        chunk_t *chunk = chunk_new(strdup(id), strdup(cipher));

        // downloading chunk
        if(!(chunksize = download(remote, chunk, buffer)))
            fprintf(stderr, "[-] download failed\n");

        printf("-> chunk restored: %lu bytes\n", chunksize);
        chunk_free(chunk);
    }

    size_t finalsize = buffer->finalsize;
    printf("[+] finalsize: %lu bytes\n", finalsize);

    // cleaning
    buffer_free(buffer);

    return PyLong_FromLong(finalsize);
}

static PyMethodDef g8storclient_cm[] = {
    {"connect",  g8storclient_connect,  METH_VARARGS, "Initialize client"},
    {"upload",   g8storclient_upload,   METH_VARARGS, "Upload a file"},
    {"download", g8storclient_download, METH_VARARGS, "Download a file"},
    {NULL, NULL, 0, NULL}
};

static struct PyModuleDef g8storclientmodule = {
    PyModuleDef_HEAD_INIT,
    "g8storclient", // name of module
    NULL,           // module documentation, may be NULL
    -1,             // -1 if the module keeps state in global variables.
    g8storclient_cm
};

PyMODINIT_FUNC PyInit_g8storclient(void) {
    return PyModule_Create(&g8storclientmodule);
}
