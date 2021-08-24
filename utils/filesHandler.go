package utils

import (
	"io/ioutil"
	"os"
	"strings"
)

type filesHandler struct {
	sourcePoliciesDir    string
	policyGenTemplateDir string
	outDir               string
}

func NewFilesHandler(sourcePoliciesDir string, policyGenTemplateDir string, outDir string) *filesHandler {
	return &filesHandler{sourcePoliciesDir: sourcePoliciesDir, policyGenTemplateDir: policyGenTemplateDir, outDir: outDir}
}

func (fHandler *filesHandler) WriteFile(filePath string, content []byte) {
	path := fHandler.outDir + "/" + filePath[:strings.LastIndex(filePath, "/")]
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.MkdirAll(path, 0775)
	}

	err := ioutil.WriteFile(fHandler.outDir+"/"+filePath, content, 0644)
	if err != nil {
		panic(err)
	}
}

func (fHandler *filesHandler) GetPolicyGenTemplates() []os.FileInfo {
	files, err := ioutil.ReadDir(fHandler.policyGenTemplateDir)
	if err != nil {
		panic(err)
	}
	return files
}

func (fHandler *filesHandler) ReadPolicyGenTemplateFile(fileName string) []byte {
	file, err := ioutil.ReadFile(fHandler.policyGenTemplateDir + "/" + fileName)
	if err != nil {
		panic(err)
	}
	return file
}
