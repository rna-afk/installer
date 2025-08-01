package gcp

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	kms "cloud.google.com/go/kms/apiv1"
	"cloud.google.com/go/kms/apiv1/kmspb"
	"github.com/pkg/errors"
	googleoauth "golang.org/x/oauth2/google"
	"google.golang.org/api/cloudresourcemanager/v3"
	compute "google.golang.org/api/compute/v1"
	dns "google.golang.org/api/dns/v1"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	serviceusage "google.golang.org/api/serviceusage/v1beta1"
	"k8s.io/apimachinery/pkg/util/sets"

	configv1 "github.com/openshift/api/config/v1"
	gcpconsts "github.com/openshift/installer/pkg/constants/gcp"
	gcptypes "github.com/openshift/installer/pkg/types/gcp"
)

//go:generate mockgen -source=./client.go -destination=./mock/gcpclient_generated.go -package=mock

const defaultTimeout = 2 * time.Minute

var (
	// RequiredBasePermissions is the list of permissions required for an installation.
	// A list of valid permissions can be found at https://cloud.google.com/iam/docs/understanding-roles.
	RequiredBasePermissions = []string{}
)

// API represents the calls made to the API.
type API interface {
	GetNetwork(ctx context.Context, network, project string) (*compute.Network, error)
	GetMachineType(ctx context.Context, project, zone, machineType string) (*compute.MachineType, error)
	GetMachineTypeWithZones(ctx context.Context, project, region, machineType string) (*compute.MachineType, sets.Set[string], error)
	GetPublicDomains(ctx context.Context, project string) ([]string, error)
	GetDNSZone(ctx context.Context, project, baseDomain string, isPublic bool) (*dns.ManagedZone, error)
	GetDNSZoneByName(ctx context.Context, project, zoneName string) (*dns.ManagedZone, error)
	GetSubnetworks(ctx context.Context, network, project, region string) ([]*compute.Subnetwork, error)
	GetProjects(ctx context.Context) (map[string]string, error)
	GetRegions(ctx context.Context, project string) ([]string, error)
	GetRecordSets(ctx context.Context, project, zone string) ([]*dns.ResourceRecordSet, error)
	GetZones(ctx context.Context, project, filter string) ([]*compute.Zone, error)
	GetEnabledServices(ctx context.Context, project string) ([]string, error)
	GetServiceAccount(ctx context.Context, project, serviceAccount string) (string, error)
	GetCredentials() *googleoauth.Credentials
	GetImage(ctx context.Context, name string, project string) (*compute.Image, error)
	GetProjectPermissions(ctx context.Context, project string, permissions []string) (sets.Set[string], error)
	GetProjectByID(ctx context.Context, project string) (*cloudresourcemanager.Project, error)
	ValidateServiceAccountHasPermissions(ctx context.Context, project string, permissions []string) (bool, error)
	GetProjectTags(ctx context.Context, projectID string) (sets.Set[string], error)
	GetNamespacedTagValue(ctx context.Context, tagNamespacedName string) (*cloudresourcemanager.TagValue, error)
	GetKeyRing(ctx context.Context, kmsKeyRef *gcptypes.KMSKeyReference) (*kmspb.KeyRing, error)
}

// Client makes calls to the GCP API.
type Client struct {
	ssn       *Session
	endpoints []configv1.GCPServiceEndpoint
}

// NewClient initializes a client with a session.
func NewClient(ctx context.Context, endpoints []configv1.GCPServiceEndpoint) (*Client, error) {
	ssn, err := GetSession(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get session")
	}

	modifiedEndpoints := FormatGCPEndpointList(endpoints, FormatGCPEndpointInput{SkipPath: false})

	client := &Client{
		ssn:       ssn,
		endpoints: modifiedEndpoints,
	}
	return client, nil
}

func (c *Client) getComputeService(ctx context.Context) (*compute.Service, error) {
	svc, err := GetComputeService(ctx, c.endpoints)
	if err != nil {
		return nil, fmt.Errorf("client failed to create compute service: %w", err)
	}
	return svc, nil
}

func (c *Client) getDNSService(ctx context.Context) (*dns.Service, error) {
	svc, err := GetDNSService(ctx, c.endpoints)
	if err != nil {
		return nil, fmt.Errorf("client failed to create dns service: %w", err)
	}
	return svc, nil
}

func (c *Client) getCloudResourceService(ctx context.Context) (*cloudresourcemanager.Service, error) {
	svc, err := GetCloudResourceService(ctx, c.endpoints)
	if err != nil {
		return nil, fmt.Errorf("client failed to create cloud resource service: %w", err)
	}
	return svc, nil
}

func (c *Client) getServiceUsageService(ctx context.Context) (*serviceusage.APIService, error) {
	svc, err := GetServiceUsageService(ctx, c.endpoints)
	if err != nil {
		return nil, fmt.Errorf("client failed to create service usage service: %w", err)
	}
	return svc, nil
}

// GetMachineType uses the GCP Compute Service API to get the specified machine type.
func (c *Client) GetMachineType(ctx context.Context, project, zone, machineType string) (*compute.MachineType, error) {
	svc, err := c.getComputeService(ctx)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()
	req, err := svc.MachineTypes.Get(project, zone, machineType).Context(ctx).Do()
	if err != nil {
		return nil, err
	}

	return req, nil
}

// GetMachineTypeList retrieves the machine type with the specified fields.
func GetMachineTypeList(ctx context.Context, svc *compute.Service, project, region, machineType, fields string) ([]*compute.MachineType, error) {
	var machines []*compute.MachineType

	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	filter := fmt.Sprintf("name = \"%s\" AND zone : %s-*", machineType, region)
	req := svc.MachineTypes.AggregatedList(project).Filter(filter).Context(ctx)
	if len(fields) > 0 {
		req.Fields(googleapi.Field(fields))
	}

	err := req.Pages(ctx, func(page *compute.MachineTypeAggregatedList) error {
		for _, scopedList := range page.Items {
			machines = append(machines, scopedList.MachineTypes...)
		}
		return nil
	})

	return machines, err
}

// GetMachineTypeWithZones retrieves the specified machine type and the zones in which it is available.
func (c *Client) GetMachineTypeWithZones(ctx context.Context, project, region, machineType string) (*compute.MachineType, sets.Set[string], error) {
	svc, err := c.getComputeService(ctx)
	if err != nil {
		return nil, nil, err
	}

	pz, err := GetZones(ctx, svc, project, region)
	if err != nil {
		return nil, nil, err
	}
	projZones := sets.New[string]()
	for _, zone := range pz {
		projZones.Insert(zone.Name)
	}

	machines, err := GetMachineTypeList(ctx, svc, project, region, machineType, "")
	if err != nil {
		return nil, nil, err
	}

	// Custom machine types are not included in aggregated lists, so let's try
	// to get the machine type directly before returning an error. Also
	// fallback to all the zones in the project
	if len(machines) == 0 {
		cctx, cancel := context.WithTimeout(ctx, defaultTimeout)
		defer cancel()

		if len(pz) == 0 {
			return nil, nil, fmt.Errorf("failed to find public zone in project %s region %s", project, region)
		}
		machine, err := svc.MachineTypes.Get(project, pz[0].Name, machineType).Context(cctx).Do()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to fetch instance type: %w", err)
		}
		return machine, projZones, nil
	}

	zones := sets.New[string]()
	for _, machine := range machines {
		zones.Insert(machine.Zone)
	}
	// Restrict to zones avaialable in the project
	zones = zones.Intersection(projZones)

	return machines[0], zones, nil
}

// GetNetwork uses the GCP Compute Service API to get a network by name from a project.
func (c *Client) GetNetwork(ctx context.Context, network, project string) (*compute.Network, error) {
	svc, err := c.getComputeService(ctx)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()
	res, err := svc.Networks.Get(project, network).Context(ctx).Do()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get network %s", network)
	}
	return res, nil
}

// GetPublicDomains returns all of the domains from among the project's public DNS zones.
func (c *Client) GetPublicDomains(ctx context.Context, project string) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	svc, err := c.getDNSService(ctx)
	if err != nil {
		return []string{}, err
	}

	var publicZones []string
	req := svc.ManagedZones.List(project).Context(ctx)
	if err := req.Pages(ctx, func(page *dns.ManagedZonesListResponse) error {
		for _, v := range page.ManagedZones {
			if v.Visibility != "private" {
				publicZones = append(publicZones, strings.TrimSuffix(v.DnsName, "."))
			}
		}
		return nil
	}); err != nil {
		return publicZones, err
	}
	return publicZones, nil
}

// GetDNSZoneByName returns a DNS zone matching the `zoneName` if the DNS zone exists
// and can be seen (correct permissions for a private zone) in the project.
func (c *Client) GetDNSZoneByName(ctx context.Context, project, zoneName string) (*dns.ManagedZone, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	svc, err := c.getDNSService(ctx)
	if err != nil {
		return nil, err
	}
	returnedZone, err := svc.ManagedZones.Get(project, zoneName).Context(ctx).Do()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get DNS Zones")
	}
	return returnedZone, nil
}

// GetDNSZone returns a DNS zone for a basedomain.
func (c *Client) GetDNSZone(ctx context.Context, project, baseDomain string, isPublic bool) (*dns.ManagedZone, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	svc, err := c.getDNSService(ctx)
	if err != nil {
		return nil, err
	}
	if !strings.HasSuffix(baseDomain, ".") {
		baseDomain = fmt.Sprintf("%s.", baseDomain)
	}

	// currently, only private and public are supported. All peering zones are private.
	visibility := "private"
	if isPublic {
		visibility = "public"
	}

	req := svc.ManagedZones.List(project).DnsName(baseDomain).Context(ctx)
	var res *dns.ManagedZone
	if err := req.Pages(ctx, func(page *dns.ManagedZonesListResponse) error {
		for idx, v := range page.ManagedZones {
			// Peering zones are not allowed during the installation process.
			if v.Visibility == visibility && v.PeeringConfig == nil {
				res = page.ManagedZones[idx]
			}
		}
		return nil
	}); err != nil {
		return nil, errors.Wrap(err, "failed to list DNS Zones")
	}
	if res == nil {
		if isPublic {
			return nil, &googleapi.Error{
				Code:    http.StatusNotFound,
				Message: "no matching public DNS Zone found",
			}
		}
		// A Private DNS Zone may be created (if the correct permissions exist)
		return nil, nil
	}
	return res, nil
}

// GetRecordSets returns all the records for a DNS zone.
func (c *Client) GetRecordSets(ctx context.Context, project, zone string) ([]*dns.ResourceRecordSet, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	svc, err := c.getDNSService(ctx)
	if err != nil {
		return nil, err
	}

	req := svc.ResourceRecordSets.List(project, zone).Context(ctx)
	var rrSets []*dns.ResourceRecordSet
	if err := req.Pages(ctx, func(page *dns.ResourceRecordSetsListResponse) error {
		rrSets = append(rrSets, page.Rrsets...)
		return nil
	}); err != nil {
		return nil, err
	}
	return rrSets, nil
}

// GetSubnetworks uses the GCP Compute Service API to retrieve all subnetworks in a given network.
func (c *Client) GetSubnetworks(ctx context.Context, network, project, region string) ([]*compute.Subnetwork, error) {
	svc, err := c.getComputeService(ctx)
	if err != nil {
		return nil, err
	}

	filter := fmt.Sprintf("network eq .*%s", network)
	req := svc.Subnetworks.List(project, region).Filter(filter)
	var res []*compute.Subnetwork

	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	if err := req.Pages(ctx, func(page *compute.SubnetworkList) error {
		res = append(res, page.Items...)
		return nil
	}); err != nil {
		return nil, err
	}
	return res, nil
}

// GetProjects gets the list of project names and ids associated with the current user in the form
// of a map whose keys are ids and values are names.
func (c *Client) GetProjects(ctx context.Context) (map[string]string, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	// Currently, this is only called during the survey. The survey does not require that
	// custom endpoints are applied to the client. Pass an empty set of endpoints.
	svc, err := c.getCloudResourceService(ctx)
	if err != nil {
		return nil, err
	}

	req := svc.Projects.Search()
	projects := make(map[string]string)
	if err := req.Pages(ctx, func(page *cloudresourcemanager.SearchProjectsResponse) error {
		for _, project := range page.Projects {
			projects[project.ProjectId] = project.Name
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return projects, nil
}

// GetProjectByID retrieves the project specified by its ID.
func (c *Client) GetProjectByID(ctx context.Context, project string) (*cloudresourcemanager.Project, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	svc, err := c.getCloudResourceService(ctx)
	if err != nil {
		return nil, err
	}

	return svc.Projects.Get(fmt.Sprintf(gcpconsts.ProjectNameFmt, project)).Context(ctx).Do()
}

// GetRegions gets the regions that are valid for the project. An error is returned when unsuccessful
func (c *Client) GetRegions(ctx context.Context, project string) ([]string, error) {
	svc, err := c.getComputeService(ctx)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()
	gcpRegionsList, err := svc.Regions.List(project).Context(ctx).Do()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get regions for project")
	}

	computeRegions := make([]string, 0, len(gcpRegionsList.Items))
	for _, region := range gcpRegionsList.Items {
		computeRegions = append(computeRegions, region.Name)
	}

	return computeRegions, nil
}

// GetZones uses the GCP Compute Service API to get a list of zones with UP status in a region from a project.
func GetZones(ctx context.Context, svc *compute.Service, project, region string) ([]*compute.Zone, error) {
	req := svc.Zones.List(project)
	zones := []*compute.Zone{}
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()
	if err := req.Pages(ctx, func(page *compute.ZoneList) error {
		for _, zone := range page.Items {
			if strings.HasSuffix(zone.Region, region) && strings.EqualFold(zone.Status, "UP") {
				zones = append(zones, zone)
			}
		}
		return nil
	}); err != nil {
		return nil, errors.Wrapf(err, "failed to get zones from project %s", project)
	}
	return zones, nil
}

// GetZones uses the GCP Compute Service API to get a list of zones from a project.
func (c *Client) GetZones(ctx context.Context, project, region string) ([]*compute.Zone, error) {
	svc, err := c.getComputeService(ctx)
	if err != nil {
		return nil, err
	}

	return GetZones(ctx, svc, project, region)
}

// GetEnabledServices gets the list of enabled services for a project.
func (c *Client) GetEnabledServices(ctx context.Context, project string) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	svc, err := c.getServiceUsageService(ctx)
	if err != nil {
		return nil, err
	}

	// List accepts a parent, which includes the type of resource with the id.
	parent := fmt.Sprintf("projects/%s", project)
	req := svc.Services.List(parent).Filter("state:ENABLED")
	var services []string
	if err := req.Pages(ctx, func(page *serviceusage.ListServicesResponse) error {
		for _, service := range page.Services {
			//services are listed in the form of project/services/serviceName
			index := strings.LastIndex(service.Name, "/")
			services = append(services, service.Name[index+1:])
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return services, nil
}

// GetServiceAccount retrieves a service account from a project if it exists.
func (c *Client) GetServiceAccount(ctx context.Context, project, serviceAccount string) (string, error) {
	svc, err := GetIAMService(ctx, c.endpoints)
	if err != nil {
		return "", errors.Wrapf(err, "failed create IAM service")
	}

	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	fullServiceAccountPath := fmt.Sprintf("projects/%s/serviceAccounts/%s", project, serviceAccount)
	rsp, err := svc.Projects.ServiceAccounts.Get(fullServiceAccountPath).Context(ctx).Do()
	if err != nil {
		return "", errors.Wrapf(err, "failed to find resource %s", fullServiceAccountPath)
	}
	return rsp.Name, nil
}

// GetCredentials returns the credentials used to authenticate the GCP session.
func (c *Client) GetCredentials() *googleoauth.Credentials {
	return c.ssn.Credentials
}

// GetImage returns the marketplace image specified by the user.
func (c *Client) GetImage(ctx context.Context, name string, project string) (*compute.Image, error) {
	svc, err := c.getComputeService(ctx)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	return svc.Images.Get(project, name).Context(ctx).Do()
}

func (c *Client) getPermissions(ctx context.Context, project string, permissions []string) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	service, err := c.getCloudResourceService(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get cloud resource manager service")
	}

	projectsService := cloudresourcemanager.NewProjectsService(service)
	rb := &cloudresourcemanager.TestIamPermissionsRequest{Permissions: permissions}
	response, err := projectsService.TestIamPermissions(fmt.Sprintf(gcpconsts.ProjectNameFmt, project), rb).Context(ctx).Do()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get Iam permissions")
	}

	return response.Permissions, nil
}

// GetProjectPermissions consumes a set of permissions and returns the set of found permissions for the service
// account (in the provided project). A list of valid permissions can be found at
// https://cloud.google.com/iam/docs/understanding-roles.
func (c *Client) GetProjectPermissions(ctx context.Context, project string, permissions []string) (sets.Set[string], error) {
	validPermissions, err := c.getPermissions(ctx, project, permissions)
	if err != nil {
		return nil, err
	}
	return sets.New[string](validPermissions...), nil
}

// ValidateServiceAccountHasPermissions compares the permissions to the set returned from the GCP API. Returns true
// if all permissions are available to the service account in the project.
func (c *Client) ValidateServiceAccountHasPermissions(ctx context.Context, project string, permissions []string) (bool, error) {
	validPermissions, err := c.GetProjectPermissions(ctx, project, permissions)
	if err != nil {
		return false, err
	}
	return validPermissions.Len() == len(permissions), nil
}

// GetProjectTags returns the list of effective tags attached to the provided project resource.
func (c *Client) GetProjectTags(ctx context.Context, projectID string) (sets.Set[string], error) {
	service, err := c.getCloudResourceService(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create cloud resource service: %w", err)
	}

	effectiveTags := sets.New[string]()
	effectiveTagsService := cloudresourcemanager.NewEffectiveTagsService(service)
	effectiveTagsRequest := effectiveTagsService.List().
		Context(ctx).
		Parent(fmt.Sprintf(gcpconsts.ProjectParentPathFmt, projectID))

	if err := effectiveTagsRequest.Pages(ctx, func(page *cloudresourcemanager.ListEffectiveTagsResponse) error {
		for _, effectiveTag := range page.EffectiveTags {
			effectiveTags.Insert(effectiveTag.NamespacedTagValue)
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("failed to fetch tags attached to %s project: %w", projectID, err)
	}

	return effectiveTags, nil
}

// GetNamespacedTagValue returns the Tag Value metadata fetched using the tag's NamespacedName.
func (c *Client) GetNamespacedTagValue(ctx context.Context, tagNamespacedName string) (*cloudresourcemanager.TagValue, error) {
	service, err := c.getCloudResourceService(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create cloud resource service: %w", err)
	}

	tagValuesService := cloudresourcemanager.NewTagValuesService(service)

	tagValue, err := tagValuesService.GetNamespaced().
		Context(ctx).
		Name(tagNamespacedName).
		Do()

	if err != nil {
		return nil, fmt.Errorf("failed to fetch %s tag value: %w", tagNamespacedName, err)
	}

	return tagValue, nil
}

func (c *Client) getKeyManagementClient(ctx context.Context) (*kms.KeyManagementClient, error) {
	kmsClient, err := kms.NewKeyManagementClient(ctx, option.WithCredentials(c.ssn.Credentials))
	if err != nil {
		return nil, fmt.Errorf("failed to create kms key management client: %w", err)
	}
	return kmsClient, nil
}

// GetKeyRing returns the key ring associated with the key name (if found).
func (c *Client) GetKeyRing(ctx context.Context, kmsKeyRef *gcptypes.KMSKeyReference) (*kmspb.KeyRing, error) {
	kmsClient, err := c.getKeyManagementClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("key ring client creation failed: %w", err)
	}

	keyRingName := fmt.Sprintf("projects/%s/locations/%s/keyRings/%s", kmsKeyRef.ProjectID, kmsKeyRef.Location, kmsKeyRef.KeyRing)
	listReq := &kmspb.ListKeyRingsRequest{
		Parent: fmt.Sprintf("projects/%s/locations/%s", kmsKeyRef.ProjectID, kmsKeyRef.Location),
	}

	// OCPBUGS-52203:  GetKeyRingRequest{Name: keyRingName} should work but the resource name (above) is not found.
	// The cloudkms.keyRings.list permission is required for this operation.
	listItr := kmsClient.ListKeyRings(ctx, listReq)
	for {
		resp, err := listItr.Next()
		if errors.Is(err, iterator.Done) {
			break
		} else if err != nil {
			return nil, fmt.Errorf("failed to iterate through list of kms keyrings: %w", err)
		}

		re := resp
		if re.Name == keyRingName {
			return re, nil
		}
	}
	return nil, fmt.Errorf("failed to find kms key ring with name %s", keyRingName)
}
