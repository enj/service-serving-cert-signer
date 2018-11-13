package api

import "strings"

const (
	InjectCABundleAnnotationName = "service.alpha.openshift.io/inject-cabundle"
	InjectionDataKey             = "service-ca.crt"
)

func HasInjectCABundleAnnotation(annotations map[string]string) bool {
	return strings.EqualFold(annotations[InjectCABundleAnnotationName], "true")
}
