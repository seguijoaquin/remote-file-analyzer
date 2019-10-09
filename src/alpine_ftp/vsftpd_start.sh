#! /bin/ash
echo -e "$PASS\n$PASS" | adduser -h $FOLDER -s /sbin/nologin $NAME
mkdir -p $FOLDER
chown $NAME:$NAME $FOLDER
vsftpd /etc/vsftpd/vsftpd.conf
