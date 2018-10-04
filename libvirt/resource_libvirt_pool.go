package libvirt

import (
	"encoding/xml"
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	libvirt "github.com/libvirt/libvirt-go"
)

func resourceLibvirtPool() *schema.Resource {
	return &schema.Resource{
		Create: resourceLibvirtPoolCreate,
		Update: resourceLibvirtPoolUpdate,
		Read:   resourceLibvirtPoolRead,
		Delete: resourceLibvirtPoolDelete,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"type": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "dir",
				ForceNew: true,
			},
			"target": {
				Type:     schema.TypeMap,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"path": {
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
					},
				},
			},
			"uuid": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"autostart": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
				ForceNew: false,
			},
			"start": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
				ForceNew: false,
			},
			"build": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
				ForceNew: false,
			},
			"source_device": {
				Type:     schema.TypeMap,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"path": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
					},
				},
			},
		},
	}
}

func resourceLibvirtPoolRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Client)
	virConn := client.libvirt
	if virConn == nil {
		return fmt.Errorf(LibVirtConIsNil)
	}

	storagePool, err := virConn.LookupStoragePoolByName(d.Id())
	if err != nil {
		return fmt.Errorf("Error lookup up storage pool by name: %s\n", err)
	}
	defer storagePool.Free()

	uuid, err := storagePool.GetUUIDString()
	if err != nil {
		return fmt.Errorf("Error lookup up storage pool uuid: %s\n", err)
	}
	log.Printf("[DEBUG] uuid: %s, storagePool: %+v\n", uuid, storagePool)
	d.Set("uuid", uuid)

	autostart, err := storagePool.GetAutostart()
	if err != nil {
		return fmt.Errorf("error lookup up storage pool autostart: %s\n", err)
	}
	d.Set("autostart", autostart)

	return nil
}

func resourceLibvirtPoolCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Client)
	virConn := client.libvirt
	if virConn == nil {
		return fmt.Errorf(LibVirtConIsNil)
	}

	poolDef := newDefPool()
	poolDef.Name = d.Get("name").(string)
	poolDef.Type = d.Get("type").(string)
	poolDef.Target.Path = d.Get("target.path").(string)

	if poolDef.Type == "fs" {
		sourceDevicePath, pathIsSet := d.Get("source_device.path").(string)
		if !pathIsSet {
			return fmt.Errorf("Error getting source device path for fs")
		}
		sourceDeviceDef := newDefPoolSourceDevice()
		sourceDeviceDef.Device[0].Path = sourceDevicePath
		log.Printf("[DEBUG] sourceDevicePath: %s", sourceDevicePath)
		poolDef.Source = &sourceDeviceDef
	}

	log.Printf("[DEBUG] poolDef: %+v, target: %+v\n", poolDef, poolDef.Target)

	poolDefXML, err := xml.Marshal(poolDef)
	if err != nil {
		return fmt.Errorf("Error serializing libvirt pool: %s", err)
	}
	log.Printf("[DEBUG] poolDefXML: %s", poolDefXML)

	pool, err := virConn.StoragePoolDefineXML(string(poolDefXML), 0)
	if err != nil {
		return fmt.Errorf("Error creating libvirt pool: %s", err)
	}

	d.Partial(true)

	name, err := pool.GetName()
	if err != nil {
		return fmt.Errorf("Error creating libvirt pool, while fetching name: %s", err)
	}
	d.SetId(name)
	d.Set("name", name)

	uuid, err := pool.GetUUIDString()
	if err != nil {
		return fmt.Errorf("Error creating libvirt pool: %s", err)
	}
	d.Set("uuid", uuid)

	autostart := d.Get("autostart").(bool)
	if autostart {
		err = pool.SetAutostart(autostart)
		if err != nil {
			return fmt.Errorf("Error setting libvirt pool autostart: %s", err)
		}
	}

	build := d.Get("build").(bool)
	if build {
		err = pool.Build(libvirt.STORAGE_POOL_BUILD_NO_OVERWRITE)
		if err != nil {
			return fmt.Errorf("Error building libvirt pool: %s", err)
		}
	}

	start := d.Get("start").(bool)
	if start {
		err = pool.Create(libvirt.STORAGE_POOL_CREATE_NORMAL)
		if err != nil {
			return fmt.Errorf("Error creating libvirt pool: %s", err)
		}
	}

	d.Partial(false)
	return nil
}

func resourceLibvirtPoolDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Client)
	virConn := client.libvirt
	if virConn == nil {
		return fmt.Errorf(LibVirtConIsNil)
	}

	storagePool, err := virConn.LookupStoragePoolByUUIDString(d.Get("uuid").(string))
	if err != nil {
		return fmt.Errorf("Error lookup up storage pool by uuid: %s\n", err)
	}

	isActive, err := storagePool.IsActive()
	if err != nil {
		return fmt.Errorf("Error lookup up if storage pool if active: %s\n", err)
	}

	if isActive {
		err = storagePool.Destroy()
		if err != nil {
			return fmt.Errorf("Error destroying storage pool: %s\n", err)
		}
	}

	err = storagePool.Undefine()
	if err != nil {
		return fmt.Errorf("Error undefining storage pool: %s\n", err)
	}

	return nil
}

func resourceLibvirtPoolExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(*Client)
	virConn := client.libvirt
	if virConn == nil {
		return false, fmt.Errorf(LibVirtConIsNil)
	}

	storagePool, err := virConn.LookupStoragePoolByName(d.Get("name").(string))
	if err != nil {
		return false, fmt.Errorf("error lookup up storage pool by name: %s\n", err)
	}
	if storagePool == nil {
		return false, nil
	}
	defer storagePool.Free()

	log.Println("[DEBUG] storagePool: %+v\n", storagePool)
	if storagePool == nil {
		return false, nil
	}

	return true, nil
}

func resourceLibvirtPoolUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Client)
	virConn := client.libvirt
	if virConn == nil {
		return fmt.Errorf(LibVirtConIsNil)
	}

	storagePool, err := virConn.LookupStoragePoolByName(d.Get("name").(string))
	if err != nil {
		return fmt.Errorf("Error lookup up storage pool by name: %s\n", err)
	}
	defer storagePool.Free()
	d.Partial(true)

	if d.HasChange("autostart") {
		autostart := d.Get("autostart").(bool)
		log.Printf("[DEBUG] autostart has change to %t", autostart)

		err = storagePool.SetAutostart(autostart)
		if err != nil {
			return fmt.Errorf("Error changing storage pool autostart: %s\n", err)
		}
	}

	d.Partial(false)
	return nil
}
