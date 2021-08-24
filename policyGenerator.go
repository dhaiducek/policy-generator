package policygen

import (
	"fmt"

	"github.com/dhaiducek/policy-generator/policybuilder"
	"github.com/dhaiducek/policy-generator/utils"
	yaml "gopkg.in/yaml.v3"
)

type PolicyGenerator struct {
	sourcePoliciesPath string
	policyGenTempPath  string
	outPath            string
	stdout             bool
	customResources    bool
}

func GeneratePolicies(generator *PolicyGenerator) {

	fHandler := utils.NewFilesHandler(generator.sourcePoliciesPath, generator.policyGenTempPath, generator.outPath)

	for _, file := range fHandler.GetPolicyGenTemplates() {
		policyGenTemp := utils.PolicyGenTemplate{}
		yamlFile := fHandler.ReadPolicyGenTempFile(file.Name())
		err := yaml.Unmarshal(yamlFile, &policyGenTemp)
		if err != nil {
			panic(err)
		}
		pBuilder := policybuilder.NewPolicyBuilder(policyGenTemp, generator.sourcePoliciesPath)

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
