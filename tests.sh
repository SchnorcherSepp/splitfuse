#!/bin/bash
WORKDIR=/tmp/splitfuse-test
ORIGDIR=$WORKDIR/files
MOUNTREVDIR=$WORKDIR/mount-parts
PARTSTORAGE=$WORKDIR/parts
FINALMOUNT=$WORKDIR/mount-files

# create folders
mkdir -p $ORIGDIR
mkdir -p $MOUNTREVDIR
mkdir -p $PARTSTORAGE
mkdir -p $FINALMOUNT

# create random test files (if not exist)
if [ ! -f $ORIGDIR/0.zero ]; then
  echo "create random test files (take some minutes)"
  # leere Datei
  touch $ORIGDIR/0.zero
  # FUSE limit 131072
  dd if=/dev/urandom of=$ORIGDIR/131071.rand bs=131071 count=1 iflag=fullblock
  dd if=/dev/urandom of=$ORIGDIR/131072.rand bs=131072 count=1 iflag=fullblock
  dd if=/dev/urandom of=$ORIGDIR/131073.rand bs=131073 count=1 iflag=fullblock
  # irgend ein interner Puffer 4096
  dd if=/dev/urandom of=$ORIGDIR/4095.rand bs=4095 count=1 iflag=fullblock
  dd if=/dev/urandom of=$ORIGDIR/4096.rand bs=4096 count=1 iflag=fullblock
  dd if=/dev/urandom of=$ORIGDIR/4097.rand bs=4097 count=1 iflag=fullblock
  # Partsize 1073741824
  dd if=/dev/urandom of=$ORIGDIR/1073741808.rand bs=67108863 count=16 iflag=fullblock
  dd if=/dev/urandom of=$ORIGDIR/1073741824.rand bs=67108864 count=16 iflag=fullblock
  dd if=/dev/urandom of=$ORIGDIR/1073741840.rand bs=67108865 count=16 iflag=fullblock
  # read buffer 16777216
  dd if=/dev/urandom of=$ORIGDIR/16777215.rand bs=16777216 count=1 iflag=fullblock
  dd if=/dev/urandom of=$ORIGDIR/16777216.rand bs=16777216 count=1 iflag=fullblock
  dd if=/dev/urandom of=$ORIGDIR/16777217.rand bs=16777216 count=1 iflag=fullblock

  # UND NOCHMAL ALLES FUER ZERO
  dd if=/dev/zero of=$ORIGDIR/131071.zero bs=131071 count=1 iflag=fullblock
  dd if=/dev/zero of=$ORIGDIR/131072.zero bs=131072 count=1 iflag=fullblock
  dd if=/dev/zero of=$ORIGDIR/131073.zero bs=131073 count=1 iflag=fullblock
  dd if=/dev/zero of=$ORIGDIR/4095.zero bs=4095 count=1 iflag=fullblock
  dd if=/dev/zero of=$ORIGDIR/4096.zero bs=4096 count=1 iflag=fullblock
  dd if=/dev/zero of=$ORIGDIR/4097.zero bs=4097 count=1 iflag=fullblock
  dd if=/dev/zero of=$ORIGDIR/1073741808.zero bs=67108863 count=16 iflag=fullblock
  dd if=/dev/zero of=$ORIGDIR/1073741824.zero bs=67108864 count=16 iflag=fullblock
  dd if=/dev/zero of=$ORIGDIR/1073741840.zero bs=67108865 count=16 iflag=fullblock
  dd if=/dev/zero of=$ORIGDIR/16777215.zero bs=16777216 count=1 iflag=fullblock
  dd if=/dev/zero of=$ORIGDIR/16777216.zero bs=16777216 count=1 iflag=fullblock
  dd if=/dev/zero of=$ORIGDIR/16777217.zero bs=16777216 count=1 iflag=fullblock
fi

# keyfile und db-scan
echo "scan rootdir"
if [ ! -f $WORKDIR/keyfile ]; then
  splitfuse newkey --keyfile $WORKDIR/keyfile
fi
splitfuse scan --dbfile $WORKDIR/dbfile --keyfile $WORKDIR/keyfile --rootdir $ORIGDIR

# MOUNT REVERSE
echo "mount reverse (take some minutes)"
splitfuse reverse --dbfile $WORKDIR/dbfile --keyfile $WORKDIR/keyfile --rootdir $ORIGDIR --mountdir $MOUNTREVDIR --debug &> /tmp/reverse.log  &
while [ ! -d $MOUNTREVDIR/08 ]; do
   /bin/sleep 2
done

# kopieren von reverse nach parts
echo "copy reverse files (partstorage)"
rsync -avzht $MOUNTREVDIR $PARTSTORAGE
# umount
fusermount -u $MOUNTREVDIR

# MOUNT NORMAL
echo "mount normal"
splitfuse normal --dbfile $WORKDIR/dbfile --keyfile $WORKDIR/keyfile --chunkdir $PARTSTORAGE/mount-parts --mountdir $FINALMOUNT --debug &> /tmp/normal.log &
sleep 2

##############################################################################

echo "#####################"
echo "## START DER TESTS ##"
echo "#####################"

echo "test" > $ORIGDIR/diese-eine-datei-darf-unterschied-sein.txt
rsync -avzht --dry-run --stats  $ORIGDIR/ $FINALMOUNT/
rm $ORIGDIR/diese-eine-datei-darf-unterschied-sein.txt

echo "#####################"
echo "md5 Hash; es dürfte hier also nichts erscheinen"
ORIGMD=`md5sum $ORIGDIR/* | awk '{ print $1 }'`
FINAMD=`md5sum $FINALMOUNT/* | awk '{ print $1 }'`
if [ "$ORIGMD" != "$FINAMD" ]; then
    echo "MD5 ERROR!!!!!!!!!!"
    echo ""
    echo "$ORIGMD"
    echo ""
    echo "$FINAMD"
    echo ""
fi

echo "#####################"
echo "nun wird ein diff gefahren; es dürfte hier also nichts erscheinen"
# simuliere parallelen Zugriff
diff -qr $ORIGDIR $FINALMOUNT &
diff -qr $ORIGDIR $FINALMOUNT &
diff -qr $ORIGDIR $FINALMOUNT &
diff -qr $ORIGDIR $FINALMOUNT &
diff -qr $ORIGDIR $FINALMOUNT

# umount
sleep 10
fusermount -u $FINALMOUNT
