#!/bin/bash
# Copyright 2016 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -o errexit
set -o nounset
set -o pipefail
set -o xtrace

# PROJECT="k8s-jkns-ci-node-e2e"
readonly testinfra="$(dirname "${0}")/.."

export NODE_IMG_VERSION="master"
export NODE_TEST_SCRIPT="test/e2e_node/jenkins/e2e-node-jenkins.sh"
export NODE_TEST_PROPERTIES="test/e2e_node/jenkins/cri_validation/jenkins-serial.properties"

### Runner
readonly runner="${testinfra}/jenkins/node-dockerized.sh"
export KUBEKINS_TIMEOUT="240m"
timeout -k 15m "${KUBEKINS_TIMEOUT}" "${runner}" && rc=$? || rc=$?

### Reporting
if [[ ${rc} -eq 124 || ${rc} -eq 137 ]]; then
    echo "Build timed out" >&2
elif [[ ${rc} -ne 0 ]]; then
    echo "Build failed" >&2
fi
echo "Exiting with code: ${rc}"
exit ${rc}
