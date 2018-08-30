// Licensed Materials - Property of IBM
// (C) Copyright IBM Corp. 2018. All Rights Reserved.
// US Government Users Restricted Rights - Use, duplication or
// disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
// Copyright 2018 IBM Corporation

package reconcile

import (
	"fmt"
	"reflect"

	multierror "github.com/hashicorp/go-multierror"

	istiomodel "istio.io/istio/pilot/pkg/model"
	"istio.io/istio/pkg/log"

	"github.ibm.com/istio-research/multicluster-roadmap/multicluster/pkg/model"
)

// AddMulticlusterConfig takes an Istio config store and a new RemoteServiceBinding or ServiceExpositionPolicy
// and returns the new and modified Istio configurations needed to implement the desired multicluster config.
func AddMulticlusterConfig(store istiomodel.ConfigStore, newconfig istiomodel.Config, ci model.ClusterInfo) ([]istiomodel.Config, []istiomodel.Config, error) {

	istioConfigs, err := model.ConvertBindingsAndExposures([]istiomodel.Config{newconfig}, ci)
	if err != nil {
		return []istiomodel.Config{}, []istiomodel.Config{}, err
	}

	outAdditions := make([]istiomodel.Config, 0)
	outModifications := make([]istiomodel.Config, 0)
	for _, istioConfig := range istioConfigs {
		orig, ok := store.Get(istioConfig.Type, istioConfig.Name, istioConfig.Namespace)
		if !ok {
			outAdditions = append(outAdditions, istioConfig)
		} else {
			if !reflect.DeepEqual(istioConfig.Spec, orig.Spec) {
				outModifications = append(outModifications, istioConfig)
			}
		}
	}

	return outAdditions, outModifications, nil
}

// ModifyMulticlusterConfig takes an Istio config store and a modified RemoteServiceBinding or ServiceExpositionPolicy
// and returns the new and modified Istio configurations needed to implement the desired multicluster config.
func ModifyMulticlusterConfig(store istiomodel.ConfigStore, config istiomodel.Config, ci model.ClusterInfo) ([]istiomodel.Config, error) {

	istioConfigs, err := model.ConvertBindingsAndExposures([]istiomodel.Config{config}, ci)
	if err != nil {
		return []istiomodel.Config{}, err
	}

	outModifications := make([]istiomodel.Config, 0)
	for _, istioConfig := range istioConfigs {
		orig, ok := store.Get(istioConfig.Type, istioConfig.Name, istioConfig.Namespace)
		if !ok {
			return nil, fmt.Errorf("Expected to modify Istio config but %#v makes an unknown config %#v", config, istioConfig)
		} else {
			if !reflect.DeepEqual(istioConfig.Spec, orig.Spec) {
				outModifications = append(outModifications, istioConfig)
			}
		}
	}

	return outModifications, nil
}

// DeleteMulticlusterConfig takes an Istio config store and a deleted RemoteServiceBinding or ServiceExpositionPolicy
// and returns the Istio configurations that should be removed to disable the multicluster config.
// Only the Type, Name, and Namespace of the output configs is guarenteed usable.
func DeleteMulticlusterConfig(store istiomodel.ConfigStore, config istiomodel.Config, ci model.ClusterInfo) ([]istiomodel.Config, error) {

	var err error
	istioConfigs, err := model.ConvertBindingsAndExposures([]istiomodel.Config{config}, ci)
	if err != nil {
		return nil, err
	}

	outDeletions := make([]istiomodel.Config, 0)
	for _, istioConfig := range istioConfigs {
		orig, ok := store.Get(istioConfig.Type, istioConfig.Name, istioConfig.Namespace)
		if !ok {
			err = multierror.Append(err, fmt.Errorf("%s %s.%s should have been realized by %s %s.%s; skipping",
				config.Type, config.Name, config.Namespace,
				istioConfig.Type, istioConfig.Name, istioConfig.Namespace))
		} else {
			// Only delete if our annotation is present
			_, ok := orig.Annotations[model.ProvenanceAnnotationKey]
			if ok {
				istioConfig.Spec = nil // Don't let caller see the details, their job is to delete based on Kind and Name
				outDeletions = append(outDeletions, istioConfig)
			} else {
				log.Infof("Ignoring unprovenanced %s %s.%s when reconciling deletion",
					istioConfig.Type, istioConfig.Name, istioConfig.Namespace)
			}
		}
	}

	return outDeletions, err
}