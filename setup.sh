#!/bin/bash

##################################################
#  check user                                    #
##################################################
if [ "$(whoami)" != "root" ]; then
   echo "This script must be run as root" 1>&2
   exit 1
fi

##################################################
#  add new user                                  #
##################################################
USER=splitfuse
adduser --system --home /etc/empty --no-create-home --disabled-login --disabled-password --group $USER > /dev/null

##################################################
#  create config folder and files                #
##################################################
CONFFOLDER=/etc/splitfuse
mkdir -p $CONFFOLDER
chown $USER:$USER $CONFFOLDER
chmod 700 $CONFFOLDER

SPLITKEYFILE=$CONFFOLDER/splitfuse.key
touch $SPLITKEYFILE
chown $USER:$USER $SPLITKEYFILE
chmod 600 $SPLITKEYFILE

RCLONECONFFILE=$CONFFOLDER/rclone.conf
touch $RCLONECONFFILE
chown $USER:$USER $RCLONECONFFILE
chmod 600 $RCLONECONFFILE

MNTRCLONE=/mnt/rclone
mkdir -p $MNTRCLONE
chown $USER:$USER $MNTRCLONE
chmod 700 $MNTRCLONE

MNTSPLIT=/mnt/splitfuse
mkdir -p $MNTSPLIT
chown $USER:$USER $MNTSPLIT
chmod 755 $MNTSPLIT

##################################################
#  create mount script                           #
##################################################
MOUNTSCRIPT=/usr/bin/sfmount
cat > $MOUNTSCRIPT << EOL
#!/bin/bash
if [ "\$(/usr/bin/whoami)" != "$USER" ]; then
   /bin/echo "This script must be run as user $USER" 1>&2
   exit 1
fi
# rclone mount
HOME=$CONFFOLDER /usr/bin/rclone --config $RCLONECONFFILE mount readonly: $MNTRCLONE &
# best race condition fix ever !!
/bin/sleep 5
# splitfuse
/usr/bin/splitfuse normal --dbfile $MNTRCLONE/index.db --keyfile $SPLITKEYFILE --chunkdir $MNTRCLONE/partstorage --mountdir $MNTSPLIT
EOL
chmod +x $MOUNTSCRIPT

##################################################
#  create upload script                          #
##################################################
UPLOADSCRIPT=/usr/bin/sfupload
REVERSEMOUNT=/tmp/sf-mount
cat > $UPLOADSCRIPT << EOL
#!/bin/bash
if [ "\$(/usr/bin/whoami)" != "$USER" ]; then
   /bin/echo "This script must be run as user $USER" 1>&2
   exit 1
fi
# rootdir
ROOTDIR=\$1
if [ -z "\$ROOTDIR" ]; then
    /bin/echo "usage: \$0 path/to/rootdir"
    exit 1
fi
# download index
/bin/echo "download db ..."
/bin/rm /tmp/index.db &> /dev/null
HOME=$CONFFOLDER /usr/bin/rclone --config $RCLONECONFFILE copy upload:index.db /tmp
OLDSTATUS="\$(/bin/ls -l /tmp/index.db)"
# update DB
/bin/echo "scan rootdir for new files ..."
/usr/bin/splitfuse scan --dbfile /tmp/index.db --keyfile $SPLITKEYFILE --rootdir \$ROOTDIR
NEWSTATUS="\$(/bin/ls -l /tmp/index.db)"
# are there new files?
if [ "\$OLDSTATUS" == "\$NEWSTATUS" ]; then
   /bin/echo "nothing new" 1>&2
   exit 1
fi
# reverse mount
/bin/echo "reverse mount ..."
/bin/mkdir -p $REVERSEMOUNT
/usr/bin/splitfuse reverse --dbfile /tmp/index.db --keyfile $SPLITKEYFILE --rootdir \$ROOTDIR --mountdir $REVERSEMOUNT &
# best race condition fix ever !!
/bin/sleep 5
# upload with rclone
/bin/echo "start rclone sync ..."
HOME=$CONFFOLDER /usr/bin/rclone --config $RCLONECONFFILE copy --transfers 1 --size-only -v $REVERSEMOUNT upload:partstorage
HOME=$CONFFOLDER /usr/bin/rclone --config $RCLONECONFFILE copy /tmp/index.db upload:/
# unmount
/bin/echo "unmount ..."
/bin/fusermount -u $REVERSEMOUNT
EOL
chmod +x $UPLOADSCRIPT
