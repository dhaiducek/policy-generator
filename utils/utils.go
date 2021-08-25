package utils

const NotApplicable = "N/A"
const FileExt = ".yaml"
const Common = "common"
const Groups = "groups"
const Sites = "sites"
const CommonNS = Common + "-sub"
const GroupNS = Groups + "-sub"
const SiteNS = Sites + "-sub"
const CustomResource = "customResource"

type PolicyGenTemplate struct {
	ApiVersion  string       `yaml:"apiVersion"`
	Kind        string       `yaml:"kind"`
	Metadata    metaData     `yaml:"metadata"`
	SourceFiles []SourceFile `yaml:"sourceFiles"`
}

type metaData struct {
	Name      string `yaml:"name"`
	Labels    labels `yaml:"labels"`
	Namespace string `yaml:"namespace"`
}

type labels struct {
	Common    bool   `yaml:"common"`
	GroupName string `yaml:"groupName"`
	SiteName  string `yaml:"siteName"`
	Mcp       string `yaml:"mcp"`
}

type SourceFile struct {
	FileName   string                 `yaml:"fileName"`
	PolicyName string                 `yaml:"policyName"`
	Name       string                 `yaml:"name"`
	Labels     map[string]string      `yaml:"labels"`
	Spec       map[string]interface{} `yaml:"spec"`
	Data       map[string]interface{} `yaml:"data"`
}
