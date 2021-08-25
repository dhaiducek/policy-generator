package policybuilder

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"reflect"
	"strings"

	"github.com/dhaiducek/policy-generator/utils"
	configpolicyv1 "github.com/open-cluster-management/config-policy-controller/pkg/apis/policy/v1"
	policyv1 "github.com/open-cluster-management/governance-policy-propagator/pkg/apis/policy/v1"
	"gopkg.in/yaml.v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type PolicyBuilder struct {
	PolicyGenTemplate utils.PolicyGenTemplate
	SourcePoliciesDir string
}

func NewPolicyBuilder(PolicyGenTemplate utils.PolicyGenTemplate, SourcePoliciesDir string) *PolicyBuilder {
	return &PolicyBuilder{PolicyGenTemplate: PolicyGenTemplate, SourcePoliciesDir: SourcePoliciesDir}
}

func (pbuilder *PolicyBuilder) Build(customResourseOnly bool) map[string]interface{} {
	policies := make(map[string]interface{})

	if len(pbuilder.PolicyGenTemplate.SourceFiles) != 0 && !customResourseOnly {
		if pbuilder.PolicyGenTemplate.Metadata.Name == "" || pbuilder.PolicyGenTemplate.Metadata.Name == utils.NotApplicable {
			panic("Error: missing policy template metadata.Name")
		}
		namespace, path, matchKey, matchValue, matchOper := pbuilder.getPolicyNsPath()
		subjects := make([]policyv1.Subject, 0)

		for _, sFile := range pbuilder.PolicyGenTemplate.SourceFiles {
			pname := pbuilder.getPolicyName()
			// pname is the policyName prefix common|{groupName}|{siteName}
			name := pname + "-" + sFile.PolicyName
			if err := CheckNameLength(namespace, name); err != nil {
				panic(err)
			}

			sPolicyFile, err := ioutil.ReadFile(pbuilder.SourcePoliciesDir + "/" + sFile.FileName + utils.FileExt)
			if err != nil {
				panic(err)
			}
			resourcesDef := pbuilder.getCustomResources(sFile, sPolicyFile)
			acmPolicy := pbuilder.getPolicy(name, namespace, resourcesDef)
			policies[path+"/"+name] = acmPolicy
			subject := CreatePolicySubject(name)
			subjects = append(subjects, subject)
		}
		placementRule := CreatePlacementRule(pbuilder.PolicyGenTemplate.Metadata.Name, namespace, matchKey, matchOper, matchValue)

		if err := CheckNameLength(namespace, placementRule.Name); err != nil {
			panic(err)
		}
		policies[path+"/"+placementRule.Name] = placementRule
		placementBinding := CreatePlacementBinding(pbuilder.PolicyGenTemplate.Metadata.Name, namespace, placementRule.Name, subjects)

		if err := CheckNameLength(namespace, placementBinding.Name); err != nil {
			panic(err)
		}
		policies[path+"/"+placementBinding.Name] = placementBinding
	} else if len(pbuilder.PolicyGenTemplate.SourceFiles) != 0 && customResourseOnly {
		for _, sFile := range pbuilder.PolicyGenTemplate.SourceFiles {
			sPolicyFile, err := ioutil.ReadFile(pbuilder.SourcePoliciesDir + "/" + sFile.FileName + utils.FileExt)

			if err != nil {
				panic(err)
			}
			resources := pbuilder.getCustomResources(sFile, sPolicyFile)

			for _, resource := range resources {
				name := resource["kind"].(string)
				name = name + "-" + resource["metadata"].(map[string]interface{})["name"].(string)

				if resource["metadata"].(map[string]interface{})["namespace"] != nil {
					name = name + "-" + resource["metadata"].(map[string]interface{})["namespace"].(string)
				}
				policies[utils.CustomResource+"/"+name] = resource
			}
		}
	}
	return policies
}

func (pbuilder *PolicyBuilder) getCustomResources(sFile utils.SourceFile, sPolicyFile []byte) []map[string]interface{} {
	yamls, err := pbuilder.splitYamls(sPolicyFile)
	resources := make([]map[string]interface{}, 0)

	if err != nil {
		panic(err)
	}
	// We are not allowing multiple yamls structure in same file to update its spec/data.
	if len(yamls) > 1 && (len(sFile.Data) > 0 || len(sFile.Spec) > 0) {
		panic("Update spec/data of multiple yamls structure in same file " + sFile.FileName +
			" not allowed. Instead separate them in multiple files")
	} else if len(yamls) > 1 && len(sFile.Data) == 0 && len(sFile.Spec) == 0 {
		for _, yaml := range yamls {
			resources = append(resources, pbuilder.getCustomResource(nil, nil, sFile.Labels, yaml, "", pbuilder.PolicyGenTemplate.Metadata.Labels.Mcp))
		}
	} else if len(yamls) == 1 {
		resources = append(resources, pbuilder.getCustomResource(sFile.Data, sFile.Spec, sFile.Labels, yamls[0], sFile.Name, pbuilder.PolicyGenTemplate.Metadata.Labels.Mcp))
	}
	return resources
}

func (pbuilder *PolicyBuilder) getCustomResource(data map[string]interface{}, spec map[string]interface{}, labels map[string]string, sourcePolicy []byte, name string, mcp string) map[string]interface{} {
	sourcePolicyMap := make(map[string]interface{})
	sourcePolicyStr := string(sourcePolicy)

	if mcp != "" && mcp != utils.NotApplicable {
		sourcePolicyStr = strings.Replace(sourcePolicyStr, "$mcp", mcp, -1)
	}
	err := yaml.Unmarshal([]byte(sourcePolicyStr), &sourcePolicyMap)

	if err != nil {
		panic(err)
	}
	if name != "" && name != utils.NotApplicable {
		sourcePolicyMap["metadata"].(map[string]interface{})["name"] = name
	}
	if len(labels) != 0 {
		sourcePolicyMap["metadata"].(map[string]interface{})["labels"] = labels
	}
	if sourcePolicyMap["spec"] != nil {
		sourcePolicyMap["spec"] = pbuilder.setValues(sourcePolicyMap["spec"].(map[string]interface{}), spec)
	}
	if sourcePolicyMap["data"] != nil {
		sourcePolicyMap["data"] = pbuilder.setValues(sourcePolicyMap["data"].(map[string]interface{}), data)
	}
	return sourcePolicyMap
}

func (pbuilder *PolicyBuilder) setValues(sourceMap map[string]interface{}, valueMap map[string]interface{}) map[string]interface{} {
	for k, v := range sourceMap {
		if valueMap[k] == nil {
			if reflect.ValueOf(v).Kind() == reflect.String && (v.(string) == "" || strings.HasPrefix(v.(string), "$")) {
				delete(sourceMap, k)
			}
			continue
		}
		if reflect.ValueOf(sourceMap[k]).Kind() == reflect.Map {
			sourceMap[k] = pbuilder.setValues(v.(map[string]interface{}), valueMap[k].(map[string]interface{}))
		} else if reflect.ValueOf(v).Kind() == reflect.Slice ||
			reflect.ValueOf(v).Kind() == reflect.Array {
			intfArray := v.([]interface{})

			if len(intfArray) > 0 && reflect.ValueOf(intfArray[0]).Kind() == reflect.Map {
				tmpMapValues := make([]map[string]interface{}, len(intfArray))
				vIntfArray := valueMap[k].([]interface{})

				for id, intfMap := range intfArray {
					if id < len(vIntfArray) {
						tmpMapValues[id] = pbuilder.setValues(intfMap.(map[string]interface{}), vIntfArray[id].(map[string]interface{}))
					} else {
						tmpMapValues[id] = intfMap.(map[string]interface{})
					}
				}
				sourceMap[k] = tmpMapValues
			} else {
				sourceMap[k] = valueMap[k]
			}
		} else {
			sourceMap[k] = valueMap[k]
		}
	}
	return sourceMap
}

func (pbuilder *PolicyBuilder) splitYamls(yamls []byte) ([][]byte, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(yamls))
	var resources [][]byte

	for {
		var resIntf interface{}
		err := decoder.Decode(&resIntf)

		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		resBytes, err := yaml.Marshal(resIntf)

		if err != nil {
			return nil, err
		}
		resources = append(resources, resBytes)
	}
	return resources, nil
}

func (pbuilder *PolicyBuilder) getPolicy(name string, namespace string, resources []map[string]interface{}) policyv1.Policy {
	if err := CheckNameLength(namespace, name); err != nil {
		panic(err)
	}
	objTempArr := make([]*configpolicyv1.ObjectTemplate, 0)

	for _, resource := range resources {
		objTempArr = append(objTempArr, CreateObjTemplates(resource))
	}
	acmConfigPolicy := CreateAcmConfigPolicy(name, objTempArr)
	policyObjDef := CreatePolicyObjectDefinition(acmConfigPolicy)
	policyObjDefRaw, err := json.Marshal(policyObjDef)
	if err != nil {
		panic(err)
	}
	policyObj := policyv1.PolicyTemplate{
		ObjectDefinition: runtime.RawExtension{
			Raw: policyObjDefRaw,
		},
	}
	policyObjDefArr := make([]*policyv1.PolicyTemplate, 1)
	policyObjDefArr = append(policyObjDefArr, &policyObj)
	acmPolicy := CreateAcmPolicy(name, namespace, policyObjDefArr)

	return acmPolicy
}

func (pbuilder *PolicyBuilder) getPolicyNsPath() (string, string, string, string, metav1.LabelSelectorOperator) {
	ns := ""
	path := ""
	matchKey := ""
	var matchOper metav1.LabelSelectorOperator
	matchValue := ""

	if pbuilder.PolicyGenTemplate.Metadata.Name != "" {
		if pbuilder.PolicyGenTemplate.Metadata.Labels.SiteName != utils.NotApplicable {
			ns = utils.SiteNS
			matchKey = utils.Sites
			matchOper = metav1.LabelSelectorOpIn
			matchValue = pbuilder.PolicyGenTemplate.Metadata.Labels.SiteName
			path = utils.Sites + "/" + pbuilder.PolicyGenTemplate.Metadata.Labels.SiteName
		} else if pbuilder.PolicyGenTemplate.Metadata.Labels.GroupName != utils.NotApplicable {
			ns = utils.GroupNS
			matchKey = pbuilder.PolicyGenTemplate.Metadata.Labels.GroupName
			matchOper = metav1.LabelSelectorOpExists
			path = utils.Groups + "/" + pbuilder.PolicyGenTemplate.Metadata.Labels.GroupName
		} else if pbuilder.PolicyGenTemplate.Metadata.Labels.Common {
			ns = utils.CommonNS
			matchKey = utils.Common
			matchOper = metav1.LabelSelectorOpIn
			matchValue = "true"
			path = utils.Common
		} else {
			panic("Error: missing metadata info either siteName, groupName or common should be set")
		}
	}
	return ns, path, matchKey, matchValue, matchOper
}

func (pbuilder *PolicyBuilder) getPolicyName() string {
	pname := ""

	if pbuilder.PolicyGenTemplate.Metadata.Labels.SiteName != utils.NotApplicable {
		pname = pbuilder.PolicyGenTemplate.Metadata.Labels.SiteName
	} else if pbuilder.PolicyGenTemplate.Metadata.Labels.GroupName != utils.NotApplicable {
		pname = pbuilder.PolicyGenTemplate.Metadata.Labels.GroupName
	} else if pbuilder.PolicyGenTemplate.Metadata.Labels.Common {
		pname = utils.Common
	} else {
		panic("Error: missing metadata info either siteName, groupName or common should be set")
	}
	// The names in the yaml must be compliant RFC 1123 domain names (all lower case)
	return strings.ToLower(pname)
}
