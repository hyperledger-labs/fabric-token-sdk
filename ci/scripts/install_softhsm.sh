#!/bin/bash
# Copyright the Hyperledger Fabric contributors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
set -euo pipefail

#Install softhsm
if [[ "$OSTYPE" == "darwin"* ]]; then
    brew install softhsm
else
    sudo apt-get install -y softhsm2
fi