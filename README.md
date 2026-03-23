```bash
/container/mounts
add name=ddns-config src=/config.json dst=/app/config.json

/container/add remote-image=ghcr.io/rsthegeek/tina-ddns:latest \
    interface=veth1 \
    root-dir=usb1/containers/images \
    name=ddns start-on-boot=yes logging=yes \
    mounts=ddns-config
```