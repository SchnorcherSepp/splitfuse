# splitfuse
[microbadger]: https://microbadger.com/images/schnorchersepp/splitfuse
[dockerstore]: https://store.docker.com/community/images/schnorchersepp/splitfuse

[![Docker Layers](https://images.microbadger.com/badges/image/schnorchersepp/splitfuse.svg)][microbadger]
[![Docker Build Status](https://img.shields.io/docker/build/schnorchersepp/splitfuse.svg)][dockerstore]


#### Installation
 - apt install *git*
 - apt install *fuse*
 - enable user_allow_other in /etc/fuse.conf
 - get from github.com, build with go-1.10 and install */usr/bin/splitfuse* (main branch)
 - get from github.com, build with go-1.10 and install */usr/bin/rclone* (main branch)

```
wget -O /tmp/install.sh https://raw.githubusercontent.com/SchnorcherSepp/splitfuse/master/install.sh
chmod +x /tmp/install.sh
sudo /tmp/install.sh
```


#### Setup
 - add system user *splitfuse* (no-create-home, disabled-login, disabled-password)
 - create config folder */etc/splitfuse*
 - create mount script */usr/bin/sfmount* (mount rclone and splitfuse)
 - create upload script */usr/bin/sfupload* (sync local storage (plain) with online storage (encrypt))

```
wget -O /tmp/setup.sh https://raw.githubusercontent.com/SchnorcherSepp/splitfuse/master/setup.sh
chmod +x /tmp/setup.sh
sudo /tmp/setup.sh
```
