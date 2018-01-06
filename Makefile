default: buildrelease

buildgo:
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix "static" .

builddocker:
	docker build -t us.gcr.io/prp-k8s/oidc-auth:latest .

pushdocker:
	gcloud docker -- push us.gcr.io/prp-k8s/oidc-auth

pushdevdocker:
	gcloud docker -- push us.gcr.io/prp-k8s/oidc-auth-dev

cleanup:
	rm k8s_portal

buildrelease: buildgo builddocker pushdocker cleanup

builddevdocker:
	docker build -t us.gcr.io/prp-k8s/oidc-auth-dev:latest -f Dockerfile_dev .

builddevrelease: buildgo builddevdocker pushdevdocker cleanup

restartpod:
	kubectl delete pods --selector=k8s-app=oidc-auth -n kube-system

pushconfig:
	-kubectl delete configmap portal-config -n kube-system
	kubectl create configmap portal-config --from-file=config.toml=config_minikube.toml -n kube-system
