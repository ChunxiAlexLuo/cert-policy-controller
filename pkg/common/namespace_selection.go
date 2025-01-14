// Licensed Materials - Property of IBM
// (c) Copyright IBM Corporation 2018, 2019. All Rights Reserved.
// Note to U.S. Government Users Restricted Rights:
// Use, duplication or disclosure restricted by GSA ADP Schedule
// Contract with IBM Corp.
// Copyright Contributors to the Open Cluster Management project

package common

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"

	policyv1 "open-cluster-management.io/cert-policy-controller/api/v1"
)

var log = ctrl.Log

// GetSelectedNamespaces returns the list of filtered namespaces according to the policy namespace selector.
func GetSelectedNamespaces(client kubernetes.Interface, selector policyv1.Target) ([]string, error) {
	// Build LabelSelector from provided MatchLabels and MatchExpressions
	var labelSelector metav1.LabelSelector
	// Handle when MatchLabels/MatchExpressions were not provided to prevent nil pointer dereference.
	// This is needed so that `include` can function independently. Not fetching any namespaces is the
	// responsibility of the calling function for when MatchLabels/MatchExpressions are both nil.
	matchLabels := map[string]string{}
	matchExpressions := []metav1.LabelSelectorRequirement{}

	if selector.MatchLabels != nil {
		matchLabels = *selector.MatchLabels
	}

	if selector.MatchExpressions != nil {
		matchExpressions = *selector.MatchExpressions
	}

	labelSelector = metav1.LabelSelector{
		MatchLabels:      matchLabels,
		MatchExpressions: matchExpressions,
	}

	// get all namespaces matching selector
	allNamespaces, err := GetAllNamespaces(client, labelSelector)
	if err != nil {
		log.Error(err, "error retrieving namespaces")

		return []string{}, err
	}

	// filter the list based on the included/excluded list of patterns
	included := selector.Include
	excluded := selector.Exclude
	log.V(2).Info("Filtering namespace list using include/exclude lists", "include", included, "exclude", excluded)

	finalList, err := Matches(allNamespaces, included, excluded)
	if err != nil {
		return []string{}, err
	}

	if len(finalList) == 0 {
		log.V(2).Info("Filtered namespace list is empty.")
	}

	log.V(2).Info("Returning final filtered namespace list", "namespaces", finalList)

	return finalList, err
}

// GetAllNamespaces gets the list of all namespaces from k8s.
func GetAllNamespaces(client kubernetes.Interface, labelSelector metav1.LabelSelector) ([]string, error) {
	parsedSelector, err := metav1.LabelSelectorAsSelector(&labelSelector)
	if err != nil {
		return nil, fmt.Errorf("error parsing namespace LabelSelector: %w", err)
	}

	listOpt := metav1.ListOptions{
		LabelSelector: parsedSelector.String(),
	}

	log.V(2).Info("Retrieving namespaces with LabelSelector", "LabelSelector", parsedSelector.String())

	nsList, err := client.CoreV1().Namespaces().List(context.TODO(), listOpt)
	if err != nil {
		log.Error(err, "could not list namespaces from the API server")

		return nil, err
	}

	var namespacesNames []string

	for _, n := range nsList.Items {
		namespacesNames = append(namespacesNames, n.Name)
	}

	return namespacesNames, nil
}
