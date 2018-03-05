#!/bin/bash

GOVERSION="https://dl.google.com/go/go1.10.linux-amd64.tar.gz"


# Check user
if [ "$(whoami)" != "root" ]; then
   echo "This script must be run as root" 1>&2
   exit 1
fi

# install git
echo "install git and fuse ..."
apt -qq update
apt install -qq git fuse -y

# create workdir
WORKDIR=/tmp/splitfusetmp
rm -R $WORKDIR &> /dev/null
mkdir -p $WORKDIR

# download and unzip GO
echo "download go ..."
DLFILE="go.tar.gz"
cd $WORKDIR
wget -q -O $DLFILE $GOVERSION
tar -xzf $DLFILE
rm $DLFILE

# clone project splitfuse
echo "clone splitfuse ..."
GOPATH=$WORKDIR/gohome $WORKDIR/go/bin/go get github.com/SchnorcherSepp/splitfuse

# build and install splitfuse
echo "build and install splitfuse ..."
cd $WORKDIR/gohome/src/github.com/SchnorcherSepp/splitfuse/
GOPATH=$WORKDIR/gohome $WORKDIR/go/bin/go build -o /usr/bin/splitfuse
