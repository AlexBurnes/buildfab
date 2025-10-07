#!/usr/bin/env bash

set -o nounset  # exit if variable is unset, set -u
set -o errexit  # exit on any command error, set -e
set -o errtrace # exit on any command error, set -E
set -o pipefail # exit on pipe fail read

clre='\033[0m' black='\e[30m' red='\033[0;31m' green='\033[0;32m' orange='\033[0;33m' yellow='\033[1;33m' blue='\033[0;34m'
purple='\033[0;35m' magenta='\033[0;35m' cyan='\033[0;36m' gray='\e[37m' white='\e[38m' bold='\e[1m' blink='\e[5m]'
trap cleanup SIGINT SIGTERM ERR EXIT
work_dir=$(pwd -P)
function cleanup() {
    trap - SIGINT SIGTERM ERR EXIT
    rc=$?
    cd ${work_dir}
    if [ ${rc} -gt 0 ]; then
        echo -e "${red}❗failure${clre}"
    fi
    exit ${rc}
}

echo -e "🔑 setup ssh and keys"
command -v ssh-agent >/dev/null || ( apk add --update openssh-client ) || (yum -y install openssh-clients)
eval $(ssh-agent -s)
mkdir -p ~/.ssh
chmod 700 ~/.ssh

cat .ssh_private_key | tr -d '\r' | ssh-add -
ssh-keyscan 192.168.17.92 >> ~/.ssh/known_hosts
ssh-keyscan gitsrv.svyazcom.ru >> ~/.ssh/known_hosts
chmod 644 ~/.ssh/known_hosts

export http_proxy="http://172.18.1.11:3128/"

echo -e "📦 clone conan"
[ ! -d /conan-io ] && git clone -v https://github.com/conan-io/conan.git /conan-io

echo -e "📦 clone project"
git clone git@gitsrv.svyazcom.ru:ClearingManager/imsi-region.git /imsi-region
cd /imsi-region


git checkout version/1.3_feat/docker
git submodule init
git submodule update


echo -e "📦 download buildfab"
wget -O - https://github.com/AlexBurnes/buildfab/releases/latest/download/buildfab-linux-amd64-install.sh | sh

echo -e "🔍 platform"
buildfab platform-view -v

echo -e "🔧 pre-install requirements"
buildfab ci-pre-install -vv

echo -e "🔧 update conan remotes"
OS=$(scripts/version os)
OS_VERSION=$(scripts/version os_version)
if [ -f build/${OS}${OS_VERSION}/env ]; then
    source build/${OS}${OS_VERSION}/env
fi
if [ -d /cache ]; then
    mkdir -p /cache/${OS}${OS_VERSION}/.conan2
    rm -fr ./.conan2
    ln -s /cache/${OS}${OS_VERSION}/.conan2 ./.conan2
    conan profile detect --force
fi

conan remote remove conan-local || true
conan remote remove conan-proxy || true
conan remote add -f conan-local http://172.18.1.111:8081/artifactory/api/conan/conan-local
conan remote add -f conan-proxy http://172.18.1.111:8081/artifactory/api/conan/conan-proxy-svc

echo -e "${green}✓ ${clre} to build project run: cd imsi-region && buildfab build"

