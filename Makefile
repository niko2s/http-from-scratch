image:
	docker build -f ./build/package/Dockerfile -t myhttpserver .

kind-load: image
	kind load docker-image myhttpserver

deploy-local: kind-load
	kubectl apply -f deployments/app/

kind-delete:
	kubectl delete -f deployments/app/


redeploy: kind-load
	kubectl rollout restart deployment/my-http-server
	kubectl rollout status deployment/my-http-server

