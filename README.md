Bottom: a lightweight CentOS/RHEL netinstall tool
=================================================

Bottom is virtually a Dnsmasq wrapper that it generates a PXE
configuration and runs a dnsmasq based on the given arguments.

A typical usage example is,

```bash
sudo ./bottom -tftproot /tmp/tftproot -ip 192.168.56.101 -mac 08:00:27:AE:0D:26 -append "ks=http://example.com/ks.cfg ksdevice=eth1"
```

To make this example work, You have to put a 'pxelinux.0' in the '/tmp/tftpboot' directory.

Additional useful parameters are:

  * *dnsmasq*: choose an alternative dnsmasq binary
  * *rom*: choose an alternative pxelinux binary (binary must exist in tftproot). e.g, gpxelinux.0

Other parameters please refer to the source code.
