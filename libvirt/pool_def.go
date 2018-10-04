package libvirt

import (
	/*	"encoding/xml"

		libvirt "github.com/libvirt/libvirt-go"*/
	"github.com/libvirt/libvirt-go-xml"
)

func newDefPool() libvirtxml.StoragePool {
	return libvirtxml.StoragePool{
		Target: &libvirtxml.StoragePoolTarget{},
	}
}

func newDefPoolSourceDevice() libvirtxml.StoragePoolSource {
	return libvirtxml.StoragePoolSource{
		Device: []libvirtxml.StoragePoolSourceDevice{libvirtxml.StoragePoolSourceDevice{}},
	}
}
