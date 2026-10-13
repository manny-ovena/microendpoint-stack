.PHONY: build test vet image deploy-kind

build:
	$(MAKE) -C shared/methodingress-controller build image
	$(MAKE) -C shared/metrics build

test:
	$(MAKE) -C shared/methodingress-controller test vet
	$(MAKE) -C shared/metrics test

vet:
	$(MAKE) -C shared/methodingress-controller vet
	$(MAKE) -C shared/metrics vet

image:
	$(MAKE) -C shared/methodingress-controller image

deploy-kind:
	bash .github/scripts/deploy.sh
