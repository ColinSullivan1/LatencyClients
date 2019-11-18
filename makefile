TAG := 0.0.1-alpha1
ACCT := csullivan9999
REQIMAGE := demo-service-requestor
REPIMAGE := demo-service-replier
 
build:
	go build service-replier.go
	go build service-requestor.go

clean:
	rm service-replier
	rm service-requestor

images:
	$(info Make: Building "$(TAG)" tagged images.)
	@docker build -f Dockerfile.requestor -t $(ACCT)/$(REQIMAGE):$(TAG) .
	@docker tag $(ACCT)/$(REQIMAGE):$(TAG) $(ACCT)/$(REQIMAGE):latest
	@docker build -f Dockerfile.replier -t $(ACCT)/$(REPIMAGE):$(TAG) .
	@docker tag $(ACCT)/$(REPIMAGE):$(TAG) $(ACCT)/$(REPIMAGE):latest

push:
	$(info Make: Pushing "$(TAG)" tagged image.)
	@docker push $(ACCT)/$(REQIMAGE):$(TAG)
	@docker push $(ACCT)/$(REQIMAGE):latest
	@docker push $(ACCT)/$(REPIMAGE):$(TAG)
	@docker push $(ACCT)/$(REPIMAGE):latest
