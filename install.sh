#!/bin/bash

##################################################
#  check user                                    #
##################################################
if [ "$(whoami)" != "root" ]; then
   echo "This script must be run as root" 1>&2
   exit 1
fi

##################################################
#  dependencies                                  #
##################################################
echo "install dependencies (git and fuse)"
apt update          &> /dev/null  # update
apt install git -y  &> /dev/null  # install git for go get command
apt install fuse -y &> /dev/null  # install fuse for FUSE support

##################################################
#  fuse.conf                                     #
##################################################
sed -i "s/#user_allow_other/user_allow_other/g" /etc/fuse.conf

##################################################
#  rm and make tempdir                           #
##################################################
TEMPDIR=/tmp/splitfusetmp
rm -R $TEMPDIR &> /dev/null
mkdir -p $TEMPDIR
cd $TEMPDIR

##################################################
#  download go (without installation)            #
##################################################
echo "download go (without installation)"
GOVERSION="https://dl.google.com/go/go1.10.linux-amd64.tar.gz"
GOFILE="go.tar.gz"
wget -q -O $GOFILE $GOVERSION
tar -xzf $GOFILE
rm $GOFILE
GOCMD=$TEMPDIR/go/bin/go
GOHOME=$TEMPDIR/gohome

##################################################
#  install splitfuse                             #
##################################################
echo "build & install splitfuse"
PROJECT="github.com/SchnorcherSepp/splitfuse"
INSTALLPATH="/usr/bin/splitfuse"
GOPATH=$GOHOME $GOCMD get $PROJECT
cd $GOHOME/src/$PROJECT
GOPATH=$GOHOME $GOCMD build -o $INSTALLPATH

##################################################
#  install rclone                                #
##################################################
echo "build & install rclone"
PROJECT="github.com/ncw/rclone"
INSTALLPATH="/usr/bin/rclone"
GOPATH=$GOHOME $GOCMD get $PROJECT
cd $GOHOME/src/$PROJECT
GOPATH=$GOHOME $GOCMD build -o $INSTALLPATH

##################################################
#  FIN (rm tempdir)                              #
##################################################
echo "clean up"
rm -R $TEMPDIR &> /dev/null
