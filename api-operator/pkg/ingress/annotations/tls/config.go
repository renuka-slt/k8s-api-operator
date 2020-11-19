package tls

import (
	"github.com/wso2/k8s-api-operator/api-operator/pkg/ingress/annotations/parser"
	networking "k8s.io/api/networking/v1beta1"
	"regexp"
	"strings"
)

const (
	Simple = "simple"
	Mutual = "mtls"
)

// Annotations suffixes
const (
	passthroughEnabledKey = "passthrough-enabled"
	terminationModeKey    = "tls-termination-mode"
	originationCertsKey   = "origination-certs"
)

var (
	terminationModeRegex = regexp.MustCompile(`^(simple|mtls)$`)
)

type Config struct {
	// PassthroughEnabled is true make TLS passthrough is applied all hosts defined in spec.tls.hosts
	// Secrets defined in spec.tls is ignored in TLS passthrough mode
	// When PassthroughEnabled is true, TLS origination is ignored
	// Default to false
	PassthroughEnabled bool

	// TerminationMode is the TLS termination mode
	// Could be one of "simple" or "mtls"
	// Default to "simple"
	TerminationMode string

	// OriginationCerts defines the certs for TLS origination
	OriginationCerts []string
}

func Parse(ing *networking.Ingress) Config {
	conf := Config{}
	var err error

	conf.PassthroughEnabled, err = parser.GetBoolAnnotation(ing, passthroughEnabledKey)
	if err != nil {
		conf.PassthroughEnabled = false
	}

	conf.TerminationMode, err = parser.GetStringAnnotation(ing, terminationModeKey)
	if err != nil || terminationModeRegex.MatchString(conf.TerminationMode) {
		conf.TerminationMode = Simple
	}

	secretStr, _ := parser.GetStringAnnotation(ing, originationCertsKey)
	secrets := strings.Split(secretStr, ",")
	for _, secret := range secrets {
		conf.OriginationCerts = append(conf.OriginationCerts, strings.TrimSpace(secret))
	}

	return conf
}
