//go:build linux
// +build linux

package iface

import (
	"net"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

// as per https://github.com/torvalds/linux/blob/7ddb58cb0ecae8e8b6181d736a87667cc9ab8389/include/uapi/linux/route.h#L31-L48
type rtentry struct {
	pad1    uint
	dst     syscall.RawSockaddrInet4
	gateway syscall.RawSockaddrInet4
	genmask syscall.RawSockaddrInet4
	flags   uint16
	pad     [512]byte
}

type Configsocket struct {
	fd    int
	iface string
}

func NewConfigSocket(iface string) (Configsocket, error) {
	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, 0)
	if err != nil {
		return Configsocket{}, err
	}

	return Configsocket{
		fd:    fd,
		iface: iface,
	}, nil
}

func (cs Configsocket) Close() error {
	return syscall.Close(cs.fd)
}

func (cs Configsocket) ifreqAddr(request uint, addr net.IP) error {
	ifr, err := unix.NewIfreq(cs.iface)
	if err != nil {
		return err
	}

	// Ensure we use the 4-byte net.IP representation.
	if err := ifr.SetInet4Addr(addr.To4()); err != nil {
		return err
	}

	if err := unix.IoctlIfreq(cs.fd, request, ifr); err != nil {
		return err
	}

	return nil
}

func (cs Configsocket) SetAddress(addr net.IP) error {
	return cs.ifreqAddr(syscall.SIOCSIFADDR, addr)
}

func (cs Configsocket) SetNetmask(addr net.IPMask) error {
	return cs.ifreqAddr(syscall.SIOCSIFNETMASK, net.IP(addr))
}

func (cs Configsocket) SetBroadcast(addr net.IP) error {
	return cs.ifreqAddr(syscall.SIOCSIFBRDADDR, addr)
}

func (cs Configsocket) Up() error {
	ifr, err := unix.NewIfreq(cs.iface)
	if err != nil {
		return err
	}

	if err := unix.IoctlIfreq(cs.fd, unix.SIOCGIFFLAGS, ifr); err != nil {
		return err
	}

	flags := ifr.Uint16()
	flags |= syscall.IFF_UP
	flags |= syscall.IFF_RUNNING
	ifr.SetUint16(flags)

	if err := unix.IoctlIfreq(cs.fd, unix.SIOCSIFFLAGS, ifr); err != nil {
		return err
	}

	return nil
}

func (cs Configsocket) AddRoute(dst, gateway net.IP, genmask net.IPMask) syscall.Errno {
	req := rtentry{
		dst:     syscall.RawSockaddrInet4{Family: syscall.AF_INET},
		gateway: syscall.RawSockaddrInet4{Family: syscall.AF_INET},
		genmask: syscall.RawSockaddrInet4{Family: syscall.AF_INET},
		flags:   syscall.RTF_UP | syscall.RTF_GATEWAY,
	}
	copy(req.dst.Addr[:], dst)
	copy(req.gateway.Addr[:], gateway)
	copy(req.genmask.Addr[:], genmask)
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(cs.fd), syscall.SIOCADDRT, uintptr(unsafe.Pointer(&req)))
	return errno
}

func (cs Configsocket) DelRoute(dst, gateway net.IP, genmask net.IPMask) syscall.Errno {
	req := rtentry{
		dst:     syscall.RawSockaddrInet4{Family: syscall.AF_INET},
		gateway: syscall.RawSockaddrInet4{Family: syscall.AF_INET},
		genmask: syscall.RawSockaddrInet4{Family: syscall.AF_INET},
		flags:   syscall.RTF_UP | syscall.RTF_GATEWAY,
	}
	copy(req.dst.Addr[:], dst)
	copy(req.gateway.Addr[:], gateway)
	copy(req.genmask.Addr[:], genmask)
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(cs.fd), syscall.SIOCDELRT, uintptr(unsafe.Pointer(&req)))
	return errno
}
