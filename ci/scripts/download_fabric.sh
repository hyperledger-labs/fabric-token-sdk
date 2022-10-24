#!/bin/bash
# Copyright the Hyperledger Fabric contributors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
#set -euo pipefail

download() {
    local BINARY_FILE=$1
    local URL=$2
    echo "===> Downloading: " "${URL}"
    curl -L --retry 5 --retry-delay 3 "${URL}" | tar xz || rc=$?
    if [ -n "$rc" ]; then
        echo "==> There was an error downloading the binary file."
        return 22
    else
        echo "==> Done."
    fi
}

pullBinaries() {
    ARCH=$(echo "$(uname -s|tr '[:upper:]' '[:lower:]'|sed 's/mingw64_nt.*/windows/')-$(uname -m |sed 's/x86_64/amd64/g')" |sed 's/darwin-arm64/darwin-amd64/g')
    MARCH=$(uname -m)
    local VERSION=$1

    BINARY_FILE=hyperledger-fabric-${ARCH}-${VERSION}.tar.gz
    echo "===> Downloading ${VERSION} specific fabric binaries for ${ARCH} Platform"
    download "${BINARY_FILE}" "https://github.com/hyperledger/fabric/releases/download/v${VERSION}/${BINARY_FILE}"
    if [ $? -eq 22 ]; then
        echo
        echo "------> ${FABRIC_TAG} platform specific fabric binary is not available to download <----"
        echo
        exit
    fi
}

function checkFabricBinaryPresence() {
    local VERSION=$1
    ## Check if binaries already exist
    ${PWD}/bin/peer version > /dev/null 2>&1

    if [[ $? -ne 0  ]]; then
        echo "no fabric binaries detected, pulling down"
        # No binaries found, pull then down
        pullBinaries $VERSION
        return
    fi

    LOCAL_VERSION=$(${PWD}/bin/peer version | sed -ne 's/^ Version: //p')

    echo "LOCAL_VERSION=$LOCAL_VERSION"

    if [ "$LOCAL_VERSION" != "$VERSION" ]; then
        echo "Local fabric binaries don't match requested, replacing with requested: $VERSION" 
        rm bin/*
        rm config/*
        pullBinaries $VERSION
        return
    fi

    echo "Fabric binaries already downloaded"
}

# download_fabric <directory> <version>
mkdir -p $1 || true
cd $1
checkFabricBinaryPresence $2