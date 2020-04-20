package gcp

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"gopkg.in/AlecAivazis/survey.v1"

	"github.com/openshift/installer/pkg/types/gcp"
	"github.com/openshift/installer/pkg/types/gcp/validation"
	"github.com/pkg/errors"
)

// Platform collects GCP-specific configuration.
func Platform() (*gcp.Platform, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	project, err := selectProject(ctx)
	if err != nil {
		return nil, err
	}

	region, err := selectRegion(project)

	return &gcp.Platform{
		ProjectID: project,
		Region:    region,
	}, nil
}

func selectProject(ctx context.Context) (string, error) {
	ssn, err := GetSession(ctx)
	if err != nil {
		return "", errors.Wrap(err, "failed to get session")
	}
	defaultProject := ssn.Credentials.ProjectID

	client := &Client{
		ssn: ssn,
	}

	var listOfCombo []string

	listOfNames, listOfIds, err := client.GetListOfProjects(ctx)
	if err != nil {
		return "", err
	}
	index := 0
	for i, v := range listOfIds {
		if v == defaultProject {
			index = i
		}
		listOfCombo = append(listOfCombo, listOfNames[i]+" ("+v+")")
	}

	defaultValue := listOfNames[index] + " (" + defaultProject + ")"

	var selectedProject string
	err = survey.Ask([]*survey.Question{
		{
			Prompt: &survey.Select{
				Message: "Project",
				Help:    "The GCP project to be used for installation.",
				Default: fmt.Sprintf("%s", defaultValue),
				Options: listOfCombo,
			},
			Validate: survey.ComposeValidators(survey.Required, func(ans interface{}) error {
				choice := strings.SplitN(ans.(string), " (", 2)[0]
				for _, v := range listOfNames {
					if v == choice {
						return nil
					}
				}
				return errors.Errorf("invalid project %q", choice)
			}),
		},
	}, &selectedProject)
	selectedProject = strings.SplitN(selectedProject, " (", 2)[1]
	selectedProject = selectedProject[:len(selectedProject)-1]
	return selectedProject, nil
}

func selectRegion(project string) (string, error) {
	longRegions := make([]string, 0, len(validation.Regions))
	shortRegions := make([]string, 0, len(validation.Regions))
	for id, location := range validation.Regions {
		longRegions = append(longRegions, fmt.Sprintf("%s (%s)", id, location))
		shortRegions = append(shortRegions, id)
	}
	regionTransform := survey.TransformString(func(s string) string {
		return strings.SplitN(s, " ", 2)[0]
	})

	sort.Strings(longRegions)
	sort.Strings(shortRegions)

	defaultRegion := "us-central1"
	var selectedRegion string
	err := survey.Ask([]*survey.Question{
		{
			Prompt: &survey.Select{
				Message: "Region",
				Help:    "The GCP region to be used for installation.",
				Default: fmt.Sprintf("%s (%s)", defaultRegion, validation.Regions[defaultRegion]),
				Options: longRegions,
			},
			Validate: survey.ComposeValidators(survey.Required, func(ans interface{}) error {
				choice := regionTransform(ans).(string)
				i := sort.SearchStrings(shortRegions, choice)
				if i == len(shortRegions) || shortRegions[i] != choice {
					return errors.Errorf("invalid region %q", choice)
				}
				return nil
			}),
			Transform: regionTransform,
		},
	}, &selectedRegion)
	if err != nil {
		return "", err
	}

	return selectedRegion, nil
}
