#! /bin/sh

CACHE_DIR=tmp/plugins
DEST_REGISTRY=docker.io/kubegems

ACTION=

CUSTOM_IMAGES='
istio/proxyv2:1.11.2
istio/istiod:1.11.2
'

CONTAINER_CMD=docker
if ! [ -x "$(command -v docker)" ]; then
    CONTAINER_CMD=podman
fi

parsed_images() {
    awk 'match($$0,/image:\s"*([a-z0-9:/@.:\-]+)/,i){print i[1]}'
}

list_images() {
    for image in ${CUSTOM_IMAGES}; do
        echo ${image}
    done
    bin/kubegems plugins template deploy/plugins/* | parsed_images
    bin/kubegems plugins template deploy/plugins-local-stack.yaml | bin/kubegems plugins -d ${CACHE_DIR} template - | parsed_images
}

copy_image() {
    tagedimage=${DEST_REGISTRY}/${1##*/}
    echo "copying [${image}] --> [${tagedimage}]"
    ${CONTAINER_CMD} pull ${image}
    ${CONTAINER_CMD} tag ${image} ${tagedimage}
    ${CONTAINER_CMD} push ${tagedimage}
}

usage() {
    echo "Usage: $0 [options] [image]"
    echo "Options:"
    echo "  -h, --help: print this help"
    echo "  -l, --list: list all images"
    echo "  -t, --to: copy all images to target registry. (example: ${DEST_REGISTRY})"
    exit 1
}

OPTS=$(getopt -o t:,l,h -l to:,list,help -- "$@")
if [ $? != 0 ]; then
    usage
fi
eval set -- "$OPTS"
while true; do
    case $1 in
    -l | --list)
        ACTION=list
        shift
        ;;
    -t | --to)
        DEST_REGISTRY=$2
        ACTION=copy
        shift 2
        ;;
    -h | --help)
        usage
        ;;
    --)
        shift
        break
        ;;
    *)
        echo "unexpected option: $1"
        usage
        ;;
    esac
done

if [ "${ACTION}" = "copy" ]; then
    for image in $(list_images); do
        copy_image ${image}
    done
    exit 0
fi

if [ "${ACTION}" = "list" ]; then
    list_images
    exit 0
fi

usage
