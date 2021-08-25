package policybuilder

import (
	"errors"
	"strings"

	"github.com/dhaiducek/policy-generator/utils"
	configpolicyv1 "github.com/open-cluster-management/config-policy-controller/pkg/apis/policy/v1"
	placementv1 "github.com/open-cluster-management/governance-policy-propagator/pkg/apis/apps/v1"
	policyv1 "github.com/open-cluster-management/governance-policy-propagator/pkg/apis/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func CreateAcmPolicy(name string, namespace string, policyObjDefArr []*policyv1.PolicyTemplate) policyv1.Policy {
	policy := policyv1.Policy{}
	policy.Name = name
	// TODO: Make annotations configurable
	annotations := make(map[string]string, 3)
	annotations["policy.open-cluster-management.io/standards"] = "NIST SP 800-53"
	annotations["policy.open-cluster-management.io/categories"] = "CM Configuration Management"
	annotations["policy.open-cluster-management.io/controls"] = "CM-2 Baseline Configuration"
	policy.Annotations = annotations
	policy.Namespace = namespace
	policy.Spec.Disabled = false
	policy.Spec.RemediationAction = "enforce"
	policy.Spec.PolicyTemplates = policyObjDefArr

	return policy
}

func CreateAcmConfigPolicy(name string, objTempArr []*configpolicyv1.ObjectTemplate) configpolicyv1.ConfigurationPolicy {
	configPolicy := configpolicyv1.ConfigurationPolicy{}
	configPolicy.Name = name + "-config"
	configPolicy.Spec.RemediationAction = "enforce"
	configPolicy.Spec.Severity = "low"
	exclude := make([]string, 1)
	exclude[0] = "kube-*"
	configPolicy.Spec.NamespaceSelector.Exclude = exclude
	include := make([]string, 1)
	include[0] = "*"
	configPolicy.Spec.NamespaceSelector.Include = include
	configPolicy.Spec.ObjectTemplates = objTempArr

	return configPolicy
}

func CreateObjTemplates(objDef runtime.RawExtension) configpolicyv1.ObjectTemplate {
	objTemp := configpolicyv1.ObjectTemplate{}
	objTemp.ComplianceType = "musthave"
	objTemp.ObjectDefinition = objDef

	return objTemp
}

func CreatePolicyObjectDefinition(acmConfigPolicy configpolicyv1.ConfigurationPolicy) configpolicyv1.ConfigurationPolicy {
	policyObjDef := configpolicyv1.ConfigurationPolicy{}
	policyObjDef = acmConfigPolicy

	return policyObjDef
}

func CreatePlacementBinding(name string, namespace string, ruleName string, subjects []policyv1.Subject) policyv1.PlacementBinding {
	placementBinding := policyv1.PlacementBinding{}
	placementBinding.Name = name + "-placementbinding"
	placementBinding.Namespace = namespace
	placementBinding.PlacementRef.Name = ruleName
	placementBinding.PlacementRef.Kind = "PlacementRule"
	placementBinding.PlacementRef.APIGroup = "apps.open-cluster-management.io"
	placementBinding.Subjects = subjects

	return placementBinding
}

func CreatePolicySubject(policyName string) policyv1.Subject {
	subject := policyv1.Subject{}
	subject.Name = policyName

	return subject
}

func CreatePlacementRule(name string, namespace string, matchKey string, matchOper metav1.LabelSelectorOperator, matchValue string) placementv1.PlacementRule {
	placmentRule := placementv1.PlacementRule{}
	placmentRule.Name = name + "-placementrule"
	placmentRule.Namespace = namespace
	expression := &metav1.LabelSelectorRequirement{}
	expression.Key = matchKey
	expression.Operator = matchOper
	if matchOper != utils.ExistOper {
		expression.Values = strings.Split(matchValue, ",")
	}
	placmentRule.Spec.ClusterSelector.MatchExpressions = append(placmentRule.Spec.ClusterSelector.MatchExpressions, *expression)

	return placmentRule
}

func CheckNameLength(namespace string, name string) error {
	// the policy (namespace.name + name) must not exceed 63 chars based on ACM documentation.
	if len(namespace+"."+name) > 63 {
		err := errors.New("Namespace.Name + ResourceName is exceeding the 63 chars limit: " + namespace + "." + name)
		return err
	}
	return nil
}
