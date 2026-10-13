.PHONY: tidy build test vet image deploy-kind

tidy:
	$(MAKE) -C shared/methodingress-controller tidy
	$(MAKE) -C shared/metrics tidy
	$(MAKE) -C services/brokers/kafka-publisher tidy

build:
	$(MAKE) -C shared/methodingress-controller build image
	$(MAKE) -C shared/metrics build
	$(MAKE) -C services/brokers/kafka-publisher build

test:
	$(MAKE) -C shared/methodingress-controller test vet
	$(MAKE) -C shared/metrics test
	$(MAKE) -C services/brokers/kafka-publisher test vet

vet:
	$(MAKE) -C shared/methodingress-controller vet
	$(MAKE) -C shared/metrics vet
	$(MAKE) -C services/brokers/kafka-publisher vet

image:
	$(MAKE) -C shared/methodingress-controller image

deploy-kind:
	bash .github/scripts/deploy.sh
