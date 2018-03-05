#!/bin/bash

GOVERSION="https://dl.google.com/go/go1.10.linux-amd64.tar.gz"


# Check user
if [ "$(whoami)" != "root" ]; then
   echo "This script must be run as root" 1>&2
   exit 1
fi

# install git, unzip and fuse
echo "install git, unzip and fuse ..."
apt -qq update
apt install -qq git unzip fuse -y

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

# install rclone
echo "install rclone ..."
cd $WORKDIR
DLRCLONE="https://downloads.rclone.org/rclone-current-linux-amd64.zip"
DLFILE="rclone.zip"
wget -q -O $DLFILE $DLRCLONE
unzip -qq -a rclone.zip -d rclone
rm $DLFILE
cd rclone/*
#binary
cp rclone /usr/bin/rclone.new
chmod 755 /usr/bin/rclone.new
chown root:root /usr/bin/rclone.new
mv /usr/bin/rclone.new /usr/bin/rclone
#manuals
mkdir -p /usr/local/share/man/man1
cp rclone.1 /usr/local/share/man/man1/
mandb -q
