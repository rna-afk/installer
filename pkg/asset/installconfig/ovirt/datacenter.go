package ovirt

import (
	"github.com/openshift/installer/pkg/stdlogger"
	ovirtsdk4 "github.com/ovirt/go-ovirt"

	"github.com/pkg/errors"
)

// datacentersAvailable looks for all datacenters available in the system based on searchFilter.
// If search filter not provided, the default filter will be "status=up"
// Returns type: *ovirtsdk.DataCentersServiceListResponse
func datacentersAvailable(conn *ovirtsdk4.Connection, searchFilter string) (*ovirtsdk4.DataCentersServiceListResponse, error) {
	if searchFilter == "" {
		searchFilter = "status=up"
	}
	dcService := conn.SystemService().DataCentersService()

	stdlogger.Debugf("searching for DataCenters with search filter: %s", searchFilter)
	dcResp, err := dcService.List().Search(searchFilter).Send()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to search available DataCenters")
	}

	return dcResp, nil
}
