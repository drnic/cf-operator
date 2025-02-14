package manifest

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"code.cloudfoundry.org/cf-operator/pkg/kube/util/ctxlog"
)

type matcher func(*InstanceGroup, *AddOnPlacementRules) (bool, error)

// jobMatch matches stemcell rules for addon placement
func (m *Manifest) stemcellMatch(instanceGroup *InstanceGroup, rules *AddOnPlacementRules) (bool, error) {
	if instanceGroup == nil || rules == nil {
		return false, nil
	}

	osList := map[string]struct{}{}

	for _, job := range instanceGroup.Jobs {
		os, err := m.GetJobOS(instanceGroup.Name, job.Name)
		if err != nil {
			return false, errors.Wrapf(err, "failed to calculate OS for BOSH job %s in instanceGroup %s", job.Name, instanceGroup.Name)
		}

		osList[os] = struct{}{}
	}

	for _, s := range rules.Stemcell {
		if _, osPresent := osList[s.OS]; osPresent {
			return true, nil
		}
	}

	return false, nil
}

// jobMatch matches job rules for addon placement
func (m *Manifest) jobMatch(instanceGroup *InstanceGroup, rules *AddOnPlacementRules) (bool, error) {
	if instanceGroup == nil || rules == nil {
		return false, nil
	}

	jobList := map[string]struct{}{}

	for _, job := range instanceGroup.Jobs {
		// We keep a map with keys release:job, so we can quickly determine later if
		// a job exists or not
		jobList[fmt.Sprintf("%s:%s", job.Release, job.Name)] = struct{}{}
	}

	for _, job := range rules.Jobs {
		if _, jobPresent := jobList[fmt.Sprintf("%s:%s", job.Release, job.Name)]; jobPresent {
			return true, nil
		}
	}

	return false, nil
}

// instanceGroupMatch matches instance group rules for addon placement
func (m *Manifest) instanceGroupMatch(instanceGroup *InstanceGroup, rules *AddOnPlacementRules) (bool, error) {
	if instanceGroup == nil || rules == nil {
		return false, nil
	}

	for _, ig := range rules.InstanceGroup {
		if ig == instanceGroup.Name {
			return true, nil
		}
	}

	return false, nil
}

// addOnPlacementMatch returns true if any placement rule of the addon matches the instance group
func (m *Manifest) addOnPlacementMatch(ctx context.Context, placementType string, instanceGroup *InstanceGroup, rules *AddOnPlacementRules) (bool, error) {
	// This check is special, not a matcher. Lifecycle always needs to match
	if (instanceGroup.LifeCycle == IGTypeErrand ||
		instanceGroup.LifeCycle == IGTypeAutoErrand) &&
		(rules == nil || rules.Lifecycle != IGTypeErrand) {
		ctxlog.Debugf(ctx, "Instance group '%s' is an errand, but the %s placement rules don't match an errand", instanceGroup.Name, placementType)
		return false, nil
	}

	if (instanceGroup.LifeCycle == IGTypeService ||
		instanceGroup.LifeCycle == IGTypeDefault) &&
		(rules != nil && rules.Lifecycle != IGTypeDefault && rules.Lifecycle != IGTypeService) {
		ctxlog.Debugf(ctx, "Instance group '%s' is a BOSH service, but the %s placement rules don't match a BOSH service", instanceGroup.Name, placementType)
		return false, nil
	}

	matchers := []matcher{
		m.stemcellMatch,
		m.jobMatch,
		m.instanceGroupMatch,
	}

	matchResult := false

	for _, matcher := range matchers {
		matched, err := matcher(instanceGroup, rules)
		if err != nil {
			return false, errors.Wrapf(err, "failed to process match for instance group %s", instanceGroup.Name)
		}

		matchResult = matchResult || matched
	}

	if !matchResult {
		ctxlog.Debugf(ctx, "Instance group '%s' did not match the %s placement rules", instanceGroup.Name, placementType)
	}

	return matchResult, nil
}
