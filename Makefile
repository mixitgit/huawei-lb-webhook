IMG=mixitd/huawei-lb-wh
TAG=0.2.11

all: release deploy

deploy:
	@kubectl apply -f k8s

release: 
	@docker build -t $(IMG) .
	@docker tag $(IMG):latest $(IMG):$(TAG)
	@docker push $(IMG):latest
	@docker push $(IMG):$(TAG)
	@sed -i '' "s|image:.*|image: $(IMG):$(TAG)|" k8s/deployment.yaml

