.PHONY: tidy build test vet image deploy-kind

tidy:
	$(MAKE) -C shared/methodingress-controller tidy
	$(MAKE) -C shared/logging tidy
	$(MAKE) -C shared/metrics tidy
	$(MAKE) -C shared/rpc tidy
	$(MAKE) -C services/brokers/kafka-publisher tidy

build:
	$(MAKE) -C shared/methodingress-controller build image
	$(MAKE) -C shared/logging build
	$(MAKE) -C shared/metrics build
	$(MAKE) -C shared/rpc build
	$(MAKE) -C services/brokers/kafka-publisher build

test:
	$(MAKE) -C shared/methodingress-controller test vet
	$(MAKE) -C shared/logging test
	$(MAKE) -C shared/metrics test
	$(MAKE) -C shared/rpc test
	$(MAKE) -C services/brokers/kafka-publisher test vet

vet:
	$(MAKE) -C shared/methodingress-controller vet
	$(MAKE) -C shared/logging vet
	$(MAKE) -C shared/metrics vet
	$(MAKE) -C shared/rpc vet
	$(MAKE) -C services/brokers/kafka-publisher vet

image:
	$(MAKE) -C shared/methodingress-controller image

deploy-kind:
	bash .github/scripts/deploy.sh
