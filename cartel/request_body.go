package cartel

import "encoding/json"

// RequestBody contains parameters for Cartel calls
type RequestBody struct {
	Token         string            `json:"token,omitempty"`
	NameTag       []string          `json:"name-tag,omitempty"`
	Role          string            `json:"role,omitempty"`
	SecurityGroup []string          `json:"security_group,omitempty"`
	Image         string            `json:"image,omitempty"`
	LDAPGroups    []string          `json:"ldap_groups,omitempty"`
	InstanceType  string            `json:"instance_type,omitempty"`
	NumVolumes    int               `json:"num_vols,omitempty"`
	VolSize       int               `json:"vol_size,omitempty"`
	VolumeType    string            `json:"vol_type,omitempty"`
	IOPs          int               `json:"iops,omitempty"`
	EncryptVols   bool              `json:"encrypt_vols"`
	SubnetType    string            `json:"subnet_type,omitempty"`
	Subnet        string            `json:"subnet,omitempty"`
	Tags          map[string]string `json:"tags,omitempty"`
	Protect       bool              `json:"protect"`
	VpcId         string            `json:"vpc_id,omitempty"`
}

func (crb *RequestBody) ToJson() []byte {
	req, _ := json.Marshal(crb)
	return req
}

type RequestOptionFunc func(*RequestBody) error

// InstanceType sets the instance type
func InstanceType(instanceType string) RequestOptionFunc {
	return func(body *RequestBody) error {
		body.InstanceType = instanceType
		return nil
	}
}

// EncryptVolumes enables volume encryption
func EncryptVolumes() RequestOptionFunc {
	return func(body *RequestBody) error {
		body.EncryptVols = true
		return nil
	}
}

// UserGroups sets the user groups (LDAP groups)
func UserGroups(groups ...string) RequestOptionFunc {
	return func(body *RequestBody) error {
		body.LDAPGroups = groups
		return nil
	}
}

// SecurityGroups sets the security groups
func SecurityGroups(groups ...string) RequestOptionFunc {
	return func(body *RequestBody) error {
		body.SecurityGroup = groups
		return nil
	}
}

// VolumesAndSize sets the number of volumes to attach and their size (in GB)
func VolumesAndSize(nrVols, size int) RequestOptionFunc {
	return func(body *RequestBody) error {
		body.VolSize = size
		body.NumVolumes = nrVols
		return nil
	}
}

// VolumeType sets the EBS volume type
func VolumeType(volumeType string) RequestOptionFunc {
	return func(body *RequestBody) error {
		body.VolumeType = volumeType
		return nil
	}
}

// IOPs sets the number of IOPs to provision for attached storage
func IOPs(iops int) RequestOptionFunc {
	return func(body *RequestBody) error {
		body.IOPs = iops
		return nil
	}
}

// SubnetType sets the subnet type
func SubnetType(subnetType string) RequestOptionFunc {
	return func(body *RequestBody) error {
		body.SubnetType = subnetType
		return nil
	}
}

// InSubnet sets the subnet
func InSubnet(subnetID string) RequestOptionFunc {
	return func(body *RequestBody) error {
		body.Subnet = subnetID
		return nil
	}
}

// VPICID Sets the VPC ID to use
func VPCID(vpcID string) RequestOptionFunc {
	return func(body *RequestBody) error {
		body.VpcId = vpcID
		return nil
	}
}
