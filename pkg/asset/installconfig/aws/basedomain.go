package aws

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/core"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	route53types "github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/sirupsen/logrus"
)

// ErrNoCredentials is returned when no AWS credentials are available.
var ErrNoCredentials = errors.New("no AWS credentials found")

// StaticCredentials is an explicit AWS credential set supplied by a caller.
type StaticCredentials struct {
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
}

// GetBaseDomainOptions returns the public Route 53 hosted zone names using the
// non-interactive credential chain. If creds is non-nil, those credentials are
// used instead of the default chain.
func GetBaseDomainOptions(ctx context.Context, endpointOpts EndpointOptions, creds *StaticCredentials) ([]string, error) {
	opts := []func(*config.LoadOptions) error{config.WithRegion(endpointOpts.Region)}
	if creds != nil {
		opts = append(opts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(creds.AccessKeyID, creds.SecretAccessKey, creds.SessionToken),
		))
	}

	cfg, err := GetConfigNoPrompt(ctx, opts...)
	if err != nil {
		return nil, err
	}
	if _, err := cfg.Credentials.Retrieve(ctx); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNoCredentials, err)
	}

	client := route53.NewFromConfig(cfg, func(o *route53.Options) {
		o.EndpointResolverV2 = &Route53EndpointResolver{
			ServiceEndpointResolver: NewServiceEndpointResolver(endpointOpts),
		}
	})
	return ListPublicBaseDomains(ctx, client)
}

// ListPublicBaseDomains returns the account's public Route 53 hosted zone
// names. It is used by non-interactive callers such as the TUI.
func ListPublicBaseDomains(ctx context.Context, client *route53.Client) ([]string, error) {
	logrus.Debugf("listing AWS hosted zones")

	publicZoneMap := map[string]struct{}{}
	exists := struct{}{}

	paginator := route53.NewListHostedZonesPaginator(client, &route53.ListHostedZonesInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list hosted zones: %w", err)
		}

		for _, zone := range page.HostedZones {
			if zone.Config != nil && !zone.Config.PrivateZone {
				publicZoneMap[strings.TrimSuffix(aws.ToString(zone.Name), ".")] = exists
			}
		}
	}

	publicZones := make([]string, 0, len(publicZoneMap))
	for name := range publicZoneMap {
		publicZones = append(publicZones, name)
	}
	sort.Strings(publicZones)
	if len(publicZones) == 0 {
		return nil, errors.New("no public Route 53 hosted zones found")
	}
	return publicZones, nil
}

// GetBaseDomain returns a base domain chosen from among the account's
// public routes.
func GetBaseDomain(ctx context.Context, client *route53.Client) (string, error) {
	publicZones, err := ListPublicBaseDomains(ctx, client)
	if err != nil {
		return "", err
	}

	var domain string
	if err := survey.AskOne(
		&survey.Select{
			Message: "Base Domain",
			Help:    "The base domain of the cluster. All DNS records will be sub-domains of this base and will also include the cluster name.\n\nIf you don't see you intended base-domain listed, create a new public Route53 hosted zone and rerun the installer.",
			Options: publicZones,
		},
		&domain,
		survey.WithValidator(func(ans interface{}) error {
			choice := ans.(core.OptionAnswer).Value
			i := sort.SearchStrings(publicZones, choice)
			if i == len(publicZones) || publicZones[i] != choice {
				return fmt.Errorf("invalid base domain %q", choice)
			}
			return nil
		}),
	); err != nil {
		return "", fmt.Errorf("failed UserInput: %w", err)
	}

	return domain, nil
}

// GetPublicZone returns a public route53 zone that matches the name.
func GetPublicZone(ctx context.Context, client *route53.Client, name string) (*route53types.HostedZone, error) {
	paginator := route53.NewListHostedZonesPaginator(client, &route53.ListHostedZonesInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list hosted zones: %w", err)
		}

		for _, zone := range page.HostedZones {
			if zone.Config != nil && !zone.Config.PrivateZone && strings.TrimSuffix(aws.ToString(zone.Name), ".") == strings.TrimSuffix(name, ".") {
				return &zone, nil
			}
		}
	}

	return nil, fmt.Errorf("no public route53 zone found matching name %q", name)
}
