#!/bin/bash

DIR=$1
CRFILE=$(dirname "$0")/copyright.go.header

FAILED_FILES=""
for f in $(find "$DIR" -name "*.go" | grep -v vendor | grep -v pb.go | grep -v pb_test.go); do
    diff <(sed -n -e '1,15p' "$f") <(sed -n -e '1,15p' "$CRFILE")
    if [ $? -ne 0 ]; then
        FAILED_FILES="$FAILED_FILES\n  > $f"
    fi
done

if [ -z "$FAILED_FILES" ]; then
    echo "All Golang files have the required copyright header!"
    exit 0
fi

echo ""
echo "---"
echo ""
echo "The following Golang fils are missing the required copyright header:"
echo ""
printf "$FAILED_FILES"
echo ""
exit 1
