/*
Copyright (c) Microsoft Corporation.
Licensed under the MIT license.
*/

package internalsvcexport

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	fleetnetv1alpha1 "go.goms.io/fleet-networking/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	hubNSForMember       = "bravelion"
	memberUserNS         = "work"
	svcName              = "app"
	svcNameConflicted    = "app2"
	svcNameNotConflicted = "app3"
)

// serviceExportConflictTrueCondition returns a true ServiceExportConflict condition.
func serviceExportConflictTrueCondition(svcNamespace string, svcName string) metav1.Condition {
	return metav1.Condition{
		Type:               string(fleetnetv1alpha1.ServiceExportConflict),
		Status:             metav1.ConditionTrue,
		ObservedGeneration: 1,
		LastTransitionTime: metav1.Now(),
		Reason:             "ConflictFound",
		Message:            fmt.Sprintf("service %s/%s is in conflict with other exported services", svcNamespace, svcName),
	}
}

// serviceExportConflictFalseCondition returns a false ServiceExportConflict condition.
func serviceExportConflictFalseCondition(svcNamespace string, svcName string) metav1.Condition {
	return metav1.Condition{
		Type:               string(fleetnetv1alpha1.ServiceExportConflict),
		Status:             metav1.ConditionFalse,
		ObservedGeneration: 2,
		LastTransitionTime: metav1.Now(),
		Reason:             "NoConflictFound",
		Message:            fmt.Sprintf("service %s/%s is exported without conflict", svcNamespace, svcName),
	}
}

// serviceExportConflictUnknownCondition returns an unknown ServiceExportConflict condition.
func serviceExportConflictUnknownCondition(svcNamespace string, svcName string) metav1.Condition {
	return metav1.Condition{
		Type:               string(fleetnetv1alpha1.ServiceExportConflict),
		Status:             metav1.ConditionUnknown,
		ObservedGeneration: 0,
		LastTransitionTime: metav1.Now(),
		Reason:             "PendingConflictResolution",
		Message:            fmt.Sprintf("service %s/%s is pending export conflict resolution", svcNamespace, svcName),
	}
}

// TestReportBackConflictCondition tests the *Reconciler.reportBackConflictCondition method.
func TestReportBackConflictCondition(t *testing.T) {
	// Setup
	internalSvcExportEmpty := &fleetnetv1alpha1.InternalServiceExport{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: hubNSForMember,
			Name:      fmt.Sprintf("%s-%s", memberUserNS, svcName),
		},
	}
	svcExportEmpty := &fleetnetv1alpha1.ServiceExport{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: memberUserNS,
			Name:      svcName,
		},
		Status: fleetnetv1alpha1.ServiceExportStatus{
			Conditions: []metav1.Condition{
				serviceExportConflictUnknownCondition(memberUserNS, svcName),
			},
		},
	}

	internalSvcExportNotConflicted := &fleetnetv1alpha1.InternalServiceExport{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: hubNSForMember,
			Name:      fmt.Sprintf("%s-%s", memberUserNS, svcNameNotConflicted),
		},
		Status: fleetnetv1alpha1.InternalServiceExportStatus{
			Conditions: []metav1.Condition{
				serviceExportConflictFalseCondition(memberUserNS, svcNameNotConflicted),
			},
		},
	}
	svcExportNotConflicted := &fleetnetv1alpha1.ServiceExport{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: memberUserNS,
			Name:      svcNameNotConflicted,
		},
		Status: fleetnetv1alpha1.ServiceExportStatus{
			Conditions: []metav1.Condition{
				serviceExportConflictFalseCondition(memberUserNS, svcNameNotConflicted),
			},
		},
	}

	internalSvcExportConflicted := &fleetnetv1alpha1.InternalServiceExport{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: hubNSForMember,
			Name:      fmt.Sprintf("%s-%s", memberUserNS, svcNameConflicted),
		},
		Status: fleetnetv1alpha1.InternalServiceExportStatus{
			Conditions: []metav1.Condition{
				serviceExportConflictTrueCondition(memberUserNS, svcNameConflicted),
			},
		},
	}
	svcExportConflicted := &fleetnetv1alpha1.ServiceExport{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: memberUserNS,
			Name:      svcNameConflicted,
		},
		Status: fleetnetv1alpha1.ServiceExportStatus{
			Conditions: []metav1.Condition{
				serviceExportConflictUnknownCondition(memberUserNS, svcName),
			},
		},
	}

	fakeMemberClient := fakeclient.NewClientBuilder().
		WithScheme(scheme.Scheme).
		WithObjects(svcExportEmpty, svcExportNotConflicted, svcExportConflicted).
		Build()
	fakeHubClient := fakeclient.NewClientBuilder().Build()
	reconciler := Reconciler{
		memberClient: fakeMemberClient,
		hubClient:    fakeHubClient,
	}
	ctx := context.Background()

	testCases := []struct {
		name              string
		svcExport         *fleetnetv1alpha1.ServiceExport
		internalSvcExport *fleetnetv1alpha1.InternalServiceExport
		expectedConds     []metav1.Condition
	}{
		{
			name:              "should not report back conflict cond (no condition yet)",
			svcExport:         svcExportEmpty,
			internalSvcExport: internalSvcExportEmpty,
			expectedConds: []metav1.Condition{
				serviceExportConflictUnknownCondition(memberUserNS, svcName),
			},
		},
		{
			name:              "should not report back conflict cond (no update)",
			svcExport:         svcExportNotConflicted,
			internalSvcExport: internalSvcExportNotConflicted,
			expectedConds: []metav1.Condition{
				serviceExportConflictFalseCondition(memberUserNS, svcNameNotConflicted),
			},
		},
		{
			name:              "should report back conflict cond",
			svcExport:         svcExportConflicted,
			internalSvcExport: internalSvcExportConflicted,
			expectedConds: []metav1.Condition{
				serviceExportConflictTrueCondition(memberUserNS, svcNameConflicted),
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := reconciler.reportBackConflictCondition(ctx, tc.svcExport, tc.internalSvcExport)
			if err != nil {
				t.Fatalf("failed to report back conflict cond: %v", err)
			}

			var updatedSvcExport = &fleetnetv1alpha1.ServiceExport{}
			err = fakeMemberClient.Get(ctx,
				types.NamespacedName{Namespace: tc.svcExport.Namespace, Name: tc.svcExport.Name},
				updatedSvcExport)
			if err != nil {
				t.Fatalf("failed to get updated svc export: %v", err)
			}
			conds := updatedSvcExport.Status.Conditions
			if !cmp.Equal(conds, tc.expectedConds) {
				t.Fatalf("conds are not correctly updated, got %+v, want %+v", conds, tc.expectedConds)
			}
		})
	}
}
