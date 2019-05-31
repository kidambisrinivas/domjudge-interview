cat /home/domjudge/configs/python-setup.sh
# in judgehost container
apt-get update && apt-get -y -q install software-properties-common python3
export CHROOTDIR=/chroot/domjudge
mount --bind /proc $CHROOTDIR/proc
chroot $CHROOTDIR /bin/sh -c "apt-get update && apt-get -q -y install software-properties-common python3"
umount $CHROOTDIR

grep ^domjudge /etc/passwd >> $CHROOTDIR/etc/passwd
grep ^domjudge /etc/shadow >> $CHROOTDIR/etc/shadow
