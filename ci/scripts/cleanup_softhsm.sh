#!/bin/bash

LABEL_TO_DELETE="ForFSC"

# Get all slot numbers
slots=$(softhsm2-util --show-slots | grep "Slot " | awk '{print $2}')

for slot in $slots
do
    # Check if the token in this slot has the label we want to delete
    label=$(softhsm2-util --show-slots | grep -A 2 "Slot $slot" | grep "Label:" | awk '{print $2}')
    if [ "$label" == "$LABEL_TO_DELETE" ]; then
        echo "Deleting token in slot $slot with label $LABEL_TO_DELETE"
        softhsm2-util --delete-token --token "$LABEL_TO_DELETE"
    fi
done