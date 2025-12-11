module Diph

go 1.25.0

require k8s.io/apimachinery v0.34.2

require (
	github.com/Masterminds/sprig v2.22.0+incompatible
	github.com/fluxcd/kustomize-controller/api v1.7.3
	github.com/spf13/cobra v1.9.1
	github.com/spf13/pflag v1.0.6
	github.com/spf13/viper v1.11.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/fluxcd/pkg/apis/kustomize v1.14.0 // indirect
	github.com/fluxcd/pkg/apis/meta v1.23.0 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/huandu/xstrings v1.3.2 // indirect
	github.com/imdario/mergo v0.3.12 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/magiconair/properties v1.8.6 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/mapstructure v1.4.3 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.3-0.20250322232337-35a7c28c31ee // indirect
	github.com/pelletier/go-toml v1.9.4 // indirect
	github.com/pelletier/go-toml/v2 v2.0.0-beta.8 // indirect
	github.com/rogpeppe/go-internal v1.12.0 // indirect
	github.com/spf13/afero v1.8.2 // indirect
	github.com/spf13/cast v1.4.1 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/stretchr/testify v1.9.0 // indirect
	github.com/subosito/gotenv v1.2.0 // indirect
	go.yaml.in/yaml/v2 v2.4.2 // indirect
	golang.org/x/crypto v0.42.0 // indirect
	golang.org/x/net v0.45.0 // indirect
	golang.org/x/sys v0.36.0 // indirect
	golang.org/x/text v0.29.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/ini.v1 v1.66.4 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	k8s.io/apiextensions-apiserver v0.34.2 // indirect
	k8s.io/klog/v2 v2.130.1 // indirect
	k8s.io/utils v0.0.0-20250604170112-4c0f3b243397 // indirect
	sigs.k8s.io/controller-runtime v0.22.4 // indirect
	sigs.k8s.io/json v0.0.0-20241014173422-cfa47c3a1cc8 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.2.1 // indirect
	sigs.k8s.io/yaml v1.6.0 // indirect
)

replace k8s.io/api => k8s.io/api v0.23.1

replace k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.23.1

replace k8s.io/apimachinery => k8s.io/apimachinery v0.23.2-rc.0

replace k8s.io/apiserver => k8s.io/apiserver v0.23.1

replace k8s.io/cli-runtime => k8s.io/cli-runtime v0.23.1

replace k8s.io/client-go => k8s.io/client-go v0.23.1

replace k8s.io/cloud-provider => k8s.io/cloud-provider v0.23.1

replace k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.23.1

replace k8s.io/code-generator => k8s.io/code-generator v0.23.2-rc.0

replace k8s.io/component-base => k8s.io/component-base v0.23.1

replace k8s.io/component-helpers => k8s.io/component-helpers v0.23.1

replace k8s.io/controller-manager => k8s.io/controller-manager v0.23.1

replace k8s.io/cri-api => k8s.io/cri-api v0.23.4-rc.0

replace k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.23.1

replace k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.23.1

replace k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.23.1

replace k8s.io/kube-proxy => k8s.io/kube-proxy v0.23.1

replace k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.23.1

replace k8s.io/kubectl => k8s.io/kubectl v0.23.1

replace k8s.io/kubelet => k8s.io/kubelet v0.23.1

replace k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.23.1

replace k8s.io/metrics => k8s.io/metrics v0.23.1

replace k8s.io/mount-utils => k8s.io/mount-utils v0.23.2-rc.0

replace k8s.io/pod-security-admission => k8s.io/pod-security-admission v0.23.1

replace k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.23.1

replace k8s.io/sample-cli-plugin => k8s.io/sample-cli-plugin v0.23.1

replace k8s.io/sample-controller => k8s.io/sample-controller v0.23.1
