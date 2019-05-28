## DOMJudge setup instructions

* Follow instructions here: https://hub.docker.com/r/domjudge/domserver/
* Login to DOMJudge here using admin credentials shared  to you
* Change judgehost password to `password` or set something different and set env variable in `judgehost` container
* Edit `/etc/default/grub` and the following lines

## Create MariaDB RDS instance

* https://aws.amazon.com/premiumsupport/knowledge-center/duplicate-master-user-mysql/

```bash
mysql -h domjudge-db.c97ivjugwy4b.us-east-1.rds.amazonaws.com -uroot domjudge -p
GRANT ALL PRIVILEGES ON domjudge . * TO 'domjudge';
GRANT SELECT, INSERT, UPDATE, DELETE, CREATE, DROP, RELOAD, PROCESS, REFERENCES, INDEX, ALTER, SHOW DATABASES, CREATE TEMPORARY TABLES, LOCK TABLES, EXECUTE, REPLICATION SLAVE, REPLICATION CLIENT, CREATE VIEW, SHOW VIEW, CREATE ROUTINE, ALTER ROUTINE, CREATE USER, EVENT, TRIGGER ON *.* TO 'domjudge'@'%' WITH GRANT OPTION;
FLUSH PRIVILEGES;
```

## Update domjudge instance Ubuntu settings

```bash
GRUB_CMDLINE_LINUX_DEFAULT="cgroup_enable=memory swapaccount=1 console=tty1 console=ttyS0"
GRUB_CMDLINE_LINUX="cgroup_enable=memory swapaccount=1 console=tty1 console=ttyS0"
```

* Add following shell script upon container start to `judgehost` container:

```bash
# in judgehost container
apt-get update && apt-get -y -q install software-properties-common python3
export CHROOTDIR=/chroot/domjudge
mount --bind /proc $CHROOTDIR/proc
chroot $CHROOTDIR /bin/sh -c "apt-get update && apt-get -q -y install software-properties-common
python3"
umount $CHROOTDIR

grep ^domjudge /etc/passwd >> $CHROOTDIR/etc/passwd
grep ^domjudge /etc/shadow >> $CHROOTDIR/etc/shadow
```

* Add following shell script to mariadb container:

```bash
# in mariadb container
mysql -h localhost -udomjudge -pdjpw domjudge -e "UPDATE language SET allow_submit=1 WHERE langid = 'py3';"
```

* Perform an update on DOMJudge cluster: 

```bash
docker-compose up -d
```

## Resources

* Python support
  * https://www.domjudge.org/docs/admin-manual-3.html
  * https://www.domjudge.org/pipermail/domjudge-devel/2013-February/001138.html
  * https://github.com/DOMjudge/domjudge-packaging/blob/master/docker/judgehost/Dockerfile
  * https://github.com/DOMjudge/domjudge-packaging/commit/a61d87b

