LIBRARY = libg8stor.so

SRC = libg8stor.c 3rdparty/xxtea-c/xxtea.c
OBJ = libg8stor.o xxtea.o

CFLAGS  = -pthread -fPIC -W -Wall -O2 -I/usr/include/hiredis -I3rdparty/xxtea-c -g
LDFLAGS = -lhiredis -lssl -lcrypto -lsnappy -lz -lm

CC = gcc

all: $(LIBRARY)

$(LIBRARY): $(OBJ)
	$(CC) -shared -o $@ $^ $(LDFLAGS)
	ar rcs libg8stor.a $(OBJ)

libg8stor.o: libg8stor.c
	$(CC) $(CFLAGS) -c $<

xxtea.o: 3rdparty/xxtea-c/xxtea.c
	$(CC) $(CFLAGS) -c $<

clean:
	rm -fv *.o

mrproper: clean
	rm -fv $(LIBRARY) *.a

