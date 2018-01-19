#!/bin/bash

DIR=$1
GOCRFILE=$(dirname "$0")/copyright.go.header
PYCRFILE=$(dirname "$0")/copyright.py.header

FAILED_FILES=""

# python license check
for f in $(find "$DIR" -name "*.go" | grep -v vendor | grep -v pb.go | grep -v pb_test.go); do
    diff <(sed -n -e '1,15p' "$f") <(sed -n -e '1,15p' "$GOCRFILE")
    if [ $? -ne 0 ]; then
        FAILED_FILES="$FAILED_FILES\n  > $f"
    fi
done

# python licence check
for f in $(find "$DIR" -name "*.py" | grep -v vendor | grep -v generated| grep -v test_suite|grep -v benchmarker); do
    diff <(sed -n -e '1,13p' "$f") <(sed -n -e '1,15p' "$PYCRFILE")
    if [ $? -ne 0 ]; then
        FAILED_FILES="$FAILED_FILES\n  > $f"
    fi
done

if [ -z "$FAILED_FILES" ]; then
    echo "All code files have the required copyright header!"
    exit 0
fi

echo ""
echo "---"
echo ""
echo "The following code fils are missing the required copyright header:"
echo ""
printf "$FAILED_FILES"
echo ""
exit 1
