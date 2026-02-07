#!/bin/bash
# Copyright the Hyperledger Fabric contributors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
set -euo pipefail

#Install softhsm
# Install SoftHSM2 based on OS
if [[ "$OSTYPE" == "darwin"* ]]; then
    echo "Detected macOS"
    brew install softhsm
elif [[ -f /etc/redhat-release ]]; then
    echo "Detected RHEL-based system"
    sudo dnf install -y softhsm
elif [[ -f /etc/debian_version ]]; then
    echo "Detected Debian-based system"
    sudo apt-get update
    sudo apt-get install -y softhsm2
else
    echo "Unsupported OS: $OSTYPE"
    exit 1
fi