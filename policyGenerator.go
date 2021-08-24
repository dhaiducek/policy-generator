package policygen

import (
	"fmt"

	"github.com/dhaiducek/policy-generator/policybuilder"
	"github.com/dhaiducek/policy-generator/utils"
	"gopkg.in/yaml.v3"
)

type PolicyGenerator struct {
	sourceResourcePath    string
	policyGenTemplatePath string
	outPath               string
	stdout                bool
	customResources       bool
}

func NewPolicyGenerator(sourceResourcePath string, policyGenTemplatePath string, outPath string, stdout bool, customResources bool) *PolicyGenerator {
	// Set default paths
	if sourceResourcePath == "" {
		sourceResourcePath = "."
	}
	if policyGenTemplatePath == "" {
		policyGenTemplatePath = "."
	}
	if outPath == "" {
		outPath = "./policies"
	}
	// Return generator (booleans default to "false")
	return &PolicyGenerator{
		sourceResourcePath,
		policyGenTemplatePath,
		outPath,
		stdout,
		customResources,
	}
}

func (generator *PolicyGenerator) GeneratePolicies() {

	fHandler := utils.NewFilesHandler(generator.sourceResourcePath, generator.policyGenTemplatePath, generator.outPath)

	for _, file := range fHandler.GetPolicyGenTemplates() {
		policyGenTemplate := utils.PolicyGenTemplate{}
		yamlFile := fHandler.ReadPolicyGenTemplateFile(file.Name())
		err := yaml.Unmarshal(yamlFile, &policyGenTemplate)
		if err != nil {
			panic(err)
		}
		pBuilder := policybuilder.NewPolicyBuilder(policyGenTemplate, generator.sourceResourcePath)

		for k, v := range pBuilder.Build(generator.customResources) {
			policy, _ := yaml.Marshal(v)
			if generator.stdout {
				fmt.Println("---")
				fmt.Println(string(policy))
			}
			fHandler.WriteFile(k+utils.FileExt, policy)
		}
	}
}
