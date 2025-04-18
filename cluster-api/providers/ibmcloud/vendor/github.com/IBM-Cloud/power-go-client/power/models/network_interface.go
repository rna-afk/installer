// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
	"github.com/go-openapi/validate"
)

// NetworkInterface network interface
//
// swagger:model NetworkInterface
type NetworkInterface struct {

	// The Network Interface's crn
	// Required: true
	Crn *string `json:"crn"`

	// The unique Network Interface ID
	// Required: true
	ID *string `json:"id"`

	// instance
	Instance *NetworkInterfaceInstance `json:"instance,omitempty"`

	// The ip address of this Network Interface
	// Required: true
	IPAddress *string `json:"ipAddress"`

	// The mac address of the Network Interface
	// Required: true
	MacAddress *string `json:"macAddress"`

	// Name of the Network Interface (not unique or indexable)
	// Required: true
	Name *string `json:"name"`

	// (deprecated - replaced by networkSecurityGroupIDs) ID of the Network Security Group the network interface will be added to
	NetworkSecurityGroupID string `json:"networkSecurityGroupID,omitempty"`

	// Network security groups that the network interface is a member of.
	NetworkSecurityGroupIDs []string `json:"networkSecurityGroupIDs"`

	// The status of the network address group
	// Required: true
	Status *string `json:"status"`

	// The user tags associated with this resource.
	UserTags []string `json:"userTags,omitempty"`
}

// Validate validates this network interface
func (m *NetworkInterface) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateCrn(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateID(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateInstance(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateIPAddress(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateMacAddress(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateName(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateStatus(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *NetworkInterface) validateCrn(formats strfmt.Registry) error {

	if err := validate.Required("crn", "body", m.Crn); err != nil {
		return err
	}

	return nil
}

func (m *NetworkInterface) validateID(formats strfmt.Registry) error {

	if err := validate.Required("id", "body", m.ID); err != nil {
		return err
	}

	return nil
}

func (m *NetworkInterface) validateInstance(formats strfmt.Registry) error {
	if swag.IsZero(m.Instance) { // not required
		return nil
	}

	if m.Instance != nil {
		if err := m.Instance.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("instance")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("instance")
			}
			return err
		}
	}

	return nil
}

func (m *NetworkInterface) validateIPAddress(formats strfmt.Registry) error {

	if err := validate.Required("ipAddress", "body", m.IPAddress); err != nil {
		return err
	}

	return nil
}

func (m *NetworkInterface) validateMacAddress(formats strfmt.Registry) error {

	if err := validate.Required("macAddress", "body", m.MacAddress); err != nil {
		return err
	}

	return nil
}

func (m *NetworkInterface) validateName(formats strfmt.Registry) error {

	if err := validate.Required("name", "body", m.Name); err != nil {
		return err
	}

	return nil
}

func (m *NetworkInterface) validateStatus(formats strfmt.Registry) error {

	if err := validate.Required("status", "body", m.Status); err != nil {
		return err
	}

	return nil
}

// ContextValidate validate this network interface based on the context it is used
func (m *NetworkInterface) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	var res []error

	if err := m.contextValidateInstance(ctx, formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *NetworkInterface) contextValidateInstance(ctx context.Context, formats strfmt.Registry) error {

	if m.Instance != nil {

		if swag.IsZero(m.Instance) { // not required
			return nil
		}

		if err := m.Instance.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("instance")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("instance")
			}
			return err
		}
	}

	return nil
}

// MarshalBinary interface implementation
func (m *NetworkInterface) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *NetworkInterface) UnmarshalBinary(b []byte) error {
	var res NetworkInterface
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}

// NetworkInterfaceInstance The attached instance to this Network Interface
//
// swagger:model NetworkInterfaceInstance
type NetworkInterfaceInstance struct {

	// Link to instance resource
	Href string `json:"href,omitempty"`

	// The attached instance ID
	InstanceID string `json:"instanceID,omitempty"`
}

// Validate validates this network interface instance
func (m *NetworkInterfaceInstance) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this network interface instance based on context it is used
func (m *NetworkInterfaceInstance) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *NetworkInterfaceInstance) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *NetworkInterfaceInstance) UnmarshalBinary(b []byte) error {
	var res NetworkInterfaceInstance
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
