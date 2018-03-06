# splitfuse

#### Installation
 - apt install *git*
 - apt install *fuse*
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

```
wget -O /tmp/setup.sh https://raw.githubusercontent.com/SchnorcherSepp/splitfuse/master/setup.sh
chmod +x /tmp/setup.sh
sudo /tmp/setup.sh
```
