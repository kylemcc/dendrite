package dendrite

import (
	"github.com/kylelemons/go-gypsy/yaml"

	"github.com/fizx/logs"
	"regexp"
	"strconv"
)

type FieldType int

const (
	String = iota
	Tokens
	Integer
	Gauge
	Metric
	Counter
	Timestamp
)

type FieldSpec struct {
	Name    string
	Alias   string
	Type    FieldType
	Group   int
	Format  string
	Pattern *regexp.Regexp
}

type ConfigGroup struct {
	Glob      string
	Pattern   string
	Fields    []FieldSpec
	Config    Config
	Name      string
	OffsetDir string
	Encoder   Encoder
}

type Config struct {
	Protocol  string
	Address   string
	OffsetDir string
	Encoder   Encoder
	Groups    []ConfigGroup
}

func (config *Config) AddGroup(name string, group yaml.Node) {
  logs.Info("Adding group: %s", name)
	groupMap := group.(yaml.Map)
	var groupStruct ConfigGroup
	groupStruct.Name = name
	groupStruct.Glob = groupMap.Key("glob").(yaml.Scalar).String()
	groupStruct.Pattern = groupMap.Key("pattern").(yaml.Scalar).String()
	groupStruct.Fields = make([]FieldSpec, 0)
	groupStruct.OffsetDir = config.OffsetDir
	groupStruct.Encoder = config.Encoder

	for alias, v := range groupMap.Key("fields").(yaml.Map) {
		var fieldDetails = v.(yaml.Map)
		var fieldSpec FieldSpec
		fieldSpec.Alias = alias
		fieldSpec.Name = alias

		tmp, _ := yaml.Child(fieldDetails, ".name")
		if tmp != nil {
			fieldSpec.Name = tmp.(yaml.Scalar).String()
		}

		fieldSpec.Group = -1
		tmp, _ = yaml.Child(fieldDetails, ".group")
		if tmp != nil {
			fieldSpec.Name = ""
			i, err := strconv.ParseInt(tmp.(yaml.Scalar).String(), 10, 64)
			if err != nil {
				logs.Error("error in parsing int", err)
			}

			fieldSpec.Group = int(i)
		}

		tmp, _ = yaml.Child(fieldDetails, ".pattern")
		if tmp != nil {
			p, err := regexp.Compile(tmp.(yaml.Scalar).String())
			if err != nil {
				logs.Error("error in compiling regexp", err)
			} else {
				fieldSpec.Pattern = p
			}
		}

		tmp, _ = yaml.Child(fieldDetails, ".format")
		if tmp != nil {
			fieldSpec.Format = tmp.(yaml.Scalar).String()
		}

		tmp, _ = yaml.Child(fieldDetails, ".type")
		if tmp == nil {
			fieldSpec.Type = String
		} else {
			switch tmp.(yaml.Scalar).String() {
			case "int":
				fieldSpec.Type = Integer
			case "gauge":
				fieldSpec.Type = Gauge
			case "metric":
				fieldSpec.Type = Metric
			case "counter":
				fieldSpec.Type = Counter
			case "string":
				fieldSpec.Type = String
			case "tokenized":
				fieldSpec.Type = Tokens
			case "timestamp", "date":
				fieldSpec.Type = Timestamp
			default:
				logs.Error("Can't recognize field type")
				panic(nil)
			}
		}

		groupStruct.Fields = append(groupStruct.Fields, fieldSpec)
	}
	config.Groups = append(config.Groups, groupStruct)
}

func (config *Config) Populate(doc *yaml.File) {
	config.Groups = make([]ConfigGroup, 0)

	root := doc.Root.(yaml.Map)

	config.Address = root.Key("address").(yaml.Scalar).String()
	config.OffsetDir = root.Key("offset_dir").(yaml.Scalar).String()

	switch root.Key("encoder").(yaml.Scalar).String() {
	case "json":
		config.Encoder = new(JsonEncoder)
	case "statsd":
		config.Encoder = new(StatsdEncoder)
	}

	p, err := doc.Get(".protocol")
	if err != nil {
		p = "tcp"
	}
	config.Protocol = p

	groups := root.Key("groups")
	for name, group := range groups.(yaml.Map) {
		config.AddGroup(name, group)
	}
}
