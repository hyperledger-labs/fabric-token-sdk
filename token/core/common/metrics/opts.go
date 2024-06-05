/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package metrics

var (
	issuesOpts = CounterOpts{
		Namespace:    "token_sdk",
		Name:         "issue_operations",
		Help:         "The number of issue operations",
		LabelNames:   []string{"network", "channel", "namespace", "token_type"},
		StatsdFormat: "%{#fqname}.%{network}.%{channel}.%{namespace}.%{token_type}",
	}
	failedIssuesOpts = CounterOpts{
		Namespace:    "token_sdk",
		Name:         "failed_issue_operations",
		Help:         "The number of failed issue operations",
		LabelNames:   []string{"network", "channel", "namespace", "token_type"},
		StatsdFormat: "%{#fqname}.%{network}.%{channel}.%{namespace}.%{token_type}",
	}
	issueDurationOpts = HistogramOpts{
		Namespace:    "token_sdk",
		Name:         "issue_duration",
		Help:         "Duration of an issue operation",
		LabelNames:   []string{"network", "channel", "namespace"},
		StatsdFormat: "%{#fqname}.%{network}.%{channel}.%{namespace}",
	}
	transfersOpts = CounterOpts{
		Namespace:    "token_sdk",
		Name:         "transfer_operations",
		Help:         "The number of transfer operations",
		LabelNames:   []string{"network", "channel", "namespace"},
		StatsdFormat: "%{#fqname}.%{network}.%{channel}.%{namespace}",
	}
	failedTransfersOpts = CounterOpts{
		Namespace:    "token_sdk",
		Name:         "failed_transfer_operations",
		Help:         "The number of failed transfer operations",
		LabelNames:   []string{"network", "channel", "namespace"},
		StatsdFormat: "%{#fqname}.%{network}.%{channel}.%{namespace}",
	}
	transferDurationOpts = HistogramOpts{
		Namespace:    "token_sdk",
		Name:         "transfer_duration",
		Help:         "Duration of a transfer operation",
		LabelNames:   []string{"network", "channel", "namespace"},
		StatsdFormat: "%{#fqname}.%{network}.%{channel}.%{namespace}",
	}
	auditsOpts = CounterOpts{
		Namespace:    "token_sdk",
		Name:         "audit_operations",
		Help:         "The number of audit operations",
		LabelNames:   []string{"network", "channel", "namespace"},
		StatsdFormat: "%{#fqname}.%{network}.%{channel}.%{namespace}",
	}
	failedAuditsOpts = CounterOpts{
		Namespace:    "token_sdk",
		Name:         "failed_audit_operations",
		Help:         "The number of failed audit operations",
		LabelNames:   []string{"network", "channel", "namespace"},
		StatsdFormat: "%{#fqname}.%{network}.%{channel}.%{namespace}",
	}
	auditDurationOpts = HistogramOpts{
		Namespace:    "token_sdk",
		Name:         "audit_duration",
		Help:         "Duration of an audit operation",
		LabelNames:   []string{"network", "channel", "namespace"},
		StatsdFormat: "%{#fqname}.%{network}.%{channel}.%{namespace}",
	}
)
