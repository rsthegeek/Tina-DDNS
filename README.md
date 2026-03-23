```bash
docker save ghcr.io/rsthegeek/tina-ddns:latest -o tina-ddns.tar
# Upload the .tar to MikroTik
/container/mounts
add name=ddns-config src=/config.json dst=/app/config.json

/container/add file=tina-ddns.tar \
    interface=veth1 \
    root-dir=usb1/containers/images \
    name=ddns start-on-boot=yes logging=yes \
    mounts=ddns-config workdir=/app
```