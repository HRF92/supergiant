#!/bin/bash

## Required Flex Volume Options.
##{
##  "volumeID": "bar",
##  "name": "foo"
##}


## Who am i?
## Where am i?
REGION=$(doctl compute droplet list | grep ${COREOS_PUBLIC_IPV4} | awk '{print $7}')
DROPLET_ID=$(doctl compute droplet list | grep ${COREOS_PUBLIC_IPV4} | awk '{print $1}')

usage() {
	err "Invalid usage. Usage: "
	err "\t$0 init"
	err "\t$0 attach <json params>"
	err "\t$0 detach <mount device>"
	err "\t$0 mount <mount dir> <mount device> <json params>"
	err "\t$0 unmount <mount dir>"
	exit 1
}

err() {
	echo -ne $* 1>&2
}

log() {
	echo -ne $* >&1
}

ismounted() {
	MOUNT=`findmnt -n ${MNTPATH} 2>/dev/null | cut -d' ' -f1`
	if [ "${MOUNT}" == "${MNTPATH}" ]; then
		echo "1"
	else
		echo "0"
	fi
}

attach() {
	VOLUMEID=$(echo $1 | jq -r '.volumeID')
	VOLUMENAME=$(echo $1 | jq -r '.name')

  doctl compute volume-action attach $VOLUMEID $DROPLET_ID >/dev/null 2>&1

  # Find the new volume.
	DEVNAME="/dev/disk/by-id/scsi-0DO_Volume_${VOLUMENAME}"

	# Wait for attach
	NEXT_WAIT_TIME=1
  until ls -l $DEVNAME >/dev/null 2>&1 || [ $NEXT_WAIT_TIME -eq 4 ]; do
   sleep $(( NEXT_WAIT_TIME++ ))
  done

	#Record the actual device name.
	DVSHRTNAME=$(ls -l /dev/disk/by-id | grep ${VOLUMENAME} | awk '{print $11}' | sed 's/\.\.\///g' | sed '/^\s*$/d')
	DMDEV="/dev/${DVSHRTNAME}"
	# Error check.
	if [ ! -b "${DMDEV}" ]; then
		err "{\"status\": \"Failure\", \"message\": \"Volume ${VOLUMEID} does not exist\"}"
		exit 1
	fi
	log "{\"status\": \"Success\", \"device\":\"${DMDEV}\"}"
	exit 0
}

detach() {
	## This is nasty, I would prefer to use doctl for detach as well... but it appears that it is bugged.
	## I will update this when a new version of doctl releases. For now raw api.
	TOKEN=$(cat ~/.config/doctl/config.yaml | grep access-token | awk '{print $2}')
	SRTDEVNAME=$(echo $1 | sed 's/\/dev\///')
	VOLNAME=$(ls -l /dev/disk/by-id | grep ${SRTDEVNAME} | awk '{print $9}' | sed 's/scsi-0DO_Volume_//')
	curl -X POST -H "Content-Type: application/json" -H "Authorization: Bearer ${TOKEN}" -d "{\"type\": \"detach\", \"droplet_id\": \"${DROPLET_ID}\", \"volume_name\": \"${VOLNAME}\", \"region\": \"nyc1\"}" "https://api.digitalocean.com/v2/volumes/actions" >/dev/null 2>&1

	if [ -b "$1" ]; then
		log "{\"status\": \"Success\"}"
		exit 0
	fi
	exit 1
}

domount() {
	MNTPATH=$1
	DMDEV=$2
	FSTYPE=$(echo $3|jq -r '.["kubernetes.io/fsType"]')


	if [ ! -b "${DMDEV}" ]; then
		err "{\"status\": \"Failure\", \"message\": \"${DMDEV} does not exist\"}"
		exit 1
	fi

	if [ $(ismounted) -eq 1 ] ; then
		log "{\"status\": \"Success\"}"
		exit 0
	fi

	VOLFSTYPE=`blkid -o udev ${DMDEV} 2>/dev/null|grep "ID_FS_TYPE"|cut -d"=" -f2`
	if [ "${VOLFSTYPE}" == "" ]; then
		mkfs -t ${FSTYPE} ${DMDEV} >/dev/null 2>&1
		if [ $? -ne 0 ]; then
			err "{ \"status\": \"Failure\", \"message\": \"Failed to create fs ${FSTYPE} on device ${DMDEV}\"}"
			exit 1
		fi
	fi

	mkdir -p ${MNTPATH} &> /dev/null

	mount ${DMDEV} ${MNTPATH} &> /dev/null
	if [ $? -ne 0 ]; then
		err "{ \"status\": \"Failure\", \"message\": \"Failed to mount device ${DMDEV} at ${MNTPATH}\"}"
		exit 1
	fi
	log "{\"status\": \"Success\"}"
	exit 0
}

unmount() {
	MNTPATH=$1
	if [ $(ismounted) -eq 0 ] ; then
		log "{\"status\": \"Success\"}"
		exit 0
	fi

	umount ${MNTPATH} &> /dev/null
	if [ $? -ne 0 ]; then
		err "{ \"status\": \"Failed\", \"message\": \"Failed to unmount volume at ${MNTPATH}\"}"
		exit 1
	fi
	rmdir ${MNTPATH} &> /dev/null

	log "{\"status\": \"Success\"}"
	exit 0
}

op=$1

if [ "$op" = "init" ]; then
	log "{\"status\": \"Success\"}"
	exit 0
fi

if [ $# -lt 2 ]; then
	usage
fi

shift

case "$op" in
	attach)
		attach $*
		;;
	detach)
		detach $*
		;;
	mount)
		domount $*
		;;
	unmount)
		unmount $*
		;;
	*)
		usage
esac

exit 1
