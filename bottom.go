// filename: bottom.go
// author: Jianing YANG <jianingy.yang@gmail.com>
// created at: Oct. 2013


// A dnsmasq wrapper for pxeboot purpose
//
// Example:
//
// sudo ./bottom -tftproot tftproot -dnsmasq /usr/local/sbin/dnsmasq -ip 192.168.56.101 -mac 08:00:27:AE:0D:26 -append "ks=http://192.168.56.1/ks.cfg ksdevice=eth1"
//
package main

import (
    "bytes"
	"flag"
    "net"
    "fmt"
    "path"
    "path/filepath"
    "strings"
    "os"
    "os/exec"
    "text/template"
)

const (
    PXECONFIG = `
DEFAULT default
LABEL default
  KERNEL {{.Kernel}}
  INITRD {{.Initrd}}
  APPEND {{.Append}}
`)

type BootService struct {
    Dnsmasq         string
    Directory       string
    TargetIP        net.IP
    TargetMAC       net.HardwareAddr
    Bind            net.IP
    NetworkId       net.IP
    NetworkMask     net.IP
    Rom             string
    Kernel          string
    Initrd          string
    Append          string
}

func main()  {
    service, err := newBootService(os.Args[1:])
    if err != nil {
        fmt.Printf("Argument error: %s\n", err)
    }

    if err := service.Start(); err != nil {
        fmt.Printf("Service error: %s\n", err)
    }
}

func newBootService(args []string) (*BootService, error) {
    var err error
    service := BootService{}

    flags := flag.NewFlagSet("", flag.ExitOnError)

    strIP := flags.String("ip", "", "IP address of the new server")
    strMAC := flags.String("mac", "", "MAC address of the new server")
    tftproot := flags.String("tftproot", ".", "Path to tftproot")

    flags.StringVar(&service.Dnsmasq, "dnsmasq", "/usr/bin/dnsmasq", "Path to dnsmasq")
    flags.StringVar(&service.Rom, "rom", "pxelinux.0", "Path to boot rom ")
    flags.StringVar(&service.Kernel, "kernel", "vmlinuz", "Path to installation kernel")
    flags.StringVar(&service.Initrd, "initrd", "initrd.img", "Path to installation init ramdisk")
    flags.StringVar(&service.Append, "append", "", "Commands append to initrd")


    flags.Parse(args)

    if service.Directory, err = filepath.Abs(*tftproot); err != nil {
        return nil, err
    }

    service.TargetIP = net.ParseIP(*strIP)
    if service.TargetIP == nil {
        return nil, fmt.Errorf("invalid IPv4 address: %s", *strIP)
    }

    service.TargetMAC, err = net.ParseMAC(*strMAC)
    if err != nil {
        return nil, err
    }

    if ifaces, err := net.Interfaces(); err != nil {
        return nil, err
    } else {
    FIND_BIND_ADDRESS:
        for _, iface := range ifaces {
            // skip loopback devices
            if strings.HasPrefix(iface.Name, "lo") {
                continue
            }
            if addrs, err := iface.Addrs(); err != nil || len(addrs) < 1 {
                // skip bad interfaces and those without ip addresses
                continue
            } else {
                for _, addr := range addrs {
                    if addrIP, addrNet, err := net.ParseCIDR(addr.String()); err != nil {
                        continue
                    } else {
                        if addrNet.Contains(service.TargetIP) {
                            service.Bind = addrIP
                            mask := []byte(addrNet.Mask)
                            service.NetworkId = addrNet.IP
                            service.NetworkMask = net.IPv4(mask[0], mask[1], mask[2], mask[3])
                            break FIND_BIND_ADDRESS
                        }
                    }
                }
            }
        }
    }

    return &service, nil
}

func checkBinary(binary string) error {
   // check if dnsmasq exists
    if fi, err := os.Stat(binary); err != nil {
        if os.IsNotExist(err) {
            return fmt.Errorf("no such file or directory: %s", binary)
        } else {
            return fmt.Errorf("error on reading %s", binary)
        }
    } else {
        if fi.IsDir() {
            return fmt.Errorf("%s is a directory", binary)
        }
    }
    return nil
}

func (service *BootService) Start() error {
    if err := checkBinary(service.Dnsmasq); err != nil {
        return err
    }

    // generate pxelinux.cfg
    conf := path.Join(service.Directory, "pxelinux.cfg", "default")
    if err := os.MkdirAll(filepath.Dir(conf), 0755); err != nil {
        return err
    }

    t := template.New("pxelinux.cfg/default")
    t, err := t.Parse(PXECONFIG);
    if err != nil {
        return err
    }
    if fo, err := os.Create(conf); err != nil {
        return err
    } else {
        t.Execute(fo, map[string]string {
            "Kernel": service.Kernel,
            "Initrd": service.Initrd,
            "Append": service.Append,
        })
    }

    // run dnsmasq command
    cmdPXE := []string{
        service.Dnsmasq,
        "-k",             // keep int foreground
        "-p", "0",        // disable DNS service
        "--log-dhcp",     // log dhcp events
        "--enable-tftp",  // enable tftp service
        fmt.Sprintf("--listen-address=%s", service.Bind),
        fmt.Sprintf("--tftp-root=%s", service.Directory),
        fmt.Sprintf("--dhcp-leasefile=%s", filepath.Join(service.Directory, "dnsmasq.lease")),
        fmt.Sprintf("--dhcp-range=%s,static,%s", service.NetworkId.String(), service.NetworkMask.String()),
        fmt.Sprintf("--dhcp-host=%s,%s", service.TargetMAC, service.TargetIP),
        fmt.Sprintf("--dhcp-boot=%s", service.Rom),
        fmt.Sprintf("--dhcp-option=6,114.114.114.114"),
    }
    fmt.Printf("Running %s\n", cmdPXE)
    cmd := exec.Command(cmdPXE[0], cmdPXE[1:]...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
    if err := cmd.Run(); err != nil {
        return fmt.Errorf("%s", stderr.String())
    }
    return nil
}
