.PHONY: build test vet image deploy-kind

build:
	$(MAKE) -C shared/methodingress-controller build image

test:
	$(MAKE) -C shared/methodingress-controller test vet

vet:
	$(MAKE) -C shared/methodingress-controller vet

image:
	$(MAKE) -C shared/methodingress-controller image

deploy-kind:
	bash .github/scripts/deploy.sh
