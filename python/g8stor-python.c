#include <Python.h>
#include <stdio.h>
#include <stdlib.h>
#include <hiredis.h>
#include "libg8stor.h"

remote_t *remotes = NULL;

static PyObject *cstor_connect(PyObject *self, PyObject *args) {
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

static PyObject *cstor_upload(PyObject *self, PyObject *args) {
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
    if(!(buffer = bufferize(file))) {
        PyObject *hashs = PyList_New(0);
        return hashs;
    }

    // chunks
    PyObject *hashs = PyList_New(buffer->chunks);

    printf("[+] uploading %d chunks\n", buffer->chunks);
    for(int i = 0; i < buffer->chunks; i++) {
        // uploading chunk
        chunk_t *chunk = upload(remote, buffer);

        // inserting hash to the list
        PyObject *pychunk = PyDict_New();
        PyDict_SetItemString(pychunk, "hash", Py_BuildValue("s", chunk->hash));
        PyDict_SetItemString(pychunk, "key", Py_BuildValue("s", chunk->key));
        PyList_SetItem(hashs, i, pychunk);

        chunk_free(chunk);
    }

    printf("[+] finalsize: %lu bytes\n", buffer->finalsize);

    // cleaning
    buffer_free(buffer);

    return hashs;
}

static PyMethodDef CstorMethods[] = {
    {"connect",  cstor_connect, METH_VARARGS, "Initialize"},
    {"upload",   cstor_upload,  METH_VARARGS, "Upload"},
    {NULL, NULL, 0, NULL}
};

static struct PyModuleDef cstormodule = {
    PyModuleDef_HEAD_INIT,
    "cstor",      // name of module
    NULL,         // module documentation, may be NULL
    -1,           // -1 if the module keeps state in global variables.
    CstorMethods
};

PyMODINIT_FUNC PyInit_g8storclient(void) {
    return PyModule_Create(&cstormodule);
}
