#!/bin/bash

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
	SIZE=$(echo $1 | jq -r '.size')

	# LVM substitutes - with --
	VOLUMEID=`echo $VOLUMEID|sed s/-/--/g`

	DMDEV="/dev/mapper/${VOLUMEID}"
	if [ ! -b "${DMDEV}" ]; then
		err "{\"status\": \"Failure\", \"message\": \"Volume ${VOLUMEID} does not exist\"}"
		exit 1
	fi
	log "{\"status\": \"Success\", \"device\":\"${DMDEV}\"}"
	exit 0
}

detach() {
	log "{\"status\": \"Success\"}"
	exit 0
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
