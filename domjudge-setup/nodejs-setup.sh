# in judgehost container
apt-get update && apt-get -y -q install software-properties-common python3
export CHROOTDIR=/chroot/domjudge
mount --bind /proc $CHROOTDIR/proc
chroot $CHROOTDIR /bin/sh -c "apt-get update && apt-get -q -y install curl software-properties-common && curl -sL https://deb.nodesource.com/setup_12.x | bash - && apt-get -q -y install nodejs && node -v"
umount $CHROOTDIR

grep ^domjudge /etc/passwd >> $CHROOTDIR/etc/passwd
grep ^domjudge /etc/shadow >> $CHROOTDIR/etc/shadow
