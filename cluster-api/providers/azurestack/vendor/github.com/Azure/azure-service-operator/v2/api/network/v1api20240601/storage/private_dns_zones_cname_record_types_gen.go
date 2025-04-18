// Code generated by azure-service-operator-codegen. DO NOT EDIT.
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.
package storage

import (
	"github.com/Azure/azure-service-operator/v2/pkg/genruntime"
	"github.com/Azure/azure-service-operator/v2/pkg/genruntime/conditions"
	"github.com/Azure/azure-service-operator/v2/pkg/genruntime/configmaps"
	"github.com/Azure/azure-service-operator/v2/pkg/genruntime/core"
	"github.com/Azure/azure-service-operator/v2/pkg/genruntime/secrets"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// +kubebuilder:rbac:groups=network.azure.com,resources=privatednszonescnamerecords,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=network.azure.com,resources={privatednszonescnamerecords/status,privatednszonescnamerecords/finalizers},verbs=get;update;patch

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="Severity",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].severity"
// +kubebuilder:printcolumn:name="Reason",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].reason"
// +kubebuilder:printcolumn:name="Message",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].message"
// Storage version of v1api20240601.PrivateDnsZonesCNAMERecord
// Generator information:
// - Generated from: /privatedns/resource-manager/Microsoft.Network/stable/2024-06-01/privatedns.json
// - ARM URI: /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/privateDnsZones/{privateZoneName}/CNAME/{relativeRecordSetName}
type PrivateDnsZonesCNAMERecord struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              PrivateDnsZonesCNAMERecord_Spec   `json:"spec,omitempty"`
	Status            PrivateDnsZonesCNAMERecord_STATUS `json:"status,omitempty"`
}

var _ conditions.Conditioner = &PrivateDnsZonesCNAMERecord{}

// GetConditions returns the conditions of the resource
func (record *PrivateDnsZonesCNAMERecord) GetConditions() conditions.Conditions {
	return record.Status.Conditions
}

// SetConditions sets the conditions on the resource status
func (record *PrivateDnsZonesCNAMERecord) SetConditions(conditions conditions.Conditions) {
	record.Status.Conditions = conditions
}

var _ configmaps.Exporter = &PrivateDnsZonesCNAMERecord{}

// ConfigMapDestinationExpressions returns the Spec.OperatorSpec.ConfigMapExpressions property
func (record *PrivateDnsZonesCNAMERecord) ConfigMapDestinationExpressions() []*core.DestinationExpression {
	if record.Spec.OperatorSpec == nil {
		return nil
	}
	return record.Spec.OperatorSpec.ConfigMapExpressions
}

var _ secrets.Exporter = &PrivateDnsZonesCNAMERecord{}

// SecretDestinationExpressions returns the Spec.OperatorSpec.SecretExpressions property
func (record *PrivateDnsZonesCNAMERecord) SecretDestinationExpressions() []*core.DestinationExpression {
	if record.Spec.OperatorSpec == nil {
		return nil
	}
	return record.Spec.OperatorSpec.SecretExpressions
}

var _ genruntime.KubernetesResource = &PrivateDnsZonesCNAMERecord{}

// AzureName returns the Azure name of the resource
func (record *PrivateDnsZonesCNAMERecord) AzureName() string {
	return record.Spec.AzureName
}

// GetAPIVersion returns the ARM API version of the resource. This is always "2024-06-01"
func (record PrivateDnsZonesCNAMERecord) GetAPIVersion() string {
	return "2024-06-01"
}

// GetResourceScope returns the scope of the resource
func (record *PrivateDnsZonesCNAMERecord) GetResourceScope() genruntime.ResourceScope {
	return genruntime.ResourceScopeResourceGroup
}

// GetSpec returns the specification of this resource
func (record *PrivateDnsZonesCNAMERecord) GetSpec() genruntime.ConvertibleSpec {
	return &record.Spec
}

// GetStatus returns the status of this resource
func (record *PrivateDnsZonesCNAMERecord) GetStatus() genruntime.ConvertibleStatus {
	return &record.Status
}

// GetSupportedOperations returns the operations supported by the resource
func (record *PrivateDnsZonesCNAMERecord) GetSupportedOperations() []genruntime.ResourceOperation {
	return []genruntime.ResourceOperation{
		genruntime.ResourceOperationDelete,
		genruntime.ResourceOperationGet,
		genruntime.ResourceOperationPut,
	}
}

// GetType returns the ARM Type of the resource. This is always "Microsoft.Network/privateDnsZones/CNAME"
func (record *PrivateDnsZonesCNAMERecord) GetType() string {
	return "Microsoft.Network/privateDnsZones/CNAME"
}

// NewEmptyStatus returns a new empty (blank) status
func (record *PrivateDnsZonesCNAMERecord) NewEmptyStatus() genruntime.ConvertibleStatus {
	return &PrivateDnsZonesCNAMERecord_STATUS{}
}

// Owner returns the ResourceReference of the owner
func (record *PrivateDnsZonesCNAMERecord) Owner() *genruntime.ResourceReference {
	group, kind := genruntime.LookupOwnerGroupKind(record.Spec)
	return record.Spec.Owner.AsResourceReference(group, kind)
}

// SetStatus sets the status of this resource
func (record *PrivateDnsZonesCNAMERecord) SetStatus(status genruntime.ConvertibleStatus) error {
	// If we have exactly the right type of status, assign it
	if st, ok := status.(*PrivateDnsZonesCNAMERecord_STATUS); ok {
		record.Status = *st
		return nil
	}

	// Convert status to required version
	var st PrivateDnsZonesCNAMERecord_STATUS
	err := status.ConvertStatusTo(&st)
	if err != nil {
		return errors.Wrap(err, "failed to convert status")
	}

	record.Status = st
	return nil
}

// Hub marks that this PrivateDnsZonesCNAMERecord is the hub type for conversion
func (record *PrivateDnsZonesCNAMERecord) Hub() {}

// OriginalGVK returns a GroupValueKind for the original API version used to create the resource
func (record *PrivateDnsZonesCNAMERecord) OriginalGVK() *schema.GroupVersionKind {
	return &schema.GroupVersionKind{
		Group:   GroupVersion.Group,
		Version: record.Spec.OriginalVersion,
		Kind:    "PrivateDnsZonesCNAMERecord",
	}
}

// +kubebuilder:object:root=true
// Storage version of v1api20240601.PrivateDnsZonesCNAMERecord
// Generator information:
// - Generated from: /privatedns/resource-manager/Microsoft.Network/stable/2024-06-01/privatedns.json
// - ARM URI: /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/privateDnsZones/{privateZoneName}/CNAME/{relativeRecordSetName}
type PrivateDnsZonesCNAMERecordList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PrivateDnsZonesCNAMERecord `json:"items"`
}

// Storage version of v1api20240601.PrivateDnsZonesCNAMERecord_Spec
type PrivateDnsZonesCNAMERecord_Spec struct {
	ARecords    []ARecord    `json:"aRecords,omitempty"`
	AaaaRecords []AaaaRecord `json:"aaaaRecords,omitempty"`

	// AzureName: The name of the resource in Azure. This is often the same as the name of the resource in Kubernetes but it
	// doesn't have to be.
	AzureName       string                                  `json:"azureName,omitempty"`
	CnameRecord     *CnameRecord                            `json:"cnameRecord,omitempty"`
	Etag            *string                                 `json:"etag,omitempty"`
	Metadata        map[string]string                       `json:"metadata,omitempty"`
	MxRecords       []MxRecord                              `json:"mxRecords,omitempty"`
	OperatorSpec    *PrivateDnsZonesCNAMERecordOperatorSpec `json:"operatorSpec,omitempty"`
	OriginalVersion string                                  `json:"originalVersion,omitempty"`

	// +kubebuilder:validation:Required
	// Owner: The owner of the resource. The owner controls where the resource goes when it is deployed. The owner also
	// controls the resources lifecycle. When the owner is deleted the resource will also be deleted. Owner is expected to be a
	// reference to a network.azure.com/PrivateDnsZone resource
	Owner       *genruntime.KnownResourceReference `group:"network.azure.com" json:"owner,omitempty" kind:"PrivateDnsZone"`
	PropertyBag genruntime.PropertyBag             `json:"$propertyBag,omitempty"`
	PtrRecords  []PtrRecord                        `json:"ptrRecords,omitempty"`
	SoaRecord   *SoaRecord                         `json:"soaRecord,omitempty"`
	SrvRecords  []SrvRecord                        `json:"srvRecords,omitempty"`
	Ttl         *int                               `json:"ttl,omitempty"`
	TxtRecords  []TxtRecord                        `json:"txtRecords,omitempty"`
}

var _ genruntime.ConvertibleSpec = &PrivateDnsZonesCNAMERecord_Spec{}

// ConvertSpecFrom populates our PrivateDnsZonesCNAMERecord_Spec from the provided source
func (record *PrivateDnsZonesCNAMERecord_Spec) ConvertSpecFrom(source genruntime.ConvertibleSpec) error {
	if source == record {
		return errors.New("attempted conversion between unrelated implementations of github.com/Azure/azure-service-operator/v2/pkg/genruntime/ConvertibleSpec")
	}

	return source.ConvertSpecTo(record)
}

// ConvertSpecTo populates the provided destination from our PrivateDnsZonesCNAMERecord_Spec
func (record *PrivateDnsZonesCNAMERecord_Spec) ConvertSpecTo(destination genruntime.ConvertibleSpec) error {
	if destination == record {
		return errors.New("attempted conversion between unrelated implementations of github.com/Azure/azure-service-operator/v2/pkg/genruntime/ConvertibleSpec")
	}

	return destination.ConvertSpecFrom(record)
}

// Storage version of v1api20240601.PrivateDnsZonesCNAMERecord_STATUS
type PrivateDnsZonesCNAMERecord_STATUS struct {
	ARecords         []ARecord_STATUS       `json:"aRecords,omitempty"`
	AaaaRecords      []AaaaRecord_STATUS    `json:"aaaaRecords,omitempty"`
	CnameRecord      *CnameRecord_STATUS    `json:"cnameRecord,omitempty"`
	Conditions       []conditions.Condition `json:"conditions,omitempty"`
	Etag             *string                `json:"etag,omitempty"`
	Fqdn             *string                `json:"fqdn,omitempty"`
	Id               *string                `json:"id,omitempty"`
	IsAutoRegistered *bool                  `json:"isAutoRegistered,omitempty"`
	Metadata         map[string]string      `json:"metadata,omitempty"`
	MxRecords        []MxRecord_STATUS      `json:"mxRecords,omitempty"`
	Name             *string                `json:"name,omitempty"`
	PropertyBag      genruntime.PropertyBag `json:"$propertyBag,omitempty"`
	PtrRecords       []PtrRecord_STATUS     `json:"ptrRecords,omitempty"`
	SoaRecord        *SoaRecord_STATUS      `json:"soaRecord,omitempty"`
	SrvRecords       []SrvRecord_STATUS     `json:"srvRecords,omitempty"`
	Ttl              *int                   `json:"ttl,omitempty"`
	TxtRecords       []TxtRecord_STATUS     `json:"txtRecords,omitempty"`
	Type             *string                `json:"type,omitempty"`
}

var _ genruntime.ConvertibleStatus = &PrivateDnsZonesCNAMERecord_STATUS{}

// ConvertStatusFrom populates our PrivateDnsZonesCNAMERecord_STATUS from the provided source
func (record *PrivateDnsZonesCNAMERecord_STATUS) ConvertStatusFrom(source genruntime.ConvertibleStatus) error {
	if source == record {
		return errors.New("attempted conversion between unrelated implementations of github.com/Azure/azure-service-operator/v2/pkg/genruntime/ConvertibleStatus")
	}

	return source.ConvertStatusTo(record)
}

// ConvertStatusTo populates the provided destination from our PrivateDnsZonesCNAMERecord_STATUS
func (record *PrivateDnsZonesCNAMERecord_STATUS) ConvertStatusTo(destination genruntime.ConvertibleStatus) error {
	if destination == record {
		return errors.New("attempted conversion between unrelated implementations of github.com/Azure/azure-service-operator/v2/pkg/genruntime/ConvertibleStatus")
	}

	return destination.ConvertStatusFrom(record)
}

// Storage version of v1api20240601.PrivateDnsZonesCNAMERecordOperatorSpec
// Details for configuring operator behavior. Fields in this struct are interpreted by the operator directly rather than being passed to Azure
type PrivateDnsZonesCNAMERecordOperatorSpec struct {
	ConfigMapExpressions []*core.DestinationExpression `json:"configMapExpressions,omitempty"`
	PropertyBag          genruntime.PropertyBag        `json:"$propertyBag,omitempty"`
	SecretExpressions    []*core.DestinationExpression `json:"secretExpressions,omitempty"`
}

func init() {
	SchemeBuilder.Register(&PrivateDnsZonesCNAMERecord{}, &PrivateDnsZonesCNAMERecordList{})
}
