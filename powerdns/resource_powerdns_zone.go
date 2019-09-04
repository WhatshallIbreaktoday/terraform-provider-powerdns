package powerdns

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourcePDNSZone() *schema.Resource {
	return &schema.Resource{
		Create: resourcePDNSZoneCreate,
		Read:   resourcePDNSZoneRead,
		Update: resourcePDNSZoneUpdate,
		Delete: resourcePDNSZoneDelete,
		Exists: resourcePDNSZoneExists,
		Importer: &schema.ResourceImporter{
			State: resourcePDNSZoneImport,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"kind": {
				Type:     schema.TypeString,
				Required: true,
			},

			"nameservers": {
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourcePDNSZoneCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Client)

	var nameservers []string
	for _, nameserver := range d.Get("nameservers").(*schema.Set).List() {
		nameservers = append(nameservers, nameserver.(string))
	}

	zoneInfo := ZoneInfo{
		Name:        d.Get("name").(string),
		Kind:        d.Get("kind").(string),
		Nameservers: nameservers,
	}

	createdZoneInfo, err := client.CreateZone(zoneInfo)
	if err != nil {
		return err
	}

	d.SetId(createdZoneInfo.ID)

	return nil
}

func resourcePDNSZoneRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Client)

	log.Printf("[DEBUG] Reading PowerDNS Zone: %s", d.Id())
	zoneInfo, err := client.GetZone(d.Get("name").(string))
	if err != nil {
		return fmt.Errorf("Couldn't fetch PowerDNS Zone: %s", err)
	}

	d.Set("name", zoneInfo.Name)
	d.Set("kind", zoneInfo.Kind)

	return nil
}

func resourcePDNSZoneUpdate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Updating PowerDNS Zone: %s", d.Id())

	client := meta.(*Client)

	zoneInfo := ZoneInfo{}
	shouldUpdate := false
	if d.HasChange("kind") {
		zoneInfo.Kind = d.Get("kind").(string)
		shouldUpdate = true
	}

	if shouldUpdate {
		return client.UpdateZone(d.Id(), zoneInfo)
	}
	return nil
}

func resourcePDNSZoneDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Client)

	log.Printf("[INFO] Deleting PowerDNS Zone: %s", d.Id())
	err := client.DeleteZone(d.Id())

	if err != nil {
		return fmt.Errorf("Error deleting PowerDNS Zone: %s", err)
	}
	return nil
}

func resourcePDNSZoneExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	name := d.Get("name").(string)

	log.Printf("[INFO] Checking existence of PowerDNS Zone: %s", name)

	client := meta.(*Client)
	exists, err := client.ZoneExists(name)

	if err != nil {
		return false, fmt.Errorf("Error checking PowerDNS Zone: %s", err)
	}
	return exists, nil
}

func resourcePDNSZoneImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {

	client := meta.(*Client)

	var data map[string]string
	if err := json.Unmarshal([]byte(d.Id()), &data); err != nil {
		return nil, err
	}

	zoneName, ok := data["name"]
	if !ok {
		return nil, errors.New("missing zone name in input data")
	}

	log.Printf("[INFO] importing PowerDNS Zone: %s", zoneName)

	zone, err := client.GetZone(zoneName)
	if err != nil {
		return nil, fmt.Errorf("couldn't fetch zone %s from PowerDNS: %v", zoneName, err)
	}

	nameservers, err := client.ListRecordsInRRSet(zoneName, zoneName, "NS")
	if err != nil {
		return nil, fmt.Errorf("couldn't fetch zone %s nameservers from PowerDNS: %v", zoneName, err)
	}

	var zoneNameservers []string
	for _, nameserver := range nameservers {
		zoneNameservers = append(zoneNameservers, nameserver.Content)
	}

	d.Set("name", zone.Name)
	d.Set("kind", zone.Kind)
	d.Set("nameservers", zoneNameservers)
	d.SetId(zoneName)

	return []*schema.ResourceData{d}, nil
}
