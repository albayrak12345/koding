package main

import (
	"errors"
	"fmt"
	"koding/kites/kloud/packer"
	"net/url"
	"strconv"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/mitchellh/packer/builder/digitalocean"
)

type Event struct {
	Id           int
	DropletID    int
	EventTypeID  int
	ActionStatus string
	Percentage   string
}

type DropletInfo struct {
	Id       int    `json:"id"`
	Hostname string `json:"hostname"`
	ImageId  int    `json:"image_id"`
	SizeId   string `json:"size_id"`
	EventId  int    `json:"event_id"`
}

type DigitalOcean struct {
	Client *digitalocean.DigitalOceanClient
	Name   string

	Creds struct {
		ClientID string `mapstructure:"client_id"`
		APIKey   string `mapstructure:"api_key"`
	}

	Builder struct {
		Type     string `mapstructure:"type"`
		ClientID string `mapstructure:"client_id"`
		APIKey   string `mapstructure:"api_key"`

		RegionID uint `mapstructure:"region_id"`
		SizeID   uint `mapstructure:"size_id"`
		ImageID  uint `mapstructure:"image_id"`

		Region string `mapstructure:"region"`
		Size   string `mapstructure:"size"`
		Image  string `mapstructure:"image"`

		PrivateNetworking bool   `mapstructure:"private_networking"`
		SnapshotName      string `mapstructure:"snapshot_name"`
		DropletName       string `mapstructure:"droplet_name"`
		SSHUsername       string `mapstructure:"ssh_username"`
		SSHPort           uint   `mapstructure:"ssh_port"`

		RawSSHTimeout   string `mapstructure:"ssh_timeout"`
		RawStateTimeout string `mapstructure:"state_timeout"`
	}
}

func (d *DigitalOcean) Prepare(raws ...interface{}) (err error) {
	d.Name = "digitalocean"
	if len(raws) != 2 {
		return errors.New("need at least two arguments")
	}

	// Credentials
	if err := mapstructure.Decode(raws[0], &d.Creds); err != nil {
		return err
	}

	// Builder data
	if err := mapstructure.Decode(raws[1], &d.Builder); err != nil {
		return err
	}

	d.Client = digitalocean.DigitalOceanClient{}.New(d.Creds.ClientID, d.Creds.APIKey)
	return nil
}

func (d *DigitalOcean) Build() (err error) {
	snapshotName := "koding-" + strconv.FormatInt(time.Now().UTC().Unix(), 10)
	d.Builder.SnapshotName = snapshotName

	data, err := templateData(d.Builder)
	if err != nil {
		return err
	}

	provider := &packer.Provider{
		BuildName: "digitalocean",
		Data:      data,
	}
	fmt.Printf("provider %+v\n", provider)

	// // this is basically a "packer build template.json"
	// if err := provider.Build(); err != nil {
	// 	return err
	// }

	// after creating the image go and get it
	images, err := d.MyImages()
	if err != nil {
		return err
	}

	var image digitalocean.Image
	for _, i := range images {
		if i.Name == snapshotName {
			image = i
		}
	}

	if image.Id == 0 {
		return fmt.Errorf("Image %s is not available in Digital Ocean", snapshotName)
	}

	// now create a the machine based on our created image
	dropletInfo, err := d.CreateDroplet("arslannew", image.Id)
	if err != nil {
		return err
	}

	fmt.Printf("dropletInfo %+v\n", dropletInfo)

	return nil
}

// CreateDroplet creates a new droplet with a hostname and the given image_id
func (d *DigitalOcean) CreateDroplet(hostname string, image_id uint) (*DropletInfo, error) {
	params := url.Values{}
	params.Set("name", hostname)

	found_size, err := d.Client.Size(d.Builder.Size)
	if err != nil {
		return nil, fmt.Errorf("Invalid size or lookup failure: '%s': %s", d.Builder.Size, err)
	}

	found_region, err := d.Client.Region(d.Builder.Region)
	if err != nil {
		return nil, fmt.Errorf("Invalid region or lookup failure: '%s': %s", d.Builder.Region, err)
	}

	params.Set("size_slug", found_size.Slug)
	params.Set("image_id", strconv.Itoa(int(image_id)))
	params.Set("region_slug", found_region.Slug)
	params.Set("private_networking", fmt.Sprintf("%v", d.Builder.PrivateNetworking))

	body, err := digitalocean.NewRequest(*d.Client, "droplets/new", params)
	if err != nil {
		return nil, err
	}

	info := &DropletInfo{}
	if err := mapstructure.Decode(body, info); err != nil {
		return nil, err
	}

	return info, nil
}

func (d *DigitalOcean) MyImages() ([]digitalocean.Image, error) {
	v := url.Values{}
	v.Set("filter", "my_images")

	resp, err := digitalocean.NewRequest(*d.Client, "images", v)
	if err != nil {
		return nil, err
	}

	var result digitalocean.ImagesResp
	if err := mapstructure.Decode(resp, &result); err != nil {
		return nil, err
	}

	return result.Images, nil
}

func (d *DigitalOcean) Start() error   { return nil }
func (d *DigitalOcean) Stop() error    { return nil }
func (d *DigitalOcean) Restart() error { return nil }
func (d *DigitalOcean) Destroy() error { return nil }
