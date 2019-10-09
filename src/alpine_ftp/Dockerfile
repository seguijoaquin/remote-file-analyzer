FROM alpine:latest

RUN apk add vsftpd
COPY vsftpd.conf /etc/vsftpd/vsftpd.conf
COPY vsftpd_start.sh /vsftpd_start.sh
RUN chmod +x /vsftpd_start.sh
EXPOSE 21

CMD "/vsftpd_start.sh"

