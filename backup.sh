#!/usr/bin/env bash

#
# Usage: ./backup.sh -u <Device ID> -p <Backup Path>
#
# Device ID can be found using idevice_id

set -o errexit
set -o pipefail

while getopts u:p: option
do
case "${option}"
in
u) DEVICE_ID=${OPTARG};;
p) BACKUP_PATH=${OPTARG};;
esac
done

if [[ -z $DEVICE_ID ]] || [[ -z $BACKUP_PATH ]]
then
cat << EOF
Usage: ./backup.sh -u <Device ID> -p <Backup Path>

Device ID can be found using idevice_id
EOF
     exit 1
fi

idevicebackup2 -u "${DEVICE_ID}" backup "${BACKUP_PATH}"

echo "Adding (${BACKUP_PATH}/${DEVICE_ID}) to IPFS..."

CID=$(ipfs add --nocopy -Q -r "${BACKUP_PATH}/${DEVICE_ID}")

echo "Added to IPFS (${CID})"

echo "Adding CID to MFS"

echo "${CID}" | ipfs files write --create /ios-backups/${DEVICE_ID}

echo "Added (${CID}) -> /ios-backups/${DEVICE_ID}"