module code.cloudfoundry.org/cf-operator

require (
	code.cloudfoundry.org/quarks-job v0.0.0-20190927121335-b6e2e4acbee4
	code.cloudfoundry.org/quarks-utils v0.0.0-20190925163230-1117fd7b77f7
	github.com/cloudflare/cfssl v0.0.0-20181102015659-ea4033a214e7
	github.com/cloudfoundry/bosh-cli v5.4.0+incompatible
	github.com/cppforlife/go-patch v0.0.0-20171006213518-250da0e0e68c
	github.com/dchest/uniuri v0.0.0-20160212164326-8902c56451e9
	github.com/fsnotify/fsnotify v1.4.7
	github.com/go-logr/zapr v0.1.1
	github.com/go-test/deep v1.0.4
	github.com/golang/mock v1.3.1
	github.com/gonvenience/bunt v1.1.1
	github.com/hpcloud/tail v1.0.0
	github.com/imdario/mergo v0.3.7
	github.com/onsi/ginkgo v1.10.1
	github.com/onsi/gomega v1.6.0
	github.com/pkg/errors v0.8.1
	github.com/spf13/afero v1.2.2
	github.com/spf13/cobra v0.0.5
	github.com/spf13/viper v1.3.2
	github.com/viovanov/bosh-template-go v0.0.0-20190801125410-a195ef3de03a
	go.uber.org/zap v1.10.0
	golang.org/x/crypto v0.0.0-20190820162420-60c769a6c586
	golang.org/x/net v0.0.0-20190813141303-74dc4d7220e7
	gopkg.in/yaml.v2 v2.2.2
	k8s.io/api v0.0.0-20190409021203-6e4e0e4f393b
	k8s.io/apiextensions-apiserver v0.0.0-20190409022649-727a075fdec8
	k8s.io/apimachinery v0.0.0-20190404173353-6a84e37a896d
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	sigs.k8s.io/controller-runtime v0.2.2
	sigs.k8s.io/yaml v1.1.0
)

go 1.13
